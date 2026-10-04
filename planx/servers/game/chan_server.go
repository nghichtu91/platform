package game

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/siddontang/go/timingwheel"
	"github.com/ugorji/go/codec"

	"github.com/nghichtu91/platform/share/planx/client"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
)

const compressSize = 1024 // 下行数据消息大于此大小就开始压缩，此值和客户端一致，若要改需要和客户端一起改

var (
	secondsTimer = timingwheel.NewTimingWheel(time.Second, 60*10)

	mh codec.MsgpackHandle
)

func init() {
	mh.RawToString = true
}

type ChanAgent struct {
	Name string

	In  chan *pb.Packet
	Out chan *pb.Packet

	closeFlag int32

	done      chan struct{}
	waitGroup util.WaitGroupWrapper
}

func NewChanAgent(name string) *ChanAgent {
	agent := &ChanAgent{
		Name: name,

		In:  make(chan *pb.Packet, 64),
		Out: make(chan *pb.Packet, 64),

		done: make(chan struct{}),
	}
	return agent
}

func (agent *ChanAgent) IsClosed() bool { return atomic.LoadInt32(&agent.closeFlag) != 0 }

// GetReadingChan of ChanAgent, caller should be a game ONLY.
func (agent *ChanAgent) GetReadingChan() <-chan *pb.Packet {
	return agent.In
}

func GetSecondsAfterTimer(dur time.Duration) <-chan struct{} {
	return secondsTimer.After(dur)
}

// SendPacket of ChanAgent, caller should be a game ONLY.
func (agent *ChanAgent) SendPacket(pkt *pb.Packet) bool {
	if agent.IsClosed() {
		return false
	}
	select {
	case agent.Out <- pkt:
	case <-agent.GetGoneChan():
		return false
	case <-GetSecondsAfterTimer(10 * time.Second):
		tilogs.L().Errorf("ChanAgent SendPacket timeout, "+
			"agent.Out len %d, acid %v，pkgID %v", len(agent.Out), agent.Name, pkt.GetPacketId())
		return false
	}

	return true
}

func (agent *ChanAgent) GetGoneChan() <-chan struct{} {
	return agent.done
}

// AllinOne模式特有：
// 因为ChanAgent是GameServer的代理，模拟tcp模式下的使用
// 因此同一个实例的ChanAgent在Gate中被当成相反的读写模式
// GameServer(Write) --> ChanAgent.Out --> Gate(GetReadingChan)
// GameServer(Read) <-- ChanAgent.In <-- Gate(SendPacket)
// 因此不得不使用独立的chan作为ChanAgent模拟断线的形式。
func (agent *ChanAgent) close() {
	close(agent.done)
}

func (agent *ChanAgent) Stop() {
	if atomic.CompareAndSwapInt32(&agent.closeFlag, 0, 1) {
		close(agent.In)
		agent.waitGroup.Wait()
		close(agent.Out) // 修复race问题，不关闭写入chan
	}
}

// ChanServer gamex在不使用gate的情况下，可以直接使用ChanServer与client进行TCP通讯
// 在极2中目前不需要使用这种模式
type ChanServer struct {
	quit             chan struct{}
	prepareHandshake PreparePlayer
	waitGroup        util.WaitGroupWrapper
	l                sync.RWMutex
}

func NewChanServer() *ChanServer {
	gamesrv := &ChanServer{
		quit: make(chan struct{}),
	}
	return gamesrv
}

func (c *ChanServer) Accept(name string, clientInfo string) *ChanAgent {
	ca := NewChanAgent(name)
	ca.waitGroup.Wrap(func() {
		c.handleConnection(ca, clientInfo)
		tilogs.L().Debugf("<GamexAccountClose> 4/4 ChanServer Agent handleConnection quit. %s", name)
		// ca.Stop()
	})
	// c.waitGroup.Wrap(func() {
	//	c.handleConnection(ca)
	//	tilogs.L().Debugf("ChanServer Agent handleConnection quit.")
	//	//ca.Stop()
	// })

	// XXX 不需要释放ca，因为这里的ca在外层会转变成另外一个角色，会被Stop
	// 所以不需要在handleConnection函数里面处理ca
	return ca
}

func (c *ChanServer) handleConnection(agent *ChanAgent, clientInfo string) {
	defer tilogs.PanicCatcher("ChanServer handleConnection")

	incoming := agent.GetReadingChan()
	var accountid, gziplimit string
	var gzipLimit uint64
	gzipLimit = 0
	select {
	case <-c.quit:
		return
	case pkt := <-incoming:
		if pkt.GetPacketId() != int32(client.PacketIDGateSession) {
			tilogs.L().Errorf("[GameCHANSerever] should get PacketIDGateSession firstly. name:%s, PacketId:%d, RawDataLen:%d",
				agent.Name, pkt.GetPacketId(), len(pkt.GetRawData()))
			return
		}

		dec := codec.NewDecoderBytes(pkt.GetRawData(), &mh)
		dec.Decode(&accountid)
		dec.Decode(&gziplimit)
		gzipLimit = compressSize
		// gl, err2 := strconv.ParseUint(gziplimit, 10, 64)
		// if err2 != nil {
		//	tilogs.L().Debugf("playerProcessor allow gzip failed with %s", gziplimit)
		//	gzipLimit = 0
		// } else {
		//	gzipLimit = gl
		//	tilogs.L().Debugf("playerProcessor allow gzip with %d", gl)
		// }
		tilogs.L().Debugf("[GameCHANServer] handleConnection with account %s", accountid)
	}

	v, ok := isDBStatusReadyAndMark(accountid)
	if !ok {
		if v != nil {
			c := v.Load()
			ss, ok := c.(*PlayerState)
			if ok {
				tilogs.L().Warnf("Account is not ready for login. account:%s laststate %v",
					accountid, ss.string())
			} else {
				tilogs.L().Warnf("Account is not ready for login. account:%s  %v",
					accountid, c)
			}
		} else {
			tilogs.L().Warnf("Account is not ready for login. account:%s",
				accountid)
		}

		// 见枚举 enum.proto ErrCode
		dbErrCode := consts.Kick_Code_Not_Ready_For_Login
		sendErrorNotify(accountid, agent.SendPacket, fmt.Errorf("Unable to login"), int32(dbErrCode))
		return
	}

	defer markDBStatusRelease(accountid)

	c.l.RLock()
	fPrepareHandshake := c.prepareHandshake
	c.l.RUnlock()
	playerRes, ok := func() (Player2, bool) {
		defer tilogs.PanicCatcher(accountid)
		return fPrepareHandshake(accountid, clientInfo)
	}()
	if !ok {
		return
	}
	defer func() {
		agent.close()
		tilogs.L().Debugf("[GameCHANServer] handleConnection account %s, exited in game part", accountid)
	}()

	if !playerRes.Online(clientInfo) {
		tilogs.L().Warnf("[GameCHANServer] handleConnection player is not online [%s]", accountid)
		return
	}
	quit := playerProcessor2(c.quit, playerRes, incoming, agent.SendPacket, gzipLimit)
	if quit {
		playerRes.ShutDownWhenOffline()
	}
	playerRes.Offline()
}

func (c *ChanServer) Stop() {
	close(c.quit)
	tilogs.L().Infof("ChanServer close quit")
}

func (c *ChanServer) Start(pp PreparePlayer) {
	if pp != nil {
		c.l.Lock()
		c.prepareHandshake = pp
		c.l.Unlock()
	} else {
		panic("[GameCHANSerever] Should start with PreparePlayer function")
	}
	<-c.quit
	c.waitGroup.Wait()
	tilogs.L().Infof("[GameCHANServer] Game Server Chan Mode exit.")
}
