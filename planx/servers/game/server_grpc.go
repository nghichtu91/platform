package game

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/nghichtu91/platform/share/planx/client"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
)

type acid_server struct {
	mgr *grpc_server
}

// CtlRpc 控制接口，目前用来强制踢玩家goroutine退出
func (s *acid_server) CtlRpc(ctx context.Context, req *pb.CtlReq) (*pb.CtlResp, error) {
	tilogs.L().Debugf("rev CtlRpc CtlReq %v", req)
	switch req.Typ {
	case pb.CtlTyp_ForceQuit:
		s.mgr.ser_mgr.fForceQuit(req.Param)
	// core.AccountMgrForceQuit(req.Param)
	case pb.CtlTyp_PingCheck:
	case pb.CtlTyp_PushCheck:
		err := s.mgr.batchCheck(req.GetParam())
		if err != nil {
			tilogs.L().Errorf("CtlRpc req %v batchCheck err %v", req, err)
			return &pb.CtlResp{Res: pb.CtlRes_Ctl_Fail, Reason: err.Error()}, nil
		}
	default:
		tilogs.L().Errorf("CtlRpc CtlReq Typ not found, %d", req.Typ)
		return &pb.CtlResp{Res: pb.CtlRes_Ctl_Fail}, nil
	}
	return &pb.CtlResp{Res: pb.CtlRes_Ctl_Success}, nil
}

// 每个玩家的stream，协议收发
func (s *acid_server) OnPacket(stream pb.Gate_OnPacketServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok || len(md["acid"]) <= 0 {
		tilogs.L().Errorf("acid_server OnPacket FromIncomingContext not found")
		return fmt.Errorf("acid_server OnPacket no meta")
	}
	acid := md["acid"][0]
	clientInfo := ""
	_clientInfo, ok := md["clientinfo"]
	if ok && len(_clientInfo) > 0 {
		clientInfo = _clientInfo[0]
	}
	// 通过渠道判断是否为机器人
	ch := ""
	_ch, ok := md["channel"]
	if ok {
		ch = _ch[0]
	}
	tilogs.L().Debugf("conn acid %s clientInfo %s", acid, clientInfo)

	agent := s.mgr.ser_mgr.NewGameServer(acid, 0, clientInfo, ch)

	defer func() {
		s.mgr.ser_mgr.RecycleGameServer(acid, 0, agent)
	}()

	if ch != consts.RobotChannel {
		allmetrics.AddShardCCU()
		defer allmetrics.ReduceShardCCU()
	}

	// rec
	var w util.WaitGroupWrapper
	w.Wrap(func() {
		defer func() {
			agent.Stop()
			tilogs.L().Debugf("<GamexAccountClose> 1/4 acid_server rec stop  %s", acid)
		}()
		for {
			pkt, err := stream.Recv()
			if err != nil && (err == io.EOF || strings.Contains(err.Error(), "code = Canceled desc = context canceled")) {
				return
			}
			if err != nil {
				tilogs.L().Errorf("acid_server Recv err %s", err.Error())
				return
			}
			if !agent.SendPacket(pkt) {
				tilogs.L().Errorf("acid_server send packet failed")
				return
			}
		}
	})
	// send
	w.Wrap(func() {
		defer func() {
			tilogs.L().Debugf("<GamexAccountClose> 2/4 acid_server send stop  %s", acid)
		}()
		ch := agent.GetReadingChan()
		for {
			select {
			case pkt, ok := <-ch:
				if !ok {
					return
				}
				stream.Send(pkt)
			case <-agent.GetGoneChan():
				return
			}
		}
	})
	w.Wait()
	tilogs.L().Debugf("<GamexAccountClose> 3/4 acid_server OnPacket close %s", acid)
	return nil
}

// gamex给gate批量发消息
func (s *acid_server) OnBatchPacket(stream pb.Gate_OnBatchPacketServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok || len(md["serverid"]) <= 0 {
		tilogs.L().Errorf("acid_server OnBatchPacket FromIncomingContext not found")
		return fmt.Errorf("acid_server OnBatchPacket no meta")
	}
	serverId := md["serverid"][0]
	tilogs.L().Infof("acid_server OnBatchPacket %s", serverId)

	revCh := make(chan *pb.BatchPacket, 4096)
	s.mgr.gate_batch_lock.Lock()
	s.mgr.gate_batch_chan[serverId] = revCh
	s.mgr.gate_batch_lock.Unlock()

	var w util.WaitGroupWrapper
	quit := make(chan struct{}, 1)
	// rec
	w.Wrap(func() {
		defer func() {
			tilogs.L().Infof("acid_server OnBatchPacket rev stop")
			close(quit)
		}()
		for {
			_, err := stream.Recv()
			if err != nil && (err == io.EOF || strings.Contains(err.Error(), "code = Canceled desc = context canceled")) {
				tilogs.L().Infof("acid_server OnBatchPacket Recv EOF %v", err)
				return
			}
			if err != nil {
				tilogs.L().Errorf("acid_server OnBatchPacket Recv err %s %s", err.Error(), serverId)
				return
			}
			tilogs.L().Warnf("acid_server OnBatchPacket Recv a msg, but should not rev msg from batchpacket")
		}
	})
	// send
	w.Wrap(func() {
		defer func() {
			tilogs.L().Infof("acid_server OnBatchPacket send stop")
		}()
		for {
			select {
			case msg := <-revCh:
				if err := stream.Send(msg); err != nil {
					tilogs.L().Errorf("acid_server OnBatchPacket stream.Send err %s", err.Error())
				}

				// gamex到gate，对于特定服的广播，打出log
				if tilogs.LevelEnable("debug") {
					needLog := false
					acids := msg.GetAcids()
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
						err := proto.Unmarshal(msg.GetPacket().GetRawData(), sessionPkt)
						if err != nil {
							tilogs.L().Errorf("acid_server OnBatchPacket unmarshal error %v", err)
						} else {
							messageId := sessionPkt.GetClientPacket().GetMessageId()
							util.SessionPacketPool.Put(sessionPkt)
							switch messageId {
							// 对于特定协议
							case 30075, 30323, 30104, 30374, 30377, 30103:
								tilogs.L().Debugf("acid_server OnBatchPacket msgId:%d, acids:%v", messageId, acids)
							}
						}
					}
				}
			case <-quit:
				return
			}
		}
	})
	w.Wait()

	s.mgr.gate_batch_lock.Lock()
	delete(s.mgr.gate_batch_chan, serverId)
	s.mgr.gate_batch_lock.Unlock()

	tilogs.L().Infof("acid_server OnBatchPacket close %s", serverId)
	return nil
}

type grpc_server struct {
	lis     net.Listener
	addr    string
	s       *grpc.Server
	ser_mgr *ChanGameServerManager

	gate_batch_chan map[string]chan *pb.BatchPacket
	gate_batch_lock sync.RWMutex

	quit      chan struct{}
	waitGroup util.WaitGroupWrapper
}

func NewGRPCSer(lis net.Listener) *grpc_server {
	return &grpc_server{
		lis:             lis,
		gate_batch_chan: make(map[string]chan *pb.BatchPacket, 1024),
		quit:            make(chan struct{}, 1),
	}
}

func (gser *grpc_server) Run(fGetPlayer PreparePlayer, fForceQuit FForceQuit) {
	gser.ser_mgr = NewChanGameServerManager(fGetPlayer, fForceQuit)
	gser.s = grpc.NewServer(grpc.MaxConcurrentStreams(1024),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(
				grpc_recovery.WithRecoveryHandler(func(p interface{}) (err error) {
					tilogs.L().Errorf("UnaryServer recover panic %v", p)
					return err
				})),
			// otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer()),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_recovery.StreamServerInterceptor(
				grpc_recovery.WithRecoveryHandler(func(p interface{}) (err error) {
					tilogs.L().Errorf("StreamServer recover panic %v", p)
					return err
				}),
			),
			// otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer()),
		)),
	)

	pb.RegisterGateServer(gser.s, &acid_server{
		mgr: gser,
	})

	gser.waitGroup.Wrap(func() {
		gser.s.Serve(gser.lis)
		tilogs.L().Infof("grpc_server grpc stop...")
	})
	gser.waitGroup.Wrap(func() {
		gser.ser_mgr.WaitAllShutdown(gser.quit)
		tilogs.L().Infof("grpc_server ChanGameServerManager stop...")
	})
	gser.waitGroup.Wrap(func() {
		defer tilogs.L().Infof("grpc_server batch send close")
		defer tilogs.PanicCatcher("grpc_server batch send panic")
		c := GetSendCh()
		for {
			select {
			case msg := <-c:
				func() {
					defer tilogs.PanicCatcher("grpc_server batch send msg panic")

					gser.gate_batch_lock.RLock()
					defer gser.gate_batch_lock.RUnlock()
					for serverId, v := range gser.gate_batch_chan {
						select {
						case v <- msg:
						default:
							tilogs.L().Errorf("grpc_server gate_batch_chan write msg but full, %s", serverId)
						}
					}
				}()
			case <-gser.quit:
				return
			}
		}
	})
	// gser.regEtcd()
	gser.waitGroup.Wait()
}

func (gser *grpc_server) Stop() {
	// gser.unregEtcd()
	gser.s.Stop()
	close(gser.quit)
	tilogs.L().Infof("grpc_server prepare stop")
}

var (
	ErrGateBatchNotFound = fmt.Errorf("gate batch not found")
	ErrGateBatchFull     = fmt.Errorf("gate batch full")
)

var batchCheckPacket = &pb.BatchPacket{
	Packet: &pb.Packet{
		PacketId: int32(client.PacketIDBatchCheck),
	},
}

// batchCheck
func (gser *grpc_server) batchCheck(gateId string) error {
	var batchChan chan *pb.BatchPacket
	gser.gate_batch_lock.RLock()
	batchChan = gser.gate_batch_chan[gateId]
	gser.gate_batch_lock.RUnlock()
	if batchChan == nil {
		return ErrGateBatchNotFound
	}
	select {
	case batchChan <- batchCheckPacket:
	default:
		return ErrGateBatchFull
	}
	return nil
}
