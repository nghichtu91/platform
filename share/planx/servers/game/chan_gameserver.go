package game

import (
	"time"

	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"

	"github.com/nghichtu91/platform/share/planx/servers/gate"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

// ChanGameServer is a reversed ChanAgent of game server, as gate agent simulation
type ChanGameServer struct {
	*ChanAgent
}

// GetReadingChan of ChanAgent, caller should be a gate ONLY.
func (c *ChanGameServer) GetReadingChan() <-chan *pb.Packet {
	return c.Out
}

// SendPacket of ChanAgent, caller should be a gate ONLY.
func (c *ChanGameServer) SendPacket(pkt *pb.Packet) bool {
	if c.IsClosed() {
		return false
	}
	select {
	case c.In <- pkt:
	case <-c.GetGoneChan():
		return false
	case <-GetSecondsAfterTimer(3 * time.Second):
		tilogs.L().Errorf("ChanGameServer SendPacket timeout")
		return false
	}
	return true
}

// func newChanGameServer(name string) *ChanGameServer {
// agent := game.NewChanAgent(name)
// return &ChanGameServer{agent}
// }

type FForceQuit func(string)

type ChanGameServerManager struct {
	chanServer *ChanServer
	fForceQuit FForceQuit
	wg         util.WaitGroupWrapper
	quit       chan struct{}
}

func NewChanGameServerManager(fGetPlayer PreparePlayer, fForceQuit FForceQuit) *ChanGameServerManager {
	cs := NewChanServer()
	go func() {
		cs.Start(fGetPlayer)
	}()

	return &ChanGameServerManager{
		chanServer: cs,
		fForceQuit: fForceQuit,
		quit:       make(chan struct{}),
	}
}

func (t *ChanGameServerManager) NewGameServer(name string, sessionId int64, clientInfo, channel string) gate.GameServer {
	// TODO 检查是否是断线重连，而且后agent还没关闭
	t.wg.Add(1)

	s := &ChanGameServer{
		t.chanServer.Accept(name, clientInfo),
	}

	tilogs.L().Debugf("New ChanGameServer conn work.")
	return s
}

func (t *ChanGameServerManager) RecycleGameServer(acid string, sessionId int64, gs gate.GameServer) {
	// TODO 暂时回收当前链接，等待玩家是否会断线重连。并尝试数据回写数据库
	// 需要配合后端Game Server算法，玩家断线后Game Server实现数据落地
	tilogs.L().Debugf("Chan RecycleGameServer.")
	gs.Stop()
	t.wg.Done()
	return
}

func (t *ChanGameServerManager) WaitAllShutdown(quit <-chan struct{}) {
	<-quit
	t.chanServer.Stop()
	close(t.quit)
	t.wg.Wait()
	return
}
