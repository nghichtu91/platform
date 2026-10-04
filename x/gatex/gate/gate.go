package gate

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	// "code.google.com/p/go-uuid/uuid"

	// "strings"

	"github.com/ugorji/go/codec"

	"github.com/nghichtu91/platform/share/planx/client"
	"github.com/nghichtu91/platform/share/planx/redispool"
	"github.com/nghichtu91/platform/share/planx/servers/chat"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/servers/gate"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/x/common/allmetrics"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"

	gateconfig "github.com/nghichtu91/platform/share/x/gatex/config"
	"github.com/nghichtu91/platform/share/x/gatex/rpc"
	"github.com/nghichtu91/platform/share/x/gatex/servers"
	"github.com/nghichtu91/platform/share/x/gatex/status"
)

var mh codec.MsgpackHandle

func init() {
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))
	mh.RawToString = true
	mh.WriteExt = true
}

type GateServer struct {
	server    servers.ConServer
	rpcServer *rpc.GateRPC

	quit     chan struct{}
	gameSrvs gate.GameServerManager

	info      map[uint]*loginInfo // sid->loginInfo
	info_lock sync.RWMutex

	waitGroup util.WaitGroupWrapper

	// chat db
	chatDb redispool.IPool
}

const (
	chatExpireTickTime = 25 * time.Minute
	chatExpireTime     = 30 * time.Minute
)

func NewGateServer(l net.Listener) *GateServer {
	var gate GateServer

	gateCfg := &gateconfig.Cfg.GateConfig

	scfg := servers.NewConnServerCfg{
		ListenTo:             gateCfg.Listen,
		NumberOfAcceptor:     gateCfg.NAcceptor,
		NumberOfWaitingQueue: gateCfg.NWaitingConn,
		SslCfg:               gateCfg.SslCfg,
		HostSslCfg:           gateCfg.HostSslCfg,
	}
	switch gateCfg.ConnServer {
	case "LimitConnServer":
		tilogs.L().Infof("Start with LimitConnServer server")
		slcfg := servers.NewLimitConnServerCfg{
			NewConnServerCfg: scfg,
			MaxConn:          gateCfg.MaxConn,
		}
		gate.server = servers.NewLimitConnServer(slcfg, nil)
	default:
		tilogs.L().Infof("Start with default ConnServer server")
		gate.server = servers.NewConnServer(scfg, nil)
	}

	gate.rpcServer = rpc.NewGateRPCServer(l)
	gate.quit = make(chan struct{})
	gate.info = make(map[uint]*loginInfo, 256)

	return &gate
}

func (g *GateServer) getShardLoginInfo(sid uint) *loginInfo {
	g.info_lock.RLock()
	defer g.info_lock.RUnlock()
	return g.info[sid]
}

func (g *GateServer) onAddShard(sid uint) {
	g.info_lock.Lock()
	info, ok := g.info[sid]
	if ok {
		info.clear()
	} else {
		g.info[sid] = newLoginInfo(sid)
	}
	g.info_lock.Unlock()
}

func (g *GateServer) onDelShard(sid uint) {
	g.info_lock.Lock()
	info, ok := g.info[sid]
	if ok {
		info.clear()
	}
	g.info_lock.Unlock()
}

// 断开指定服务器的所有客户端连接
func (g *GateServer) onKickShard(sid uint) {
	g.info_lock.Lock()
	info, ok := g.info[sid]
	if ok {
		info.kickAll()
	}
	g.info_lock.Unlock()
}

func (g *GateServer) forwardToClients(quit chan struct{}, AccountID string, gs gate.GameServer, agent *client.PacketConnAgent) {
	defer tilogs.PanicCatcher(AccountID)
	tilogs.L().Infof("[GateServer] forwardToClients start, account id: %s", AccountID)
	pktChan := gs.GetReadingChan()
Loop:
	for {
		select {
		case <-quit:
			break Loop
		// case <-g.quit:
		// 	break Loop
		// case <-gs.GetGoneChan(): // 这里可以等forwardToServer先关闭，通过quit chan再关闭这个，避免竞争gone chan
		// 	break Loop
		case pkt, ok := <-pktChan:
			if !ok || pkt == nil {
				break Loop
			}
			switch pkt.GetPacketId() {
			case int32(client.PacketIDPingPong):
				if string(pkt.GetRawData()) == "PING" {
					tilogs.L().Warnf("[GateServer] forwardToClients get PING PONG!")
				}
			case int32(client.PacketIDGatePkt):
				// sessionPkt := &pb.SessionPacket{}
				sessionPkt := util.SessionPacketPool.Get().(*pb.SessionPacket)
				sessionPkt.Reset()
				err := proto.Unmarshal(pkt.GetRawData(), sessionPkt)
				if err != nil {
					tilogs.L().Errorf("proto marshal error %v", err)
					break Loop
				}

				// LOG MARKER
				// client.Log("rsp", AccountID, sessionpkt.PacketData)

				allmetrics.CountAddNewSend()
				clientPkt := sessionPkt.ClientPacket
				util.SessionPacketPool.Put(sessionPkt)

				if ok := agent.SendPacket(clientPkt); !ok {
					break Loop
				}

				// if err != nil {
				// if e, ok := err.(net.Error); ok && e.Timeout() {
				// tilogs.L().Warnf("Gate.servive  reply to client failed1! %s", e.Error())
				// continue
				// }
				// tilogs.L().Warnf("Gate.servive  reply to client failed2! %s", err.Error())
				// //XXX 这里考虑回收GameServer然后等待客户端重连
				// break Loop
				// }
			}
		}
	}
	tilogs.L().Debugf("[GateServer] forwardToClients exit, account id: %s", AccountID)

}

func (g *GateServer) forwardToGameServer(SessionID string, AccountID string, agent *client.PacketConnAgent, gs gate.GameServer) {
	defer tilogs.PanicCatcher(AccountID)
	tilogs.L().Debugf("[GateServer] forwardToGameServer start, for account %s", AccountID)
	// g.registerConn(SessionID, agent)
	agentReadingChan := agent.GetReadingChan()

	// 定时更新chat redis数据库
	t := time.NewTicker(chatExpireTickTime)
	defer t.Stop()
Loop:
	for {
		select {
		// case <-g.quit:
		// 	break Loop
		case <-gs.GetGoneChan():
			break Loop
		case <-t.C:
			tilogs.L().Infof("update chat expire mapping, key:%s", AccountID)
			if err := chat.ExpirePlayerToken(g.chatDb, AccountID, int32(chatExpireTime/time.Second)); err != nil {
				chat.DelPlayerToken(g.chatDb, AccountID)
				tilogs.L().Infof("update chat expire mapping failed, del mapping, key:%s", AccountID)
				t.Stop()
			}

		case pkt, ok := <-agentReadingChan:
			if !ok {
				break Loop
			}

			// LOG MARKER
			// client.Log("req", AccountID, pkt)

			switch pkt.GetPacketId() {
			case int32(client.PacketIDPingPong):
				// agent.SendPacket(client.NewPingPacket())
				// 网关服务器会收到客户端的PING信息，目前不做任何处理
			case int32(client.PacketChatToken):
				origin, encToken := chat.GenerateChatToken(time.Now().Unix())
				chatErr := chat.SetPlayerToken(g.chatDb, AccountID, origin)
				if chatErr != nil {
					tilogs.L().Warnf("chat redis write data failed, AccountID:%s", AccountID)
				} else {
					agent.SendPacket(client.NewChatTokenPacket(encToken))
					tilogs.L().Infof("account:%s, chat token origin:%s, encToken:%s", AccountID, origin, encToken)
				}
			default:
				// XXX: Will it be a performance factor? there is Lock inside it
				allmetrics.CountAddNewRequest(1)
				sessionPkt := util.SessionPacketPool.Get().(*pb.SessionPacket)
				sessionPkt.Reset()
				sessionPkt.SessionID = SessionID
				sessionPkt.AccountID = AccountID
				sessionPkt.ClientPacket = pkt
				// sessionPkt := &pb.SessionPacket{SessionID: SessionID, AccountID: AccountID, ClientPacket: pkt}
				sessionBytes, err := proto.Marshal(sessionPkt)
				util.SessionPacketPool.Put(sessionPkt)

				if err != nil {
					tilogs.L().Errorf("marshal session packet error %v", err)
					break Loop
				}

				pkt := &pb.Packet{
					PacketId: int32(client.PacketIDGatePkt),
					RawData:  sessionBytes,
				}
				// XXX should not recreate always
				if ok := gs.SendPacket(pkt); !ok {
					break Loop
				}
			}
		}
	}
	// g.unRegisterConn(SessionID)
	tilogs.L().Debugf("[GateServer] forwardToGameServer end, for account %s", AccountID)
}

func (g *GateServer) chooseGameServer(accountID string) string {
	// TODO 完善GameServer的选择，需要在实现多个Game服务同一个Gate的时候才需要考虑实现
	if len(gateconfig.Cfg.GateConfig.GameServers) == 0 {
		return "allinone"
	}
	return gateconfig.Cfg.GateConfig.GameServers[0]
}

// func (g *GateServer) GetLoginStatusChan() <-chan LoginStatus {
// return g.loginStatusChan
// }

// 建立玩家和后端服务器的链接，后段服务器可以有多个
// 玩家会被分配到固定的服务器上，所有后端服务器功能相同
// 暂时不会按照功能拆分
// 方便高效断线重连接模式的实现
func (g *GateServer) handleConnection(con net.Conn) {
	var AccountID, IpAddr string
	// tilogs.L().Debugf("Gate Server, handleConnection with %s", con.RemoteAddr().String())
	IpAddr, _, _ = net.SplitHostPort(con.RemoteAddr().String())
	// tilogs.L().Debugf("Gate Server, handleConnection start run with ip %s", IpAddr)
	defer tilogs.PanicCatcher(AccountID, IpAddr)

	allmetrics.AddCCU()
	defer func() {
		allmetrics.ReduceCCU()
		g.server.ReleaseConnChan(con)
		con.Close()
		// 删除chat redis mapping
		tilogs.L().Infof("handleConnection close, key:%s", AccountID)
		chat.DelPlayerToken(g.chatDb, AccountID)
	}()

	tilogs.L().Debugf("<Gate> gate.handleConnection la:%s ra:%s", con.LocalAddr().String(), con.RemoteAddr().String())
	agent := client.NewPacketConnAgent(AccountID, con)
	defer agent.Stop()
	if ok, ln, gzipLimit, sessionId, clientInfo := g.handShake(con, agent); ok {
		AccountID = ln.String() // makeAccountID(ln.GameID, ln.ShardID, ln.UserID)
		agent.AccountId = AccountID
		a, _ := db.ParseAccount(AccountID)
		agent.Sid = a.ShardId

		// /////////////////////////////////////
		// 通知login服务器我上线
		client.LogSession(AccountID, "online")
		status.NotifyLogInOff(gateconfig.LoginStatus{
			LoginToken: ln.LoginToken,
			AccountID:  AccountID,
			LogInOff:   true,
		})
		loginInfo := g.getShardLoginInfo(ln.Account.ShardId)
		if loginInfo == nil {
			return
		}
		// 清理掉线玩家的信息
		defer func() {
			// JWS2-14313 被自己顶号
			err := loginInfo.activeSessionCache.Delete(fmt.Sprintf("%d", sessionId))
			if err != nil {
				tilogs.L().WithUser(AccountID).Errorf("<Gate> gate.handleConnection delete session %d failed, err %s", sessionId, err.Error())
			}
			loginInfo.mu.Lock()
			if info, ok := loginInfo.sessionMap[AccountID]; ok {
				if info.id == agent.SessionId {
					delete(loginInfo.sessionMap, AccountID)
					tilogs.L().Debugf("<Gate> gate.handleConnection delete from sessionMap and activeSessionCache, "+
						"accountId %s sessionId %d", AccountID, sessionId)
				}
			}
			sessionSize := len(loginInfo.sessionMap)
			loginInfo.mu.Unlock()
			tilogs.L().Infof("update gate count metrics, sid:%d, %d. del session %d, acid %s",
				loginInfo.sid, sessionSize, sessionId, AccountID)
		}()

		// 通知Login服务器我下线
		defer func() {
			client.LogSession(AccountID, "offline")
			status.NotifyLogInOff(gateconfig.LoginStatus{
				LoginToken: ln.LoginToken,
				AccountID:  AccountID,
				LogInOff:   false,
			})
		}()

		// /////////////////////////////////////

		// addr := g.chooseGameServer(AccountID) // TODO 找到对应的gamex是否启动了，若没有则断开连接
		gs := g.gameSrvs.NewGameServer(AccountID, 0, clientInfo, "")
		if gs == nil {
			return
		}
		defer g.gameSrvs.RecycleGameServer(AccountID, 0, gs)
		// /////////////////////////////////////
		// go func() { //just for debug
		// <-time.After(5 * time.Second)
		// tilogs.L().Warnf("Send out a kick")
		// sendKickNotify(agent)
		// }()

		// /////////////////////////////////////
		// 发送account id到后端建立服务关系
		var out []byte // XXX: If it comes from []byte buffer pool, it would be cool
		enc := codec.NewEncoderBytes(&out, &mh)
		enc.Encode(AccountID)
		enc.Encode(strconv.Itoa(int(gzipLimit)))
		handShakePkt := client.NewPacket(out, client.PacketIDGateSession)
		gs.SendPacket(handShakePkt)

		var wg util.WaitGroupWrapper
		backendQuit := make(chan struct{})

		// agent, gs之间的退出是交叉退出，任何人出现问题，则大家都退出！
		wg.Wrap(func() {
			agent.Start()
			tilogs.L().Debugf("<GateAccountClose> 2/4 close %s", AccountID)
		})
		wg.Wrap(func() {
			g.forwardToGameServer(ln.LoginToken, AccountID, agent, gs)
			defer close(backendQuit) // TODO 实现高级断线重连就从这里切入，这里是临时实现
			agent.Stop()
			tilogs.L().Debugf("<GateAccountClose> 3/4 close %s", AccountID)
		})
		wg.Wrap(func() {
			g.forwardToClients(backendQuit, AccountID, gs, agent)
			agent.Stop()
			tilogs.L().Debugf("<GateAccountClose> 4/4 close %s", AccountID)
			// TODO gate game 分布式模式.如果gs比agent提前出现问题，这里就退出了，然后其他人都卡死.
			// gs重启，也不会有机会恢复这里
			// allinone 模式下，gs同样可能出问题，但是已经处理过相关退出了 201510.29
		})
		wg.Wait()
		tilogs.L().Debugf("<GateAccountClose> 1/4 handleConnection %s", AccountID)
	}
	// 结束链接
}

func (g *GateServer) loop() {
	cc := g.server.GetWaitingConnChan()
	for conn := range cc {
		myconn := conn
		// g.waitGroup.Wrap(func() { g.handleConnection(myconn) })
		g.waitGroup.Wrap(func() { g.handleConnV2(myconn) })
	}
}

func (g *GateServer) Listen() error {
	if err := g.server.Listen(); err != nil {
		return err
	}
	if err := g.rpcServer.Listen(); err != nil {
		return err
	}
	return nil
}

func (g *GateServer) Start(gsm gate.GameServerManager, internalIpPort string) {
	g.gameSrvs = gsm
	go g.server.Start()

	// rpc server of gate
	// _rpc = rpc.NewGateRPCServer(l)
	// signalhandler.SignalKillHandler(gatexRPC)
	var wg_service util.WaitGroupWrapper
	wg_service.Wrap(func() { g.rpcServer.Start(g) })

	status.Start(
		gateconfig.Cfg.GateConfig.PublicIP,
		internalIpPort,
	)

	go g.loop()
	// g.waitGroup.Wrap(func() { g.loop() })
	g.waitGroup.Wrap(func() { g.gameSrvs.WaitAllShutdown(g.quit) })
	g.waitGroup.Wait()
	// close(g.loginStatusChan)

	g.server.Stop()
	g.rpcServer.Stop()
	status.Stop()

	wg_service.Wait()
}

func (g *GateServer) Stop() {
	close(g.quit)
}

func (g *GateServer) IsReg2Etcd() bool {
	return gateconfig.RunMode == gateconfig.RunModeTest ||
		gateconfig.RunMode == gateconfig.RunModeProd
}

func (g *GateServer) NewChatRedis() bool {
	g.chatDb = chat.NewRedisPool("chatredis",
		gateconfig.EtcdConf.ChatxRedisAddr,
		gateconfig.EtcdConf.ChatxRedisDbPwd,
		gateconfig.EtcdConf.ChatxRedisDb,
		gateconfig.EtcdConf.ChatxRedisAddr, &gateconfig.EtcdConf, redispool.DefaultRedisPoolCapacity)
	return g.chatDb != nil
}
