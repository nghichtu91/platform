package gate

import (
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
)

type GameServerManager interface {
	NewGameServer(acid string, sessionId int64, clientInfo, channel string) GameServer
	RecycleGameServer(acid string, sessionId int64, gs GameServer)
	WaitAllShutdown(quit <-chan struct{})
}

type GameServer interface {
	GetReadingChan() <-chan *pb.Packet
	SendPacket(pkt *pb.Packet) bool
	GetGoneChan() <-chan struct{}
	Stop()
}
