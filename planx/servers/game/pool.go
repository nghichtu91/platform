package game

import (
	"sync"
)

var (
	sendPacketPool = sync.Pool{
		New: func() interface{} {
			return new(SendPacket)
		}}
)

func GetSendPacket() *SendPacket {
	return sendPacketPool.Get().(*SendPacket)
}

func PutSendPacket(pkt *SendPacket) {
	pkt.ID = 0
	pkt.SessionID = ""
	pkt.Resp = nil
	sendPacketPool.Put(pkt)
}
