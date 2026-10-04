package scene

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/nghichtu91/platform/share/planx/merge_util"
	"github.com/nghichtu91/platform/share/planx/scene_pool"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tracing"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/msg/scene/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type scene_server struct {
	sceMgr *sceneMgr
}

const (
	IDSplit                      = ":"
	MainCityDefaultSceneLineNum  = 3
	xMainCityDefaultSceneLineNum = 4
	StaticSceneType              = "1" // 静态场景
	XSceneType                   = "5" // 跨服场景类型，算静态
	XDynamicSceneType            = "6" // 跨服动态场景
)

const (
	SyncMetricNameGetSceneLineInfo = "getscelineinforeq"
	SyncMetricNameChangeSceneLine  = "changescelinereq"
)

const (
	MetaDataSceneInfo = "sceneinfo"
	MetaDataShardInfo = "shardinfo"
)

const (
	MessageC2SRoleMove        = 25001
	MessageC2SRoleStop        = 25002
	MessageC2SRoleCheck       = 25003
	MessageG2SPlayerKeepAlive = 10012
)

var (
	ErrSceneMapNotExists = errors.New("scene map not exists")
)

func (s *scene_server) OnStream(stream pb.GameScene_OnStreamServer) error {
	// 获取Stream构建时的MetaData。
	mapID, sceneUniq, sceneType, shardID, warzoneID, err := getSceneInfoFromContext(stream.Context())
	if err != nil {
		return err
	}

	uid := util.Int32Merge2Int64(int32(warzoneID), int32(mapID))

	switch sceneType {
	case StaticSceneType:
		// 首先删除上次的相应场景线管理器。
		// s.sceMgr.stopSceneLineMgr(uid)
		// 开启新的场景线管理器并创建默认的分线。
		// sl := s.sceMgr.startSceneLineMgr(int32(mapID), 0, sceneType, MainCityDefaultSceneLineNum)

		// JWS2-47855 单服场景也改成复用场景线管理器
		sl := s.sceMgr.getSceneLineMgr(uid)
		if sl == nil {
			sl = s.sceMgr.startSceneLineMgr(int32(mapID), 0, sceneType, MainCityDefaultSceneLineNum)
		}

		innerQuit := make(chan struct{}, 1)
		// Recv
		var w util.WaitGroupWrapper
		w.Wrap(func() {
			defer func() {
				tilogs.L().Infof("scene_server stream.Recv stop, sceneID %v", sceneUniq)
				close(innerQuit)
			}()
			for {
				msg := scene_pool.GetMsg()
				err := stream.RecvMsg(msg)
				// msg, err := stream.Recv()
				if err != nil && err == io.EOF {
					return
				}
				if err != nil {
					tilogs.L().Errorf("scene_server stream.Recv, sceneID %v, err %v", sceneUniq, err.Error())
					return
				}

				// 处理收到的异步请求消息。
				s.handleRevMsg(msg)
			}
		})
		// send
		w.Wrap(func() {
			defer func() {
				tilogs.L().Infof("scene_server send stop, sceneId %s", sceneUniq)
			}()
			for {
				select {
				case msg, ok := <-sl.getSceneOutChan():
					if !ok {
						return
					}
					err := stream.Send(msg)
					if err != nil {
						tilogs.L().Errorf("scene %v send err %v", msg.GetSceneId(), err)
					}
					// 监控统计
					s.recordSendMetrics(msg)
					scene_pool.PutMsg(msg)
				case <-innerQuit:
					return
				}
			}
		})
		w.Wait()
	case XSceneType:
		// xscene同个战区复用单个场景，直到xcross scene关闭才会关。
		// TODO 和单服不同的是，单服gamex重启场景都重建，但跨服单个gamex重启，跨服场景保留，不知会不会有数据没清理
		sl := s.sceMgr.getSceneLineMgr(uid)
		if sl == nil {
			sType := s.sceMgr.getSceneType(int32(mapID))
			sl = s.sceMgr.startSceneLineMgr(int32(mapID), int32(warzoneID), fmt.Sprint(sType), xMainCityDefaultSceneLineNum)
		}

		innerQuit := make(chan struct{}, 1)
		outChan := make(chan *pb.SceneMsg, 8196)
		sl.outChans.Store(uint32(shardID), outChan)
		// 主服所有对应的合服ShardID，也一并放入outChans
		otherShards, err := merge_util.GetSIDsFromReal(shardID)
		if err != nil {
			tilogs.L().Errorf("scene_server OnStream get shard %d sub shards failed, err %s", shardID, err.Error())
		}
		for _, sid := range otherShards {
			if sid == shardID {
				continue
			}
			sl.outChans.Store(uint32(sid), outChan)
		}

		tilogs.L().Debugf("scene_server OnStream xscene uid %d outChans add shard %d with outChan %v",
			uid, shardID, outChan)

		// Recv
		var w util.WaitGroupWrapper
		w.Wrap(func() {
			defer func() {
				tilogs.L().Infof("scene_server stream.Recv xscene uid %d stop, shardID %d, zoneID %d, uid %d",
					uid, shardID, warzoneID, uid)
				close(innerQuit)
				// gamex重连时，有可能出现新chan已经替换旧chan的情况
				// 需要检验一下目前的chan还是不是当前goroutine的
				v, ok := sl.outChans.Load(uint32(shardID))
				if ok && v != nil {
					if v.(chan *pb.SceneMsg) == outChan {
						tilogs.L().Debugf("scene_server xscene uid %d OnStream outChans del shard %d with outChan %v",
							uid, shardID, outChan)
						sl.outChans.Delete(uint32(shardID))
					}
				}
				close(outChan)
				// s.sceMgr.stopSceneLineMgr(int32(uID))
			}()
			for {
				msg := scene_pool.GetMsg()
				err := stream.RecvMsg(msg)
				// msg, err := stream.Recv()
				if err != nil && err == io.EOF {
					return
				}
				if err != nil {
					tilogs.L().Errorf("scene_server stream.Recv, sceneID %v, err %v", sceneUniq, err.Error())
					return
				}

				// 处理收到的异步请求消息。
				s.handleRevMsg(msg)
			}
		})
		// send
		w.Wrap(func() {
			defer func() {
				tilogs.L().Infof("scene_server send stop, sceneId %s", sceneUniq)
			}()
			for {
				select {
				case msg, ok := <-outChan:
					if !ok {
						return
					}
					err := stream.Send(msg)
					if err != nil {
						tilogs.L().Errorf("scene %v send err %v", msg.GetSceneId(), err)
					}
					// 监控统计
					s.recordSendMetrics(msg)
					scene_pool.PutMsg(msg)
				case <-innerQuit:
					return
				}
			}
		})
		w.Wait()
	case XDynamicSceneType:
		innerQuit := make(chan struct{}, 1)
		outChan := make(chan *pb.SceneMsg, 8192)

		s.sceMgr.setDynamicOutChans(int32(warzoneID), sceneUniq, uint32(shardID), outChan)

		// Recv
		var w util.WaitGroupWrapper
		w.Wrap(func() {
			defer func() {
				tilogs.L().Infof("scene_server Recv stop, sceneID: %v", sceneUniq)
				close(innerQuit)
				s.sceMgr.deleteDynamicOutChans(int32(warzoneID), sceneUniq, uint32(shardID), outChan)
				// 不删除Map
			}()
			for {
				msg := scene_pool.GetMsg()
				err := stream.RecvMsg(msg)
				// msg, err := stream.Recv()
				if err != nil && err == io.EOF {
					return
				}
				if err != nil {
					tilogs.L().Errorf("scene_server Recv, sceneID: %v, err: %v", sceneUniq, err.Error())
					return
				}

				// 监控统计
				s.recordRequestMetrics(msg)
				sce := s.sceMgr.getScene(msg.GetSceneId())
				if sce == nil {
					tilogs.L().Errorf("scene_server Recv, sceneID: %v s.sceMgr.getScene is nil", msg.GetSceneId())
					continue
				}
				sce.in(msg)
			}
		})
		// Send
		w.Wrap(func() {
			defer func() {
				tilogs.L().Infof("scene_server Send stop, sceneID: %s", sceneUniq)
			}()
			for {
				select {
				case msg, ok := <-outChan:
					if !ok {
						return
					}
					err := stream.Send(msg)
					if err != nil {
						tilogs.L().Errorf("sceneID: %v send err: %v", msg.GetSceneId(), err)
					}
					// 监控统计
					s.recordSendMetrics(msg)
					scene_pool.PutMsg(msg)
				case <-innerQuit:
					return
				}
			}
		})
		w.Wait()
	default:
		// default分支是动态场景
		// 每个mapID会新建一个stream
		innerQuit := make(chan struct{}, 1)

		// JWS2-47856 动态场景也改成复用
		// 同一个动态地图的不同分线共享一个outChan。
		outChan := s.sceMgr.getDynamicOutChan(sceneUniq)
		if outChan == nil {
			outChan = make(chan *pb.SceneMsg, 8192)
			s.sceMgr.setDynamicOutChan(sceneUniq, outChan)
		}

		// Recv
		var w util.WaitGroupWrapper
		w.Wrap(func() {
			defer func() {
				tilogs.L().Infof("scene_server Recv stop, sceneID: %v", sceneUniq)
				close(innerQuit)
				// s.sceMgr.deleteDynamicOutChan(sceneUniq)
				// s.sceMgr.stopDynamicMapScene(int32(mapID))
			}()
			for {
				msg := scene_pool.GetMsg()
				err := stream.RecvMsg(msg)
				// msg, err := stream.Recv()
				if err != nil && err == io.EOF {
					return
				}
				if err != nil {
					tilogs.L().Errorf("scene_server Recv, sceneID: %v, err: %v", sceneUniq, err.Error())
					return
				}

				// 监控统计
				s.recordRequestMetrics(msg)
				sce := s.sceMgr.getScene(msg.GetSceneId())
				if sce == nil {
					tilogs.L().Errorf("scene_server Recv, sceneID: %v s.sceMgr.getScene is nil", msg.GetSceneId())
					continue
				}
				sce.in(msg)
			}
		})
		// Send
		w.Wrap(func() {
			defer func() {
				tilogs.L().Infof("scene_server Send stop, sceneID: %s", sceneUniq)
			}()
			for {
				select {
				case msg, ok := <-outChan:
					if !ok {
						return
					}
					err := stream.Send(msg)
					if err != nil {
						tilogs.L().Errorf("sceneID: %v send err: %v", msg.GetSceneId(), err)
					}
					// 监控统计
					s.recordSendMetrics(msg)
					scene_pool.PutMsg(msg)
				case <-innerQuit:
					return
				}
			}
		})
		w.Wait()
	}

	return nil
}

func (s *scene_server) OnCommand(ctx context.Context, msg *pb.SceneMsg) (*pb.SceneMsg, error) {
	tracer := opentracing.GlobalTracer()
	spanContext, err := tracing.ExtractSpanContext(ctx, tracer)
	if err == nil {
		span := tracer.StartSpan(
			"sceneServerOnCommand-"+strconv.Itoa(int(msg.MsgType))+"-"+strconv.Itoa(int(msg.MsgId)),
			ext.RPCServerOption(spanContext))
		defer span.Finish()
		ctx = opentracing.ContextWithSpan(ctx, span)
	}

	// 监控统计（如果msg中携带的信息无需enum中的msgID的定义，需要在onCommandCountSizeMetricsRecord新增Case）。
	metricsName := s.onCommandCountSizeMetricsRecord(msg)
	beforeTimeNano := time.Now().UnixNano()
	defer recordTime(metricsName, metricsName, beforeTimeNano)

	// 场景线ID不带分线，可能处于传送过程中，此处增加追踪用日志，无后续问题时，此段代码应移除。
	// 2023.04.07 移除下面的逻辑
	/*
		if !strings.Contains(msg.GetSceneId(), IDSplit) {
			if msg.GetMsgType() != pb.MsgType_EnterScene &&
				msg.GetMsgType() != pb.MsgType_ForceEnterScene &&
				msg.GetMsgType() != pb.MsgType_GetSceneLine &&
				msg.GetMsgType() != pb.MsgType_GetSceneLineNotRealTime {
				retErr := fmt.Errorf("scene_server OnCommand sceneID %s without sceneLine, msgID: %d, msg: %s",
					msg.GetSceneId(), msg.GetMsgId(), msg.String())
				tilogs.L().Errorf(retErr.Error())
				return nil, retErr
			}
		}
	*/

	// 动态场景和静态场景的处理逻辑不同。
	mapID := getMapBySceneUniq(msg.GetSceneId())
	uid := util.Int32Merge2Int64(msg.GetWarzoneId(), mapID)
	if mapID == 0 {
		tilogs.L().Errorf("------OnCommand %s", msg.String())
	}
	sceneType := s.sceMgr.getSceneType(mapID)
	tilogs.L().Infof("OnCommand mapID:%d uid:%d sceneType:%d warZoneId:%d", mapID, uid, sceneType, msg.GetWarzoneId())
	switch strconv.Itoa(int(sceneType)) {
	case StaticSceneType, XSceneType:
		// 静态场景。
		// 获取相应的场景线管理对象。
		slMgr := s.sceMgr.getSceneLineMgr(uid)
		if slMgr == nil {
			tilogs.L().Errorf("scene_server OnCommand s.sceMgr.getSceneLineMgr slMgr is nil, uID: %d, msg: %s", mapID, msg.String())
			// 2023.04.07 改为指定Err，方便gamex处理异常
			msg.ErrDes = ErrSceneMapNotExists.Error()
			return msg, ErrSceneMapNotExists
		}

		// 通过SceneLineManager向Scene发起同步请求。
		retMsg := slMgr.inSync(ctx, msg)
		if retMsg != nil {
			return retMsg, nil
		} else {
			retErr := fmt.Errorf("scene_server OnCommand slMgr.inSync retMsg is nil, mapType: %v, sceneUniq: %v, msg: %v", sceneType, msg.GetSceneId(), msg)
			tilogs.L().Errorf(retErr.Error())
			return nil, retErr
		}
	default:
		return s.OnCommandDynamicScene(uid, mapID, int(sceneType), msg, ctx)
	}
}

func (s *scene_server) OnCommandDynamicScene(uid int64, mapID int32, sceneType int, msg *pb.SceneMsg, ctx context.Context) (*pb.SceneMsg, error) {
	sceneLineUniq := getDynamicSceneUniq(msg.GetSceneId(), msg.GetWarzoneId())
	// 动态场景。
	scene := s.sceMgr.getScene(sceneLineUniq)
	if scene == nil {
		if !s.sceMgr.addScene(mapID, msg.GetWarzoneId(), sceneLineUniq, strconv.Itoa(int(sceneType)), nil, nil, true, nil) {
			retErr := fmt.Errorf("scene_server OnCommand s.sceMgr.addScene error, uID: %v, sceneID: %v, sceneType: %v", mapID, sceneLineUniq, sceneType)
			tilogs.L().Errorf(retErr.Error())
			return nil, retErr
		}
		scene = s.sceMgr.getScene(sceneLineUniq)
	}

	// 向场景线发起同步请求。
	resp := scene.inSync(ctx, msg)
	if resp.msg != nil {
		return resp.msg, nil
	} else {
		retErr := fmt.Errorf("scene_server OnCommand scene.inSync retMsg is nil, mapType: %v, sceneUniq: %v, msg: %v", sceneType, sceneLineUniq, msg)
		tilogs.L().Errorf(retErr.Error())
		return nil, retErr
	}
}

// onCommandCountSizeMetricsRecord OnCommand时，记录msg的数量和大小，返回Metric的名字。
//
// Note: 如果msg中携带的信息无需enum中的msgID的定义，则需要在Record中新增Case。
func (s *scene_server) onCommandCountSizeMetricsRecord(msg *pb.SceneMsg) string {
	switch msg.GetMsgType() {
	case pb.MsgType_GetSceneLine:
		// 获取各个场景分线的信息。
		return s.recordLineRequestMetrics(SyncMetricNameGetSceneLineInfo)
	case pb.MsgType_ChangeSceneLine:
		// 切换场景分线时，玩家预占位。
		return s.recordLineRequestMetrics(SyncMetricNameChangeSceneLine)
	default:
		return s.recordRequestMetrics(msg)
	}
}

// getSceneInfoFromContext 从context的元数据中获取SceneInfo。
func getSceneInfoFromContext(context context.Context) (int, string, string, int, int, error) {
	// 获取context中的MetaData。
	md, ok := metadata.FromIncomingContext(context)
	if !ok || len(md[MetaDataSceneInfo]) <= 2 {
		tilogs.L().Errorf("scene_server getSceneInfoFromContext FromIncomingContext metaData error, metaData: %v", md)
		return 0, "", "", 0, 0, fmt.Errorf("scene_server OnStream no meta")
	}

	mapID, err := strconv.Atoi(md[MetaDataSceneInfo][0])
	if err != nil {
		tilogs.L().Errorf("scene_server getSceneInfoFromContext uID is not int, uID: %v, err: %v", md[MetaDataSceneInfo][0], err)
		return 0, "", "", 0, 0, err
	}
	sceneUniq := md[MetaDataSceneInfo][1] // 主城场景和mapID相同，动态场景不同。
	sceneType := md[MetaDataSceneInfo][2]

	// 跨服场景使用
	var shardID, warzoneID int
	switch len(md[MetaDataShardInfo]) {
	case 2:
		warzoneID, _ = strconv.Atoi(md[MetaDataShardInfo][1])
		fallthrough
	case 1:
		shardID, _ = strconv.Atoi(md[MetaDataShardInfo][0])
	}

	tilogs.L().Infof("scene_server getSceneInfoFromContext get mapID: %v, sceneUniq: %v, sceneType: %v, shardID: %d, warzoneID %d",
		mapID, sceneUniq, sceneType, shardID, warzoneID)
	return mapID, sceneUniq, sceneType, shardID, warzoneID, nil
}

// handleRevMsg 处理接受到的异步请求。
func (s *scene_server) handleRevMsg(msg *pb.SceneMsg) {
	// 监控统计（消息处理时间的监控统计指标在场景线中计算和记录）。
	s.recordRequestMetrics(msg)

	// 场景线ID不带分线，可能处于传送过程中，此处增加追踪用日志，无后续问题时，此段代码应移除。
	if !strings.Contains(msg.GetSceneId(), IDSplit) {
		if msg.GetMsgId() == MessageC2SRoleMove || msg.GetMsgId() == MessageC2SRoleStop ||
			msg.GetMsgId() == MessageC2SRoleCheck || msg.GetMsgId() == MessageG2SPlayerKeepAlive {
			// 客户端发送的移动包和保活包可以接受，因为有可能发生在传送的过程中。
			tilogs.L().Warnf("handleRevMsg sceneID: %v without sceneLine, msgID: %v, msg: %v", msg.GetSceneId(), msg.GetMsgId(), msg)
		} else {
			tilogs.L().Errorf("handleRevMsg sceneID %v without sceneLine, msgID: %v, msg: %v", msg.GetSceneId(), msg.GetMsgId(), msg)
		}
		return
	}

	// 根据sceneID找到对应场景线。
	scene := s.sceMgr.getScene(msg.GetSceneId())
	if scene == nil {
		/*
			虽然OnStream和OnCommand可能会导致时序问题，但是场景线
			在不活跃1分钟后才会被删除，所以这里理论上不会跑到。
		*/
		tilogs.L().Errorf("handleRevMsg s.sceMgr.getScene scene is nil, sceneID: %v", msg.GetSceneId())
		return
	}

	// 向相应的场景线发起异步请求。
	scene.in(msg)
}

type grpc_server struct {
	lis       net.Listener
	s         *grpc.Server
	sceMgr    *sceneMgr
	waitGroup util.WaitGroupWrapper
}

func NewGRPCSer(lis net.Listener, genLogicScene GenLogicScene, getType GetSceneType, gid, shardId uint) *grpc_server {
	return &grpc_server{
		lis:    lis,
		sceMgr: createSceneMgr(genLogicScene, getType, gid, shardId),
	}
}

func (gser *grpc_server) Run() {
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

	pb.RegisterGameSceneServer(gser.s, &scene_server{
		sceMgr: gser.sceMgr,
	})

	gser.waitGroup.Wrap(func() {
		err := gser.s.Serve(gser.lis)
		if err != nil {
			tilogs.L().Errorf("scene grpc sever err %v", err)
		}
		tilogs.L().Infof("grpc_server grpc start...")
	})
}

func (gser *grpc_server) Stop() {
	gser.s.Stop()
	gser.sceMgr.stop()
	gser.waitGroup.Wait()
	tilogs.L().Infof("grpc_server stopped")
}
