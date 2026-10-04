package game

import (
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
)

var (
	c = make(chan *pb.BatchPacket, 1024)
)

func GetSendCh() <-chan *pb.BatchPacket {
	return c
}

// SendBatchMsg
// 通过gate再广播
// 之前是在gamex就发给每个acid一个协议
// 现在是发给gatex一条协议，再由gate进行广播
func SendBatchMsg(acids []string, packet *SendPacket, msgCode, clientMsgName string, gid, shardId uint) {
	pkt := genPacket("", packet, 1024) // TODO 先写死，应该从gmtools拿
	SendNMetrics(msgCode, clientMsgName, len(packet.Resp.RawBytes), int32(len(acids)), gid, shardId)
	SendNMetrics(msgCode+"zip", "", len(pkt.RawData), int32(len(acids)), gid, shardId)
	select {
	case c <- &pb.BatchPacket{
		Acids:  acids,
		Packet: pkt,
	}:
	default:
		tilogs.L().Errorf("batch message channel full")
	}
}
