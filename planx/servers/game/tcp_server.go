package game

import (
	"net"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/servers"
	"github.com/nghichtu91/platform/share/planx/util"
)

type TCPServer struct {
	*servers.ConnServer
	quit             chan struct{}
	prepareHandshake PreparePlayer
}

func NewTCPServer(listento string) *TCPServer {
	scfg := servers.NewConnServerCfg{
		ListenTo:         listento,
		NumberOfAcceptor: 1,
	}
	gamesrv := &TCPServer{
		ConnServer: servers.NewConnServer(scfg),
		quit:       make(chan struct{}),
	}
	return gamesrv
}

// TODO
func (c *TCPServer) handleConnection(con net.Conn) {
	//defer tilogs.PanicCatcher("TCPServer handleConnection")
	//
	//defer c.ReleaseConnChan(con)
	//defer con.Close()
	////defer func() {
	////if v := recover(); v != nil {
	////tilogs.L().Errorf("[GameTCPServer] handleConnection recover error %v", v)
	////trace := make([]byte, 1024)
	////count := runtime.Stack(trace, true)
	////tilogs.L().Errorf("[GameTCPServer] Stack of %d bytes: %s\n", count, trace)
	////}
	////}()
	//
	//agent := client.NewPacketConnAgent(con.RemoteAddr().String(), con)
	//go agent.Start(c.quit)
	//tilogs.L().Infof("[GameTCPServer] handleConnection %s", con.RemoteAddr())
	//agentReadingChan := agent.GetReadingChan()
	//
	//var accountid, ip, gziplimit string
	//var gzipLimit uint64
	//select {
	//case <-c.quit:
	//	break
	//case pkt := <-agentReadingChan:
	//	if pkt.GetPacketId() != int32(client.PacketIDGateSession) {
	//		tilogs.L().Errorf("[GameCHANSerever] should get PacketIDGateSession firstly.")
	//		break
	//	}
	//	var mh codec.MsgpackHandle
	//	mh.RawToString = true
	//	//accountid = "0:0:1001"
	//	dec := codec.NewDecoderBytes(pkt.GetRawData(), &mh)
	//	dec.Decode(&accountid)
	//	dec.Decode(&ip)
	//	dec.Decode(&gziplimit)
	//	gl, err2 := strconv.ParseUint(gziplimit, 10, 64)
	//	if err2 != nil {
	//		tilogs.L().Debugf("playerProcessor allow gzip failed with %s", gziplimit)
	//		gzipLimit = 0
	//	} else {
	//		gzipLimit = gl
	//		tilogs.L().Debugf("playerProcessor allow gzip with %d", gl)
	//	}
	//	tilogs.L().Debugf("[GameTCPServer] handleConnection with account %s", accountid)
	//
	//	v, ok := isDBStatusReadyAndMark(accountid)
	//	if !ok {
	//		if v != nil {
	//			c := v.Load()
	//			ss := c.(*PlayerState)
	//			tilogs.L().Warnf("Account is not ready for login. account:%s laststate %v",
	//				accountid, ss.string())
	//		} else {
	//			tilogs.L().Warnf("Account is not ready for login. account:%s",
	//				accountid)
	//		}
	//		sendErrorNotify(agent.SendPacket, fmt.Errorf("Unable to login"))
	//		return
	//	}
	//
	//	playerRes, ok := c.prepareHandshake(accountid)
	//	if !ok {
	//		return
	//	}
	//
	//	if !playerRes.Online("") {
	//		return
	//	}
	//	quit := playerProcessor2(c.quit, playerRes, agentReadingChan, agent.SendPacket, gzipLimit)
	//	if quit {
	//		playerRes.ShutDownWhenOffline()
	//	}
	//	playerRes.Offline()
	//}
	//
	//tilogs.L().Infof("[GameTCPServer] handleConnection %s exit", con.RemoteAddr())
}

func (c *TCPServer) loop() {
	conChan := c.GetWaitingConnChan()
	for {
		select {
		case <-c.quit:
			tilogs.L().Infof("[GameTCPServer] dispatcher, server quit")
			return
		case conn, ok := <-conChan:
			if ok {
				myc := conn
				go c.handleConnection(myc)
			}
		}
	}
}

func (c *TCPServer) Stop() {
	close(c.quit)
}

func (c *TCPServer) Start(pp PreparePlayer) {
	if pp != nil {
		c.prepareHandshake = pp
	} else {
		panic("[GameTCPServer] Should start with PreparePlayer function")
	}
	go c.ConnServer.Start()
	var waitGroup util.WaitGroupWrapper
	waitGroup.Wrap(func() { c.loop() })
	waitGroup.Wait()
	c.ConnServer.Stop()
}
