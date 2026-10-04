package gate

import (
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"

	"github.com/nghichtu91/platform/share/planx"

	"github.com/nghichtu91/platform/share/planx/servers/gate"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"

	"github.com/nghichtu91/platform/share/planx/util"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

var gGamexMgr *ProtoGameServerManager

// ProtoGameServerManager 管理 gamex<->gatex server<->gatex agent<->client 链路中的gatex agent
// 每个agent为ProtoGameServer
type ProtoGameServerManager struct {
	gamesMgr        *gamex_mgr
	acid2GameServer *AcidGameServers
	batchChan       chan *pb.BatchPacket
	wg              util.WaitGroupWrapper
	quit            chan struct{}
}

func NewProtoGameServerManager(mgr *gamex_mgr) *ProtoGameServerManager {
	gGamexMgr = &ProtoGameServerManager{
		gamesMgr: mgr,
		acid2GameServer: &AcidGameServers{
			acid2GameServer: make(map[string]*ProtoGameServer, 10240),
		},
		batchChan: make(chan *pb.BatchPacket, 1024),
		quit:      make(chan struct{}),
	}
	gGamexMgr.start()
	return gGamexMgr
}

func (mg *ProtoGameServerManager) NewGameServer(accountId string, sessionId int64, clientInfo, channel string) gate.GameServer {
	conn, err := mg.gamesMgr.getConn(accountId)
	if err != nil {
		tilogs.L().Errorf("ProtoGameServerManager gamesMgr.getConn err %s", err.Error())
		return nil
	}
	if conn == nil {
		tilogs.L().Errorf("ProtoGameServerManager gamesMgr.getConn nil")
		return nil
	}
	md := metadata.Pairs("acid", accountId, "clientinfo", clientInfo, "channel", channel)
	mctx := metadata.NewOutgoingContext(context.Background(), md)
	ctx, cancel := context.WithCancel(mctx)
	stream, err := pb.NewGateClient(conn.getClientConn()).OnPacket(ctx)
	if err != nil {
		tilogs.L().Errorf("ProtoGameServerManager gate_client.OnPacket err %s", err.Error())
		cancel()
		return nil
	}
	mg.wg.Add(1)
	conn.addCcu()
	ser := &ProtoGameServer{
		acid:      accountId,
		sessionId: sessionId,
		conn:      conn,
		stream:    stream,
		out:       make(chan *pb.Packet, 1024),
		done:      make(chan struct{}),
		ctxCancel: cancel,
	}
	go ser.runReading()
	mg.acid2GameServer.add(accountId, ser)
	tilogs.L().Debugf("conn acid %s", accountId)
	return ser
}

func (mg *ProtoGameServerManager) RecycleGameServer(acid string, sessionId int64, gs gate.GameServer) {
	mg.acid2GameServer.del(acid, sessionId)
	gs.Stop()
	mg.wg.Done()
}

func (mg *ProtoGameServerManager) WaitAllShutdown(quit <-chan struct{}) {
	<-quit
	close(mg.quit)
	mg.wg.Wait()
	tilogs.L().Infof("ProtoGameServerManager WaitAllShutdown finish ...")
}

func (mg *ProtoGameServerManager) forceQuit(accountId string) {
	conn, err := mg.gamesMgr.getConn(accountId)
	if err != nil {
		tilogs.L().Errorf("ProtoGameServerManager gamesMgr.getConn %s err %s", accountId, err.Error())
		return
	}
	resp, err := pb.NewGateClient(conn.getClientConn()).CtlRpc(context.Background(),
		&pb.CtlReq{
			Typ:   pb.CtlTyp_ForceQuit,
			Param: accountId,
		})
	if err != nil {
		tilogs.L().Errorf("ProtoGameServerManager gate_client.forceQuit %s err %s", accountId, err.Error())
		return
	}
	if resp.GetRes() != pb.CtlRes_Ctl_Success {
		tilogs.L().Errorf("ProtoGameServerManager gate_client.forceQuit resp fail, %s", accountId)
		return
	}
}

func (mg *ProtoGameServerManager) start() {
	mg.wg.Wrap(func() {
		defer func() {
			tilogs.L().Infof("ProtoGameServerManager start has stopped")
		}()
		for {
			select {
			case packet := <-mg.batchChan:
				mg.acid2GameServer.send2Client(packet)
			case <-mg.quit:
				return
			}
		}
	})
}

func (mg *ProtoGameServerManager) SendBatchPacket(packet *pb.BatchPacket) {
	ctx, cancel := context.WithTimeout(context.Background(), planx.DownTimeOut)
	defer cancel()
	select {
	case mg.batchChan <- packet:
	case <-ctx.Done():
		tilogs.L().Errorf("ProtoGameServerManager sendBatchPacket full")
	}
}

// ProtoGameServer gamex->gatex server->gatex agent->client链路中的gatex server
// out用于给gatex agent发送消息
type ProtoGameServer struct {
	acid      string
	sessionId int64
	conn      *gpr_conn
	stream    pb.Gate_OnPacketClient
	out       chan *pb.Packet
	done      chan struct{}
	closeFlag int32
	ctxCancel func()
}

func (ser *ProtoGameServer) runReading() {
	defer func() {
		ser.Stop()
		tilogs.L().Debugf("<GateAccountClose> 2/4 ProtoGameServer runReading stop  %s", ser.acid)
	}()
	for {
		if ser.IsClosed() {
			return
		}
		p, err := ser.stream.Recv()
		if err != nil && (err == io.EOF || strings.Contains(err.Error(), "code = Canceled desc = context canceled")) {
			return
		}
		if err != nil {
			tilogs.L().Errorf("ProtoGameServer runReading stream.Recv err %s", err.Error())
			return
		}
		select {
		case ser.out <- p:
		default:
			tilogs.L().Errorf("ProtoGameServer runReading out channel full !!")
		}
	}
}

func (ser *ProtoGameServer) GetReadingChan() <-chan *pb.Packet {
	return ser.out
}

func (ser *ProtoGameServer) SendPacket(pkt *pb.Packet) bool {
	if ser.IsClosed() {
		return false
	}
	if err := ser.stream.Send(pkt); err != nil {
		tilogs.L().Errorf("ProtoGameServer SendPacket err %s", err.Error())
		ser.Stop()
		return false
	}
	util.PacketPool.Put(pkt)
	return true
}

func (ser *ProtoGameServer) GetGoneChan() <-chan struct{} {
	return ser.done
}

func (ser *ProtoGameServer) Stop() {
	if atomic.CompareAndSwapInt32(&ser.closeFlag, 0, 1) {
		// ser.stream.CloseSend() // 调用grpc的stream.CloseSend会有race问题，因此只用context的cancel关闭stream，https://github.com/grpc/grpc-go/issues/2927
		close(ser.done)
		ser.conn.redCcu()
		ser.ctxCancel()
	}
}

func (ser *ProtoGameServer) IsClosed() bool {
	return atomic.LoadInt32(&ser.closeFlag) != 0
}

// AcidGameServers 用于群发的GameServer
type AcidGameServers struct {
	sync.RWMutex
	acid2GameServer map[string]*ProtoGameServer
}

func (ags *AcidGameServers) add(acid string, ser *ProtoGameServer) {
	ags.Lock()
	ags.acid2GameServer[acid] = ser
	ags.Unlock()
}

func (ags *AcidGameServers) del(acid string, sessionId int64) {
	ags.Lock()
	if ser, ok := ags.acid2GameServer[acid]; ok {
		if ser.sessionId == sessionId {
			delete(ags.acid2GameServer, acid)
		}
	}
	ags.Unlock()
}

func (ags *AcidGameServers) send2Client(packet *pb.BatchPacket) {
	sers := make(map[string]*ProtoGameServer, 512)
	tilogs.L().Debugf("AcidGameServers send2Client BatchPacket %v", packet)
	ags.RLock()
	for _, acid := range packet.Acids {
		ser, ok := ags.acid2GameServer[acid]
		if ok {
			sers[acid] = ser
		}
	}
	ags.RUnlock()
	for acid, ser := range sers {
		select {
		case ser.out <- packet.Packet:
		default:
			tilogs.L().Warnf("AcidGameServers send2Client fail, acid %s", acid)
		}
	}

	if tilogs.LevelEnable("debug") {
		needLog := false
		acids := packet.GetAcids()
		if len(acids) > 0 {
			info := strings.Split(acids[0], ":")
			if len(info) == 3 {
				shardId, _ := strconv.ParseUint(info[1], 10, 0)
				if (shardId >= 610023 && shardId <= 610026) ||
					(shardId >= 610043 && shardId <= 610050) ||
					(shardId >= 610063 && shardId <= 610078) ||
					(shardId == 610219) {
					needLog = true
				} else {
					needLog = false
				}
			}
		}
		if needLog {
			sessionPkt := util.SessionPacketPool.Get().(*pb.SessionPacket)
			sessionPkt.Reset()
			err := proto.Unmarshal(packet.GetPacket().GetRawData(), sessionPkt)
			if err != nil {
				tilogs.L().Errorf("AcidGameServers send2Client unmarshal error %v", err)
			} else {
				messageId := sessionPkt.GetClientPacket().GetMessageId()
				util.SessionPacketPool.Put(sessionPkt)
				switch messageId {
				//对于特定协议
				case 30075, 30323, 30104, 30374, 30377, 30103:
					tilogs.L().Debugf("AcidGameServers send2Client msgId:%d, acid:%v", messageId, acids)
				}
			}
		}
	}
}
