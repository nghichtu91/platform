package logic

import (
	"github.com/golang/protobuf/proto"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

// 生成返回的聊天协议
func generateChatMsgResp(r *pb.ChatMsgResp) (resp *pb.ClientPackage, err error) {
	var buf []byte
	buf, err = proto.Marshal(r)
	resp = &pb.ClientPackage{
		MessageId: proto.Uint32(uint32(pb.ChatOperation_S2CChatMsg)),
		RawData:   buf,
	}

	return resp, err
}

// 生成返回的历史消息协议
func generateHistoryMsgResp(his []*pb.ChatMsgData, errCode pb.ErrorCode) (resp *pb.ClientPackage, err error) {
	var buf []byte

	r := &pb.HistoryMsgResp{
		Msg:  his,
		Code: &errCode,
	}

	buf, err = proto.Marshal(r)
	resp = &pb.ClientPackage{
		MessageId: proto.Uint32(uint32(pb.ChatOperation_S2CHistroyMsg)),
		RawData:   buf,
	}

	return resp, err
}
