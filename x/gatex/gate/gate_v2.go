package gate

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/ugorji/go/codec"
	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/client"
	"github.com/nghichtu91/platform/share/planx/servers/chat"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/servers/gate"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
	"github.com/nghichtu91/platform/share/x/gatex/config"
	"github.com/nghichtu91/platform/share/x/gatex/status"
	googleproto "google.golang.org/protobuf/proto"
)

const (
	defaultPingTime = 15
)

var (
	ErrMsgSize  = errors.New("msg size error")
	ErrReadSize = errors.New("msg read error")
)

var (
	pmo = new(googleproto.MarshalOptions)
)

// Transfer 整合之前gate的server和agent，直接传递数据
type Transfer struct {
	Srv       gate.GameServer
	Agent     *client.PacketConnAgent
	gs        *GateServer
	ch        chan *pb.ClientPacket // 用于特殊需求内部传输
	sessionID string
}

// handleConnV2
// 建立client <-> transfer <-> gamex 连接
func (g *GateServer) handleConnV2(con net.Conn) {
	var AccountID, IpAddr string

	IpAddr, _, _ = net.SplitHostPort(con.RemoteAddr().String())

	defer tilogs.PanicCatcher("handleConnV2 panic, ip:%s", IpAddr)

	tilogs.L().Debugf("<Gate> gate.handleConnection la:%s ra:%s", con.LocalAddr().String(), con.RemoteAddr().String())

	trans := new(Transfer)
	trans.gs = g
	trans.ch = make(chan *pb.ClientPacket, 16)
	trans.Agent = client.NewPacketConnAgent(AccountID, con)

	defer func() {
		trans.Agent.Stop()
		g.server.ReleaseConnChan(con)
		con.Close()
		// 删除chat redis mapping
		tilogs.L().Infof("handleConnection close, key:%s", AccountID)
		chat.DelPlayerToken(g.chatDb, AccountID)
	}()

	// 握手处理
	ok, ln, gzipLimit, sessionId, clientInfo := g.handShake(con, trans.Agent)
	if ok {
		AccountID = ln.String()
		trans.Agent.AccountId = AccountID
		trans.Agent.PacketConn.AccountId = AccountID
		a, _ := db.ParseAccount(AccountID)
		trans.Agent.Sid = a.ShardId

		// 避免机器人影响ccu，但是压测服保留
		channel := consts.RobotChannel
		if !trans.Agent.IsRobot || planx.IsRunPrefTest(config.RunMode) {
			channel = ""
			allmetrics.AddCCU()
			defer allmetrics.ReduceCCU()
		}
		_log := tilogs.L().WithUser(AccountID).With("sessionid", sessionId).With("clientip", con.RemoteAddr().String())

		_log.Infof("<Gate> handleConnV2 handShake success, LocalAddr:%s", con.LocalAddr().String())

		// 通知login服务器我上线
		client.LogSession(AccountID, "online")
		status.NotifyLogInOff(config.LoginStatus{
			LoginToken: ln.LoginToken,
			AccountID:  AccountID,
			LogInOff:   true,
		})
		loginInfo := g.getShardLoginInfo(ln.Account.ShardId)
		if loginInfo == nil {
			_log.Errorf("<Gate> handleConnV2 getShardLoginInfo nil, sid %d", ln.Account.ShardId)
			return
		}

		// 清理掉线玩家的信息
		defer func() {
			loginInfo.activeSessionCache.Delete(strconv.Itoa(int(sessionId)))
			loginInfo.mu.Lock()
			if info, ok := loginInfo.sessionMap[AccountID]; ok {
				if info.id == trans.Agent.SessionId {
					delete(loginInfo.sessionMap, AccountID)
					_log.Debugf("<Gate> handleConnV2 delete from sessionMap and activeSessionCache")
				}
			}
			sessionSize := len(loginInfo.sessionMap)
			loginInfo.mu.Unlock()
			_log.Infof("<Gate> handleConnV2, update gate count metrics, sid:%d, sessionSize:%d. del session",
				loginInfo.sid, sessionSize)
		}()

		// 通知Login服务器我下线
		defer func() {
			client.LogSession(AccountID, "offline")
			status.NotifyLogInOff(config.LoginStatus{
				LoginToken: ln.LoginToken,
				AccountID:  AccountID,
				LogInOff:   false,
			})
		}()

		// 单服维护中
		if loginInfo.shardMaintaining {
			_log.Errorf("<Gate> handleConnV2 getShardLoginInfo, sid %d maintaining", ln.Account.ShardId)
			return
		}

		trans.Srv = g.gameSrvs.NewGameServer(AccountID, sessionId, clientInfo, channel)
		if trans.Srv == nil {
			_log.Errorf("<Gate> handleConnV2, NewGameServer fail")
			return
		}
		defer g.gameSrvs.RecycleGameServer(AccountID, sessionId, trans.Srv)
		trans.sessionID = ln.LoginToken

		// 握手
		handShakePkt := util.PacketPool.Get().(*pb.Packet)
		handShakePkt.PacketId = int32(client.PacketIDGateSession)
		handShakePkt.RawData = handShakePkt.RawData[:0]

		// var out []byte
		enc := codec.NewEncoderBytes(&handShakePkt.RawData, &mh)
		enc.Encode(AccountID)
		enc.Encode(strconv.Itoa(int(gzipLimit)))
		// handShakePkt := client.NewPacket(out, client.PacketIDGateSession)

		if !trans.Srv.SendPacket(handShakePkt) {
			_log.Errorf("<GateAccountClose> handshake failed")
			return
		}

		// 这里开始和v1有较大区别
		// 新开goroutine处理 client -> transfer -> gamex
		// 逻辑本身处理 client <- transfer <- gamex

		var wg util.WaitGroupWrapper
		wg.Wrap(trans.C2S)
		trans.S2C()

		// 下面是原来的逻辑
		// var wg util.WaitGroupWrapper
		// backendQuit := make(chan struct{})
		//
		// wg.Wrap(func() {
		// 	trans.Agent.Start()
		// 	tilogs.L().Debugf("<GateAccountClose> 2/4 close %s", AccountID)
		// })
		// wg.Wrap(func() {
		// 	g.forwardToGameServer(ln.LoginToken, AccountID, trans.Agent, trans.Srv)
		// 	defer close(backendQuit) //TODO 实现高级断线重连就从这里切入，这里是临时实现
		// 	trans.Agent.Stop()
		// 	tilogs.L().Debugf("<GateAccountClose> 3/4 close %s", AccountID)
		// })
		// wg.Wrap(func() {
		// 	g.forwardToClients(backendQuit, AccountID, trans.Srv, trans.Agent)
		// 	trans.Agent.Stop()
		// 	tilogs.L().Debugf("<GateAccountClose> 4/4 close %s", AccountID)
		// 	// TODO gate game 分布式模式.如果gs比agent提前出现问题，这里就退出了，然后其他人都卡死.
		// 	// gs重启，也不会有机会恢复这里
		// 	// allinone 模式下，gs同样可能出问题，但是已经处理过相关退出了 201510.29
		// })
		wg.Wait()
		_log.Debugf("<GateAccountClose> 1/4 handleConnection")
	} else if sessionId > 0 {
		_log := tilogs.L().WithUser(AccountID).With("sessionid", sessionId).With("clientip", con.RemoteAddr().String())

		loginInfo := g.getShardLoginInfo(ln.Account.ShardId)
		if loginInfo == nil {
			_log.Errorf("<Gate> handleConnV2 getShardLoginInfo nil, sid %d", ln.Account.ShardId)
			return
		}
		loginInfo.activeSessionCache.Delete(strconv.Itoa(int(sessionId)))
		loginInfo.mu.Lock()
		if info, ok := loginInfo.sessionMap[AccountID]; ok {
			if info.id == trans.Agent.SessionId {
				delete(loginInfo.sessionMap, AccountID)
				_log.Debugf("<Gate> handleConnV2 delete from sessionMap and activeSessionCache")
			}
		}
		sessionSize := len(loginInfo.sessionMap)
		loginInfo.mu.Unlock()
		_log.Infof("<Gate> handleConnV2 false close conn, update gate count metrics, sid:%d, sessionSize:%d. del session",
			loginInfo.sid, sessionSize)
	}
	// 结束链接
}

// S2C server message to client
func (t *Transfer) S2C() {
	pktChan := t.Srv.GetReadingChan()

	// pingPkt := client.NewPingPacket(t.Agent.Sid)
	// t.Agent.Send(pingPkt)
	defer tilogs.PanicCatcher(t.Agent.AccountId)
	defer func() {
		close(t.Agent.SendErrCloseChan) // 通知reading主loop退出
		tilogs.L().Debugf("[PacketConnAgent] [%s] sending routine exit, la:%s ra:%s",
			t.Agent.AccountId, t.Agent.Conn.LocalAddr(), t.Agent.Conn.RemoteAddr())
	}()

	for {
		select {
		case <-t.Agent.ReadQuitChan:
			return
		case <-t.Srv.GetGoneChan():
			return
		// case <-secondsTimer.After(defaultPingTime * time.Second):
		//	// 周期性发PING呼叫客户端，可以主动发现客户端掉线问题
		//	client.UpdatePingPacket(t.Agent.Sid, pingPkt)
		//	if ok := t.Agent.Send(pingPkt); !ok {
		//		return
		//	}
		case pkt := <-t.ch: // 特殊通道
			if ok := t.Agent.Send(pkt); !ok {
				return
			}
		case pkt, ok := <-t.Agent.AsyncSendChan:
			if !ok || pkt == nil {
				return
			}
			atomic.StoreInt64(&t.Agent.LastActionTime, time.Now().UnixNano())
			if ok := t.Agent.Send(pkt); !ok {
				return
			}
		case pkt, ok := <-pktChan:
			if !ok || pkt == nil {
				return
			}
			switch pkt.GetPacketId() {
			case int32(client.PacketIDPingPong):
				if string(pkt.GetRawData()) == "PING" {
					tilogs.L().Warnf("[GateServer] forwardToClients get PING PONG!")
				}
			case int32(client.PacketIDGatePkt):
				sessionPkt := util.SessionPacketPool.Get().(*pb.SessionPacket)
				sessionPkt.Reset()
				err := proto.Unmarshal(pkt.GetRawData(), sessionPkt)
				if err != nil {
					tilogs.L().Errorf("proto marshal error %v", err)
					return
				}

				allmetrics.CountAddNewSend()
				clientPkt := sessionPkt.ClientPacket
				util.SessionPacketPool.Put(sessionPkt)
				// MessageG2CInitFubenInfo = 30003;
				// MessageG2CBattleFrameDatas = 50002;
				// MessageG2CNewEnterFighter= 30021
				MessageId := clientPkt.GetMessageId()
				if MessageId == 30021 {
					tilogs.L().Infof("gate_send_msg accountId:%v messageId:%v", t.Agent.AccountId, clientPkt.GetMessageId())
				}
				atomic.StoreInt64(&t.Agent.LastActionTime, time.Now().UnixNano())
				if ok := t.Agent.Send(clientPkt); !ok {
					return
				}
			}
		}
	}
}

// C2S client message to server
func (t *Transfer) C2S() {
	var (
		err error
	)

	defer func() {
		// close(t.Agent.ReadChan)
		close(t.Agent.ReadQuitChan)
		tilogs.L().Debugf("[PacketConnAgent] [%s] reading routine exit, la:%s ra:%s",
			t.Agent.AccountId, t.Agent.Conn.LocalAddr(), t.Agent.Conn.RemoteAddr())
	}()

	pingPkt := client.NewPingPacket(t.Agent.Sid)
	// 后续全部使用这个packet接收客户端消息
	pkt := &pb.ClientPacket{
		PacketId:  proto.Int32(0),
		MessageId: proto.Uint32(0),
		Compress:  proto.Bool(false),
		RawData:   make([]byte, 4096),
	}

	for {
		select {
		case <-t.Agent.SendErrCloseChan:
			return
		case <-secondsTimer.After(chatExpireTickTime):
			tilogs.L().Infof("update chat expire mapping, key:%s", t.Agent.AccountId)
			if err := chat.ExpirePlayerToken(t.gs.chatDb, t.Agent.AccountId, int32(chatExpireTime/time.Second)); err != nil {
				chat.DelPlayerToken(t.gs.chatDb, t.Agent.AccountId)
				tilogs.L().Infof("update chat expire mapping failed, del mapping, key:%s", t.Agent.AccountId)
			}
		default:
		}

		// 从net读
		t.Agent.PacketConn.SetReadDeadline(time.Now().Add(planx.GateClientTimeOut))
		err = t.Agent.Read2Packet(pkt)
		if err != nil {
			e, ok := err.(net.Error)
			if ok && e.Temporary() {
				if e.Timeout() {
					tilogs.L().Infof("[PacketConnAgent] read timeout, quit. [%s], %s, err %v",
						t.Agent.AccountId, t.Agent.Conn.LocalAddr(), t.Agent.Conn.RemoteAddr())
					return
				}
				continue
			} else {
				if strings.Contains(err.Error(), ErrMsgSize.Error()) {
					tilogs.L().Errorf("[PacketConnAgent] [%s] Msg Size Error, %s, %v",
						t.Agent.AccountId, t.Agent.Conn.RemoteAddr(), err.Error())
				}
				return
			}
		}
		t.Agent.PacketConn.SetReadDeadline(time.Time{})

		// 处理packet
		switch pkt.GetPacketId() {
		case int32(client.PacketIDPingPong):
			select {
			case t.ch <- pingPkt: // 这个包发回客户端，不往gamex发
				// tilogs.L().Debugf("rev ping, %s", t.Agent.AccountId)
			default:
				tilogs.L().Errorf("failed to send ping packet, %s", t.Agent.AccountId)
				return
			}
		case int32(client.PacketChatToken):
			origin, encToken := chat.GenerateChatToken(time.Now().Unix())
			chatErr := chat.SetPlayerToken(t.gs.chatDb, t.Agent.AccountId, origin)
			if chatErr != nil {
				tilogs.L().Warnf("chat redis write data failed, AccountID:%s", t.Agent.AccountId)
				continue
			} else {
				tokenPkt := client.NewChatTokenPacket(encToken)
				tilogs.L().Infof("account:%s, chat token origin:%s, encToken:%s", t.Agent.AccountId, origin, encToken)
				select {
				case t.ch <- tokenPkt: // 这个包发回客户端，不往gamex发
				default:
					tilogs.L().Errorf("failed to send chat token packet, %s", t.Agent.AccountId)
					return
				}
			}
		default:
			packet := util.PacketPool.Get().(*pb.Packet)
			packet.PacketId = int32(client.PacketIDGatePkt)
			packet.RawData = packet.RawData[:0]

			allmetrics.CountAddNewRequest(1)
			sessionPkt := util.SessionPacketPool.Get().(*pb.SessionPacket)
			sessionPkt.Reset()
			sessionPkt.SessionID = t.sessionID
			sessionPkt.AccountID = t.Agent.AccountId
			sessionPkt.ClientPacket = pkt
			// sessionBytes, err := proto.Marshal(sessionPkt)
			packet.RawData, err = pmo.MarshalAppend(packet.RawData, sessionPkt)
			sessionPkt.ClientPacket = nil // 需要放回pool前解除sessionPkt和pkt的关系
			util.SessionPacketPool.Put(sessionPkt)

			if err != nil {
				tilogs.L().Errorf("marshal session packet error %v, %s", err, t.Agent.AccountId)
				return
			}

			// pkt := &pb.Packet{
			// 	PacketId: int32(client.PacketIDGatePkt),
			// 	RawData:  sessionBytes,
			// }

			if ok := t.Srv.SendPacket(packet); !ok {
				return
			}
		}
	}
}
