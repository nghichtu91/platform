package game

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/servers"
	"github.com/nghichtu91/platform/share/planx/servers/db"
)

type Player interface {
	//获取玩家唯一帐号ID
	AccountID() db.Account

	//获取玩家能够使用的路由指针
	GetMux() *servers.Mux

	//玩家handleConnection退出时,defer保证调用
	OnExit()

	//来自游戏数据库和逻辑层面的严重问题
	//用于合理化退出和玩家断线前通知客户端，使得客户端有机会配合异常处理
	ErrorNotify() <-chan error

	//逻辑层面其他模块和Gorountine发来的Msg
	MsgChannel() <-chan servers.Request

	//游戏主动推送给客户端的协议
	PushNotify() <-chan INotifySyncMsg

	//从NotifyTag结构建立sync回包
	MkSyncNotifyInfo(m INotifySyncMsg) *servers.Response
}

type INotifySyncMsg interface {
	SetAddr(addr string)
	GetAddr() string
}

type INotifyPushMsg interface {
	INotifySyncMsg
	MakeResponse() *servers.Response
}

type PlayerMsgTyp uint

const (
	_ PlayerMsgTyp = iota
	t_save
	t_msg
	t_push
	t_req
)

type PlayerState struct {
	Typ  PlayerMsgTyp
	Code string
}

func (s *PlayerState) set(t PlayerMsgTyp, code string) {
	s.Typ = t
	s.Code = code
}
func (s *PlayerState) string() string {
	return fmt.Sprintf("{%d %s}", s.Typ, s.Code)
}

// playerProcessor 处理一个玩家的所有请求
// PacketIDReqResp类型一次只处理一个
// TODO: PacketIDContent类型的消息还没有开始设计和使用,
// 从目前的设计看， ReqResp的设计会影响到其他类型的处理速度
// 但是如果拆成两个互不影响的goroutine就要考虑锁的问题。
//func playerProcessor(
//	quit <-chan struct{},
//	pr Player,
//	packetChanR <-chan *client.Packet,
//	sendpkt func(*client.Packet) bool,
//	gzipLimit uint64,
//) bool {
//	prAccountID := pr.AccountID()
//	prAccountIDStr := prAccountID.String()
//	defer tilogs.PanicCatcher(prAccountID.String())
//	mux := pr.GetMux()
//
//	playerState := getDBStatusInUse(prAccountIDStr)
//
//	var mhr codec.MsgpackHandle
//	var mhw codec.MsgpackHandle
//	mhr.MapType = reflect.TypeOf(map[string]interface{}(nil))
//	mhr.RawToString = false
//	mhr.WriteExt = true
//
//	mhw.MapType = reflect.TypeOf(map[string]interface{}(nil))
//	mhw.RawToString = false
//	mhw.WriteExt = true
//
//	timerTicker30 := dbTimer.After(30 * time.Second)
//	for {
//		select {
//		case <-quit:
//			return false
//		case perr, ok := <-pr.ErrorNotify():
//			if ok {
//				sendErrorNotify(sendpkt, perr)
//			}
//			tilogs.L().Errorf("<Gate> [GameServer] playerProcessor got pr.ErrorNotify: %s", perr.Error())
//			return false
//		case <-timerTicker30:
//			playerState.Store(&PlayerState{t_save, ""})
//			timerTicker30 = dbTimer.After(30 * time.Second)
//			mux.Serve(servers.SYSTEM_TICK_30_REQUEST)
//		case msg, ok := <-pr.MsgChannel():
//			if ok {
//				playerState.Store(&PlayerState{t_msg, msg.Code})
//				mux.Serve(msg)
//			}
//		case v, ok := <-pr.PushNotify():
//			rsp := pr.MkSyncNotifyInfo(v)
//			playerState.Store(&PlayerState{t_push, rsp.Code})
//			if ok && rsp != nil && rsp.RawBytes != nil {
//				sendPushNotify(sendpkt, rsp, prAccountID.String(), gzipLimit)
//			}
//		case spkt, ok := <-packetChanR:
//			if !ok {
//				tilogs.L().Infof("<Gate> account playerProcessor quit %s", prAccountIDStr)
//				return false
//			}
//
//			switch spkt.GetContentType() {
//			case client.PacketIDPingPong:
//				tilogs.L().Warnf("[GameServer] forwardToServices should not get pingpong")
//			case client.PacketIDGateSession:
//				// TODO GateSession的实现，用户多路复用
//			case client.PacketIDGatePkt:
//				// 解析SessionPacket
//				tb := time.Now()
//				var sessionpkt client.SessionPacket
//				sdec := codec.NewDecoderBytes(spkt.GetBytes(), &mhr)
//				if err := sdec.Decode(&sessionpkt); err != nil {
//					tilogs.L().Errorf("PacketIDGatePkt Decode err %s", err.Error())
//				}
//				sessionID := sessionpkt.SessionID
//				pkt := sessionpkt.PacketData
//				//FIXME 根据不同的session创建每个玩家自己的goroutine, 这里现在可以拿到AccountID
//
//				// 解析客户端数据包
//				switch pkt.GetContentType() {
//				case client.PacketIDReqResp:
//					var (
//						action   string
//						rawBytes []byte
//						//value  map[string]interface{}
//					)
//					dec := codec.NewDecoderBytes(pkt.GetBytes(), &mhr)
//					dec.Decode(&action)
//					dec.Decode(&rawBytes)
//					//dec.Decode(&value)
//					if len(rawBytes) <= 0 {
//						tilogs.L().Errorf("Client's sent a empty binary!")
//					} else {
//						playerState.Store(&PlayerState{t_req, action})
//						starttime := client.LogRequestStartTime()
//						r := servers.Request{Code: action, RawBytes: rawBytes}
//						v := mux.Serve(r)
//						afterZipLen := len(v.RawBytes)
//						if v != nil && v.RawBytes != nil {
//							var out []byte //XXX: If it comes from []byte buffer pool, it would be cool
//							enc := codec.NewEncoderBytes(&out, &mhw)
//							if gzipLimit > 0 && uint64(len(v.RawBytes)) > gzipLimit { //gzip compress is needed
//								var b bytes.Buffer
//								w := gzip.NewWriter(&b)
//								w.Write(v.RawBytes)
//								w.Close()
//								enc.Encode("/gzip/" + v.Code)
//								enc.Encode(b.Bytes())
//								afterZipLen = len(b.Bytes())
//							} else {
//								enc.Encode(v.Code) //TODO: 返回值应该使用不同的类型，所以需要考虑如何传递过来
//								enc.Encode(v.RawBytes)
//							}
//							//tilogs.L().Debugf("********, %v", out)
//
//							pkt := client.NewPacket(out, client.PacketIDReqResp)
//							var outsession []byte //XXX: If it comes from []byte buffer pool, it would be cool
//							encsess := codec.NewEncoderBytes(&outsession, &mhw)
//
//							encsess.Encode(
//								client.SessionPacket{
//									SessionID:  sessionID,
//									AccountID:  prAccountID.String(),
//									PacketData: pkt})
//
//							npkt := client.NewPacket(outsession, client.PacketIDGatePkt)
//							sendpkt(npkt)
//						}
//						if v != nil {
//							client.LogResponseTime(action, v.Code, starttime)
//						}
//						te := time.Now()
//						tilogs.L().Debugf("playerProcessor_req %s acid %s %v %v %v， size  before gzip = %d,  after gzip = %d",
//							action, prAccountID.String(),
//							tb, te, te.Sub(tb), len(v.RawBytes), afterZipLen)
//					}
//
//				case client.PacketIDContent:
//					tilogs.L().Infof("TODO: PacketIDContent %v", pkt)
//				}
//			}
//		}
//	}
//}

//func sendPushNotify(sendpkt func(*client.Packet) bool, push *servers.Response, acid string, gzipLimit uint64) {
//	var mhw codec.MsgpackHandle
//	mhw.MapType = reflect.TypeOf(map[string]interface{}(nil))
//	mhw.RawToString = false
//	mhw.WriteExt = true
//
//	var out []byte //XXX: If it comes from []byte buffer pool, it would be cool
//	enc := codec.NewEncoderBytes(&out, &mhw)
//	if gzipLimit > 0 && uint64(len(push.RawBytes)) > gzipLimit { //gzip compress is needed
//		var b bytes.Buffer
//		w := gzip.NewWriter(&b)
//		w.Write(push.RawBytes)
//		w.Close()
//		enc.Encode("/gzip/" + push.Code)
//		enc.Encode(b.Bytes())
//		tilogs.L().Debugf("push gzip")
//	} else {
//		enc.Encode(push.Code) //TODO: 返回值应该使用不同的类型，所以需要考虑如何传递过来
//		enc.Encode(push.RawBytes)
//	}
//
//	pkt := client.NewPacket(out, client.PacketIDContent)
//	var outsession []byte //XXX: If it comes from []byte buffer pool, it would be cool
//	encsess := codec.NewEncoderBytes(&outsession, &mhw)
//	encsess.Encode(
//		client.SessionPacket{
//			SessionID:  "",
//			AccountID:  acid,
//			PacketData: pkt})
//
//	npkt := client.NewPacket(outsession, client.PacketIDGatePkt)
//	sendpkt(npkt)
//	tilogs.L().Debugf("send push msg %v, size before gzip = %d", push.Code, len(push.RawBytes))
//}
