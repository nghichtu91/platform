package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/siddontang/go/timingwheel"
	net2 "github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
)

type PacketConn struct {
	AccountId string
	net.Conn
	ioread  *bufio.Reader
	iowrite *bufio.Writer
}

const defaultBufferSize = 16 * 1024 // by zhangzhen 之前是256*1024
const defaultPingTime = 15          // seconds 改为15，和15s没收到客户端的pingpong就是断开连接的超时一致
var secondsTimer *timingwheel.TimingWheel

func init() {
	secondsTimer = timingwheel.NewTimingWheel(time.Second, 600)
}

var (
	BufWPool sync.Pool
	BufRPool sync.Pool
)

func NewBufioWriter(w io.Writer, size int) *bufio.Writer {
	if v := BufWPool.Get(); v != nil {
		bw := v.(*bufio.Writer)
		bw.Reset(w)
		return bw
	}
	return bufio.NewWriterSize(w, size)
}

func NewBufioReader(r io.Reader, size int) *bufio.Reader {
	if v := BufRPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	return bufio.NewReaderSize(r, size)
}

func newPacketConn(accountID string, connection net.Conn) PacketConn {
	conn := PacketConn{
		AccountId: accountID,
		Conn:      connection,
		// ioread:  bufio.NewReaderSize(connection, defaultBufferSize),
		// iowrite: bufio.NewWriterSize(connection, defaultBufferSize),
		ioread:  NewBufioReader(connection, defaultBufferSize),
		iowrite: NewBufioWriter(connection, defaultBufferSize),
	}
	return conn
}

func (p *PacketConn) setReadDeadline(t time.Time) error {
	return p.Conn.SetReadDeadline(t)
}

func (p *PacketConn) setWriteDeadline(t time.Time) error {
	return p.Conn.SetWriteDeadline(t)
}

func (p *PacketConn) ReadPacket(t time.Duration) (*pb.ClientPacket, error) {
	if p.ioread == nil {
		return nil, fmt.Errorf("PacketConn.ReadPacket conn already closed")
	}
	pkt, err := ReadPacket(p.ioread, func(id int) {
		p.setReadDeadline(time.Now().Add(t * time.Duration(id)))
	})
	if err != nil {
		e, ok := err.(net.Error)
		if ok && e.Temporary() {
			if !e.Timeout() {
				tilogs.L().Warnf("PacketConn.ReadPacket got Temorary error, %s, %s, %v", p.AccountId, p.RemoteAddr(), err)
			} else {
				tilogs.L().Debugf("PacketConn.ReadPacket Timeout, Heartbeat broken, %s, %s, %v", p.AccountId, p.RemoteAddr(), err)
			}
		} else if err == io.EOF || net2.NetIsClosed(err) {
			// tilogs.L().Debugf("PacketConn.ReadPacket Closed, %s", p.RemoteAddr())
		} else {
			tilogs.L().Warnf("PacketConn.ReadPacket Error, %s, %s, %v", p.AccountId, p.RemoteAddr(), err)
		}
		return nil, err
	}
	return pkt, err
}

func (p *PacketConn) Read2Packet(pkt *pb.ClientPacket) error {
	if p.ioread == nil {
		return fmt.Errorf("PacketConn.ReadPacket conn already closed")
	}
	err := Read2Packet(p.ioread, pkt)
	if err != nil {
		e, ok := err.(net.Error)
		if ok && e.Temporary() {
			if !e.Timeout() {
				tilogs.L().Warnf("PacketConn.ReadPacket got Temorary error, %s, %s, %v", p.AccountId, p.RemoteAddr(), err)
			} else {
				tilogs.L().Debugf("PacketConn.ReadPacket Timeout, Heartbeat broken, %s, %s, %v", p.AccountId, p.RemoteAddr(), err)
			}
		} else if err == io.EOF || net2.NetIsClosed(err) {
			// tilogs.L().Debugf("PacketConn.ReadPacket Closed, %s", p.RemoteAddr())
		} else {
			tilogs.L().Warnf("PacketConn.ReadPacket Error, %s, %s, %v", p.AccountId, p.RemoteAddr(), err)
		}
		return err
	}

	return nil
}

func (p *PacketConn) sendPacket(pkt *pb.ClientPacket) error {
	if pkt == nil {
		tilogs.L().Warnf("PacketConn.SendPacket got a nil packet, ignored")
		return nil
	}
	if p.iowrite == nil {
		tilogs.L().Warnf("PacketConn.SendPacket conn already closed")
		return nil
	}
	_, err := SendPacket(p.iowrite, pkt)
	p.iowrite.Flush()
	if err != nil {
		e, ok := err.(net.Error)
		if ok && e.Temporary() {
			if e.Timeout() {
				tilogs.L().Warnf("PacketConn.SendPacket maybe timeout, packet might be lost, %s, %v", p.RemoteAddr(), err)
			} else {
				tilogs.L().Warnf("PacketConn.SendPacket got Temporary error, packet might be lost, %s, %v", p.RemoteAddr(), err)
			}
		} else if err == io.EOF || net2.NetIsClosed(err) {
			// tilogs.L().Debugf("PacketConn.SendPacket client closed, %s", p.RemoteAddr())
		} else {
			tilogs.L().Warnf("PacketConn.SendPacket Error, %s, %v", p.RemoteAddr(), err)
		}
	}
	return err
}

// PacketConnAgent 是链路 gamex<->gatex server<->gatex agent<->client中的agent角色
// 用于和客户端建立tcp连接，并把包数据转发给gatex server
type PacketConnAgent struct {
	PacketConn
	SessionId      int64
	AccountId      string
	Sid            uint
	WriteDeadLine  time.Duration
	IdleTimeSpan   time.Duration
	LastActionTime int64

	// 用于记录是否为机器人，避免CCU等数据被影响
	IsRobot bool

	ReadChan      chan *pb.ClientPacket
	SendChan      chan *pb.ClientPacket
	AsyncSendChan chan *pb.ClientPacket

	SendErrCloseChan chan struct{}
	ReadQuitChan     chan struct{}

	runMode string
	once    sync.Once
}

func NewPacketConnAgent(accountID string, connection net.Conn) *PacketConnAgent {
	pca := newPacketConn(accountID, connection)
	return &PacketConnAgent{
		AccountId:     accountID,
		PacketConn:    pca,
		WriteDeadLine: time.Second * 20,

		LastActionTime: time.Now().UnixNano(),
		IdleTimeSpan:   time.Second * 5,

		ReadChan:      make(chan *pb.ClientPacket, 64),
		SendChan:      make(chan *pb.ClientPacket, 64),
		AsyncSendChan: make(chan *pb.ClientPacket, 64),

		SendErrCloseChan: make(chan struct{}),
		ReadQuitChan:     make(chan struct{}),
	}
}

func (agent *PacketConnAgent) Start() {
	// quit := make(chan struct{})

	var wg util.WaitGroupWrapper

	wg.Wrap(func() { agent.reading() })
	agent.sending()
	agent.Close()

	// sendingChan不需要给关闭，没有goroutine依赖它退出，它会被gc回收
	// close(agent.SendChan)
	wg.Wait()
	// close(agent.gone)
	tilogs.L().Debugf("PacketConnAgent down")
}

// Stop is the same as Conn.Close()
// It is safe to be called many times as you like
func (agent *PacketConnAgent) Stop() {
	agent.Conn.Close()

	// 需要清理避免内存问题
	agent.once.Do(func() {
		agent.iowrite.Reset(nil)
		BufWPool.Put(agent.iowrite)
		agent.iowrite = nil
		agent.ioread.Reset(nil)
		BufRPool.Put(agent.ioread)
		agent.ioread = nil
	})
}

func (agent *PacketConnAgent) GetReadingChan() <-chan *pb.ClientPacket {
	return agent.ReadChan
}

// func (agent *PacketConnAgent) GetGoneChan() <-chan struct{} {
// 	return agent.gone
// }

func (agent *PacketConnAgent) IsIdle() bool {
	diff := time.Now().UnixNano() - atomic.LoadInt64(&agent.LastActionTime)
	return diff > int64(agent.IdleTimeSpan)
}

// Send SendChan <- pkt with protection
func (agent *PacketConnAgent) SendPacket(pkt *pb.ClientPacket) bool {
	if pkt != nil {
		select {
		case agent.SendChan <- pkt:
		default:
			conn := &agent.PacketConn
			tilogs.L().Warnf("[PacketConnAgent] [%s] SendPacket fulled!, %s", agent.AccountId, conn.RemoteAddr())
			return false
		}
		return true
	}
	return true
}

func (agent *PacketConnAgent) AsyncSendPacket(pkt *pb.ClientPacket) bool {
	if pkt != nil {
		select {
		case agent.AsyncSendChan <- pkt:
		default:
			conn := &agent.PacketConn
			tilogs.L().Warnf("[PacketConnAgent] [%s] AsyncSendPacket fulled!, %s", agent.AccountId, conn.RemoteAddr())
			return false
		}
		return true
	}
	return true
}

func (agent *PacketConnAgent) setNoReadTimeout() {
	agent.PacketConn.SetReadDeadline(time.Time{})
}

func (agent *PacketConnAgent) setNoWriteTimeout() {
	agent.PacketConn.SetWriteDeadline(time.Time{})
}

func (agent *PacketConnAgent) Send(pkt *pb.ClientPacket) bool {
	err := agent.PacketConn.SetWriteDeadline(time.Now().Add(agent.WriteDeadLine))
	if err != nil {
		tilogs.L().Warnf("[PacketConnAgent] [%s] WriteDeadline Error, %s, %v", agent.AccountId, agent.PacketConn.RemoteAddr(), err)
		agent.setNoWriteTimeout()
		return false
	}

	err = agent.sendPacket(pkt)
	// DO NOT USE ELSE HERE
	if err != nil {
		e, ok := err.(net.Error)
		if ok && e.Temporary() {
			if e.Timeout() {
				agent.setNoWriteTimeout()
				return false
			}
		} else {
			agent.setNoWriteTimeout()
			return false
		}
	}
	agent.setNoWriteTimeout()
	return true
}

func (agent *PacketConnAgent) sending() {
	conn := &agent.PacketConn
	// timerdur := defaultPingTime * time.Second
	// timeC := time.NewTicker(defaultPingTime * time.Second)
	pingPkt := NewPingPacket(agent.Sid)

	send := func(pkt *pb.ClientPacket) bool {
		err := conn.SetWriteDeadline(time.Now().Add(agent.WriteDeadLine))
		if err != nil {
			tilogs.L().Warnf("[PacketConnAgent] [%s] WriteDeadline Error, %s, %v", agent.AccountId, conn.RemoteAddr(), err)
			conn.SetWriteDeadline(time.Time{})
			return false
		}

		err = agent.sendPacket(pkt)
		// DO NOT USE ELSE HERE
		if err != nil {
			e, ok := err.(net.Error)
			if ok && e.Temporary() {
				if e.Timeout() {
					// tilogs.L().Errorf("[PacketConnAgent] [%s] Write TCP maybe timeout, packet might be lost, %s, %v", agent.Name, conn.RemoteAddr(), err)
					conn.SetWriteDeadline(time.Time{})
					return false
				} else {
					// tilogs.L().Errorf("[PacketConnAgent] [%s] Write TCP got Temporary error, packet might be lost, %s, %v", agent.Name, conn.RemoteAddr(), err)
				}
			} else if err == io.EOF {
				// tilogs.L().Infof("[PacketConnAgent] [%s] Write TCP client closed, %s", agent.Name, conn.RemoteAddr())
				conn.SetWriteDeadline(time.Time{})
				return false
			} else {
				// tilogs.L().Errorf("[PacketConnAgent] [%s] Write TCP Error, %s, %v, %t", agent.Name, conn.RemoteAddr(), err, err)
				conn.SetWriteDeadline(time.Time{})
				return false
			}
		}
		conn.SetWriteDeadline(time.Time{})
		return true
	}

	// 客户端建立连接后，立刻给客户端发个ping协议，主要为先同步下服务器时间
	send(pingPkt)
Loop:
	for {
		select {
		case <-agent.ReadQuitChan:
			break Loop
		// case <-Quiting:
		// 	break Loop
		case <-secondsTimer.After(defaultPingTime * time.Second):
			// 周期性发PING呼叫客户端，可以主动发现客户端掉线问题
			UpdatePingPacket(agent.Sid, pingPkt)
			if ok := send(pingPkt); !ok {
				break Loop
			}
		case pkt := <-agent.SendChan:
			atomic.StoreInt64(&agent.LastActionTime, time.Now().UnixNano())
			if ok := send(pkt); !ok {
				break Loop
			}
		}
	}
	close(agent.SendErrCloseChan) // 通知reading主loop退出
	tilogs.L().Debugf("[PacketConnAgent] [%s] sending routine exit, la:%s ra:%s", agent.AccountId, conn.LocalAddr(), conn.RemoteAddr())
}

func (agent *PacketConnAgent) reading() {

	conn := &agent.PacketConn
Loop:
	for {
		select {
		// case <-Quiting:
		// 	break Loop
		case <-agent.SendErrCloseChan:
			break Loop
		default:
		}

		// err := conn.SetReadDeadline(time.Now().Add(time.Second * 15))
		// if err != nil {
		// tilogs.L().Errorf("[PacketConnAgent] [%s] SetReadDeadline Error, %s, %v", agent.Name, conn.RemoteAddr(), err)
		// break Loop
		// }

		pkt, err := conn.ReadPacket(time.Second)
		if err != nil {
			e, ok := err.(net.Error)
			if ok && e.Temporary() {
				if e.Timeout() {
					tilogs.L().Infof("[PacketConnAgent] read timeout, quit. [%s], %s, err %v",
						agent.AccountId, conn.RemoteAddr(), err.Error())
					break Loop
				}
				continue Loop
			} else {
				if strings.Contains(err.Error(), msg_size_err.Error()) {
					tilogs.L().Errorf("[PacketConnAgent] [%s] Msg Size Error, %s, %v", agent.AccountId, conn.RemoteAddr(), err.Error())
				}
				break Loop
			}
		}
		conn.SetReadDeadline(time.Time{})

		select {
		case agent.ReadChan <- pkt:
			atomic.StoreInt64(&agent.LastActionTime, time.Now().UnixNano())
		case <-time.After(10 * time.Second):
			tilogs.L().Errorf("[PacketConnAgent] [%s] reading ignore a packet, due to chan timeout!, la:%s ra:%s", agent.AccountId, conn.LocalAddr(), conn.RemoteAddr())
		}
	}
	close(agent.ReadChan)
	close(agent.ReadQuitChan)
	tilogs.L().Debugf("[PacketConnAgent] [%s] reading routine exit, la:%s ra:%s", agent.AccountId, conn.LocalAddr(), conn.RemoteAddr())
}
