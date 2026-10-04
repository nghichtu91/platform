package gate

import (
	"github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/client"
	"github.com/nghichtu91/platform/share/planx/servers"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
	protogen "github.com/nghichtu91/platform/share/x/gatex/gate/pb"
)

type SendNotify interface {
	// 被踢原因 // 多少秒后服务器主动断开 // 多少秒内不允许再登录
	SendKickNotify(acid string, sessionId int64, sendpkt func(packet *pb.ClientPacket) bool, reason string, after int, nologin int, kickErrCode int)
}

func init() {
	NotifyInfo = &SendNotifyImpl{}
}

var NotifyInfo SendNotify

type SendNotifyImpl struct {
}

const (
	IDS_RELOGIN  = "IDS_LOGIN_TIPS_KICKNOTIFY"
	IDS_OFFLINE  = "IDS_ERROR_NETWORK_90000"
	IDS_NEWLOGIN = "IDS_KICKNOTIFY_NEWLOGIN"
)

func (s *SendNotifyImpl) SendKickNotify(acid string, sessionId int64, sendpkt func(packet *pb.ClientPacket) bool, reason string, after int, nologin int, kickErrCode int) {
	//sendKickNotify(sendpkt, reason, after, nologin)
	resp := &servers.Response{}
	resp.Code = planx.PushCodeKick
	var err error
	if reason != IDS_RELOGIN && reason != IDS_OFFLINE {
		reason = IDS_RELOGIN
	}
	resp.RawBytes, err = proto.Marshal(&protogen.KickPush{
		Msg:         proto.String(reason),
		After:       proto.Int32(int32(after)),
		NoLogin:     proto.Int32(int32(nologin)),
		KickErrCode: proto.Int32(int32(kickErrCode)),
	})
	_log := tilogs.L().WithUser(acid).With("sessionid", sessionId)
	if err != nil {
		_log.Errorf("<SendKickNotify> %v", err)
		return
	}
	EncodeAndSendPush(sendpkt, resp)
	_log.Infof("SendKickNotify kickErrCode %d", kickErrCode)
}

type InfoNotifyToClient struct {
	GagTime int64 `codec:"g"`
}

func EncodeAndSendPush(sendpkt func(packet *pb.ClientPacket) bool, response *servers.Response) {
	_id := int32(client.PacketIDContent)
	_compress := false
	pkt := &pb.ClientPacket{
		PacketId:  &_id,
		MessageId: &response.Code,
		Compress:  &_compress,
		RawData:   response.RawBytes,
	}
	sendpkt(pkt)
}
