package resetresp

import (
	"github.com/golang/protobuf/proto"
	gp "google.golang.org/protobuf/proto"

	log "github.com/nghichtu91/platform/share/planx/tilogs"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

var (
	pmo = new(gp.MarshalOptions)
)

func ResetChatResp(p *pb.ClientPackage, resp *pb.ChatMsgResp, err error) {
	p.MessageId = proto.Uint32(uint32(pb.ChatOperation_S2CChatMsg))
	resp.Msg = nil
	if err == nil {
		resp.Code = pb.ErrorCode_Success.Enum()
	} else {
		errCode, ok := pb.ErrorCode_value[err.Error()]
		// 经过grpc的错误 会被包装一层
		if !ok {
			errCode = int32(pb.ErrorCode_SomethingError)
			log.L().Warnf("resetChatResp error code not exist! err:%v", err)
		}
		resp.Code = pb.ErrorCode(errCode).Enum()
	}

	// buf, e := proto.Marshal(resp)
	// if e != nil {
	// 	return
	// }
	// p.RawData = buf

	marshalMsg2ClientPkg(p, resp)
}

func ResetJoinRoomResp(p *pb.ClientPackage, resp *pb.JoinRoomResp, err error) {
	p.MessageId = proto.Uint32(uint32(pb.ChatOperation_S2CJoinRoom))
	if err == nil {
		resp.Code = pb.ErrorCode_Success.Enum()
	} else {
		errCode, ok := pb.ErrorCode_value[err.Error()]
		if !ok {
			errCode = int32(pb.ErrorCode_SomethingError)
			log.L().Warnf("reset join room error code not exist! err:%v", err)
		}
		resp.Code = pb.ErrorCode(errCode).Enum()
	}

	// buf, e := proto.Marshal(resp)
	// if e != nil {
	// 	log.L().Warnf("JoinRoomResp marshal err:%v", e)
	// 	return
	// }
	// p.RawData = buf

	marshalMsg2ClientPkg(p, resp)
}

func ResetLeaveRoomResp(p *pb.ClientPackage, resp *pb.LeaveRoomResp, err error) {
	p.MessageId = proto.Uint32(uint32(pb.ChatOperation_S2CLeaveRoom))
	if err == nil {
		resp.Code = pb.ErrorCode_Success.Enum()
	} else {
		errCode, ok := pb.ErrorCode_value[err.Error()]
		if !ok {
			errCode = int32(pb.ErrorCode_SomethingError)
			log.L().Warnf("reset leave room error code not exist! err:%v", err)
		}
		resp.Code = pb.ErrorCode(errCode).Enum()
	}

	// buf, e := proto.Marshal(resp)
	// if e != nil {
	// 	log.L().Warnf("LeaveRoomResp marshal err:%v", e)
	// 	return
	// }
	// p.RawData = buf

	marshalMsg2ClientPkg(p, resp)
}

// ResetHistoryResp 将ClientPackage内容替换为HistoryMsgResp
func ResetHistoryResp(p *pb.ClientPackage, resp *pb.HistoryMsgResp, err error) {
	p.MessageId = proto.Uint32(uint32(pb.ChatOperation_S2CHistroyMsg))
	if err == nil {
		resp.Code = pb.ErrorCode_Success.Enum()
	} else {
		errCode, ok := pb.ErrorCode_value[err.Error()]
		if !ok {
			errCode = int32(pb.ErrorCode_SomethingError)
			log.L().Warnf("reset history error code not exist! err:%v", err)
		}
		resp.Code = pb.ErrorCode(errCode).Enum()
	}

	// buf, e := proto.Marshal(resp)
	// if e != nil {
	// 	log.L().Warnf("HistoryResp marshal err:%v", e)
	// 	return
	// }
	// p.RawData = buf

	marshalMsg2ClientPkg(p, resp)
}

func marshalMsg2ClientPkg(p *pb.ClientPackage, resp gp.Message) {
	var err error
	p.RawData = p.RawData[:0]
	p.RawData, err = pmo.MarshalAppend(p.RawData, resp)
	if err != nil {
		log.L().Warnf("marshalMsg2ClientPkg id %d err:%v", p.GetRawData(), err)
	}
}
