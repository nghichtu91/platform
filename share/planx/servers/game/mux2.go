package game

import (
	"github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/client"
	"github.com/nghichtu91/platform/share/planx/servers"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
	protogen "github.com/nghichtu91/platform/share/x/gatex/gate/pb"
)

// var (
// 	mhr codec.MsgpackHandle
// 	mhw codec.MsgpackHandle
// )
//
// func init() {
// 	mhr.MapType = reflect.TypeOf(map[string]interface{}(nil))
// 	mhr.RawToString = false
// 	mhr.WriteExt = true
//
// 	mhw.MapType = reflect.TypeOf(map[string]interface{}(nil))
// 	mhw.RawToString = false
// 	mhw.WriteExt = true
// }

type RecPacket struct {
	SessionID string
	servers.Request
}

type SendPacket struct {
	ID        client.PacketID
	SessionID string
	Resp      *servers.Response
}

type ErrInfo struct {
	Err      error
	KickCode int32
}

type Player2 interface {
	AccountID() db.Account
	GetRecChan() chan servers.Request
	GetSendChan() chan *SendPacket
	GetErrChan() chan ErrInfo
	Online(clientInfo string) bool
	ShutDownWhenOffline()
	Offline()
}

func playerProcessor2(
	quit <-chan struct{},
	pr Player2,
	packetChanR <-chan *pb.Packet,
	sendpkt func(*pb.Packet) bool,
	gzipLimit uint64,
) (isQuit bool) {
	prAccountID := pr.AccountID()
	prAccountIDStr := prAccountID.String()
	defer tilogs.PanicCatcher(prAccountID.String())

	for {
		select {
		case <-quit:
			return true
		case errInfo, ok := <-pr.GetErrChan():
			if ok {
				sendErrorNotify(prAccountIDStr, sendpkt, errInfo.Err, errInfo.KickCode)
			}
			tilogs.L().Errorf("<Gate> [GameServer] playerProcessor got pr.ErrorNotify: %v, acid: %s", errInfo, prAccountIDStr)
			return true
		case pkt, ok := <-packetChanR:
			if !ok {
				tilogs.L().Infof("<Gate> account playerProcessor quit %s", prAccountIDStr)
				return false
			}

			packetId := client.PacketID(pkt.GetPacketId())
			switch packetId {
			case client.PacketIDPingPong:
				tilogs.L().Warnf("[GameServer] forwardToServices should not get pingpong, acid: %s", prAccountIDStr)
			case client.PacketIDGateSession:
				// TODO GateSession的实现，用户多路复用
			case client.PacketIDGatePkt:
				// 解析SessionPacket
				// sessionPkt := &pb.SessionPacket{}
				sessionPkt := util.SessionPacketPool.Get().(*pb.SessionPacket)
				sessionPkt.Reset()
				err := proto.Unmarshal(pkt.RawData, sessionPkt)
				if err != nil {
					tilogs.L().Errorf("[GameServer] unmarshal session packet: %s, acid: %s", err, prAccountIDStr)
					return false
				}
				// sessionID := sessionPkt.SessionID
				// sessionID := "0"
				// tilogs.L().Infof("sessionID is %v", sessionID)
				clientPkt := sessionPkt.ClientPacket
				// FIXME 根据不同的session创建每个玩家自己的goroutine, 这里现在可以拿到AccountID

				// 解析客户端数据包
				switch clientPkt.GetPacketId() {
				case int32(client.PacketIDReqResp), int32(client.PacketIDContent):
					rawData := clientPkt.GetRawData()
					if clientPkt.GetCompress() {
						// _bef := len(rawData)
						rawData = util.UnCompressProto(rawData)
						if rawData == nil {
							tilogs.L().Errorf("[GameServer] uncompressGzip fail, %s", prAccountIDStr)
							return false
						}
						// tilogs.L().Debugf("uncompress bef %d aft %d", _bef, len(rawData))
					}
					r := servers.Request{Code: clientPkt.GetMessageId(), RawBytes: rawData}
					c := pr.GetRecChan()
					select {
					case c <- r:
					default:
						util.SessionPacketPool.Put(sessionPkt)
						tilogs.L().Errorf("<Gate> [GameServer] playerProcessor put to RecChan timeout, when req %d, acid: %s", r.Code, prAccountIDStr)
						return false
					}
				}
				util.SessionPacketPool.Put(sessionPkt)
			}
		case resp := <-pr.GetSendChan():
			Send(prAccountIDStr, resp, gzipLimit, sendpkt)
			PutSendPacket(resp)
		}
	}
}

func Send(accountId string, s *SendPacket, gzipLimit uint64, sendPkt func(*pb.Packet) bool) {
	pkt := genPacket(accountId, s, gzipLimit)
	if pkt != nil {
		sendPkt(pkt)
	}
}

const max_Msg_Size = 1024 * 1024

func genPacket(accountId string, s *SendPacket, gzipLimit uint64) *pb.Packet {
	if s.Resp != nil {
		out := s.Resp.RawBytes
		compress := false
		if len(out) > compressSize && gzipLimit <= 0 {
			tilogs.L().Errorf("msg:%d before compress len:%d > 1024 but not gzip: %d", s.Resp.Code, len(out), gzipLimit)
		}
		if out != nil && gzipLimit > 0 && uint64(len(s.Resp.RawBytes)) > gzipLimit { // gzip compress is needed
			out = util.CompressProto(s.Resp.RawBytes)
			compress = true
		}
		_id := int32(s.ID)
		clientPacket := &pb.ClientPacket{
			PacketId:  &_id,
			MessageId: &s.Resp.Code,
			Compress:  &compress,
			RawData:   out,
		}
		if len(out) >= max_Msg_Size {
			l := tilogs.L().WithUser(accountId)
			l.Errorf("msgCompress:%d >= 1M, len %d gzipLimit:%v", s.Resp.Code, len(out), gzipLimit)
		}
		// if len(out) > 1024 {
		//	tilogs.L().Warnf("send msg > 1024 acid %s len %d", accountId, len(out))
		// }
		sessionPkt := util.SessionPacketPool.Get().(*pb.SessionPacket)
		sessionPkt.Reset()
		sessionPkt.AccountID = accountId
		sessionPkt.SessionID = s.SessionID
		sessionPkt.ClientPacket = clientPacket

		// sessionPkt := &pb.SessionPacket{
		// 	SessionID:    s.SessionID,
		// 	AccountID:    accountId,
		// 	ClientPacket: clientPacket,
		// }

		sessionBytes, err := proto.Marshal(sessionPkt)
		util.SessionPacketPool.Put(sessionPkt)
		if err != nil {
			tilogs.L().Errorf("proto marshal error %v, acid:%s", err, accountId)
			return nil
		}
		pkt := &pb.Packet{
			PacketId: int32(client.PacketIDGatePkt),
			RawData:  sessionBytes,
		}
		return pkt
	}
	return nil
}

func sendErrorNotify(
	accountId string,
	sendpkt func(*pb.Packet) bool,
	errNotify error,
	kickCode int32,
) {
	var reason string // 被踢原因
	var after int     // 多少秒后服务器主动断开
	var nologin int   // 多少秒内不允许再登录

	reason = errNotify.Error()
	after = 3
	nologin = 10

	resp := &servers.Response{}
	resp.Code = planx.PushCodeKick

	var err error
	resp.RawBytes, err = proto.Marshal(&protogen.KickPush{
		Msg:         proto.String("NODISPLAY " + reason),
		After:       proto.Int32(int32(after)),
		NoLogin:     proto.Int32(int32(nologin)),
		KickErrCode: proto.Int32(kickCode),
	})
	if err != nil {
		tilogs.L().Errorf("<sendErrorNotify> %v, acid:%s", err, accountId)
		return
	}

	sendpkt(genPacket(accountId,
		&SendPacket{
			ID:   client.PacketIDContent,
			Resp: resp,
		}, 0))
}
