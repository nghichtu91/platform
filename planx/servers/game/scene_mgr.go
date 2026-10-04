package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/scene_pool"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tracing"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/msg/scene/pb"
)

const (
	metaKeySceneInfo = "sceneinfo"
	metaKeyShardInfo = "shardinfo"

	sceneMsgChanDefaultSize = 4096 // 单个场景channel长度
	sceneDefaultSize        = 256  // 初始化默认场景数量
)

type GenOnRevSceneServMsg func() IOnRevSceneServMsg

type IOnRevSceneServMsg interface {
	OnRevSceneServMsg(sceneServId string, mapId int32, msg *pb.SceneMsg) // 在goroutine中处理
	NotifyPlayerReEnterScene(sceneId string)                             // 场景服跟逻辑服重新连接后，通知场景服初始化之前在gamex已经在主城的玩家(scene重启)
}

var scnServMgr *sceneServsMgr

var (
	ErrSceneMgrNotExists = errors.New("scene manager not exists")

	sceneUID uint64
)

// sceneServsMgr gamex管理scene相关类型
type sceneServsMgr struct {
	shardId        string
	disMgr         *dis.DiscoveryMgr
	sceneServs     map[string]*sceneServ // key: sceneServId，也有xscene
	configSceneIds map[int32]int32       // 策划配置的需要几个场景

	xSceneID  string          // 目前只连一个跨服场景
	xSceneIDs map[int32]int32 // 策划配置跨服场景需要几个配置
	// xSceneID2srvID sync.Map        // 跨服场景使用的场景ID [MapLevelID][SrvID] (MapLevelID是levelinfodata表的主键)
	xMgr      *xSceneMgr // 控制xScene watcher
	warZoneID int32
	// xSceneIDSuffix string // 跨服场景ID需要加上 ":战区ID" 后缀

	configCrossSceneIDs map[int32]int32 // 策划配置的跨服场景

	servDisCounter int // 服务发现Add和Del的次数统计，查问题用

	gen  GenOnRevSceneServMsg
	wait *util.WaitGroupWrapper
	lock sync.RWMutex

	quit chan struct{}
}

func StartSceneServMgr(etcdServer string, gid, shardId uint, mainCityIds, crossMapIds map[int32]int32, gen GenOnRevSceneServMsg) error {
	scnServMgr = &sceneServsMgr{
		shardId:             strconv.Itoa(int(shardId)),
		sceneServs:          make(map[string]*sceneServ, 2),
		configSceneIds:      mainCityIds,
		configCrossSceneIDs: crossMapIds,
		gen:                 gen,
		wait:                &util.WaitGroupWrapper{},
		quit:                make(chan struct{}, 1),
	}
	// 启动服务器发现scene
	scnServMgr.disMgr = dis.NewDiscoveryMgr(dis.DiscoveryConfig{
		EtcdRoot: etcdServer,
		BeDiscoverySerTyp: dis.ServiceType{
			Gid:    strconv.Itoa(int(gid)),
			SerTyp: etcd.Ser_Scene,
		},
		BeDiscoverySerId: strconv.Itoa(int(shardId)), // 只发现服务于本shard的scene
		HasPrivateAddr:   true,
	}, scnServMgr, nil)
	if scnServMgr.disMgr == nil {
		return fmt.Errorf("scene_mgr discovery StartSceneServMgr fail")
	}

	go scnServMgr.watchXScene(etcdServer, gid, shardId, scnServMgr.quit, &util.WaitGroupWrapper{})

	return scnServMgr.disMgr.Start()
}

func ChangeSceneLine(ctx context.Context, mapID int32, sceneId string, msg *pb.SceneMsg) *pb.SceneMsg {
	if scnServMgr == nil {
		return nil
	}

	id, isCross := GetSceneSrvID(mapID)
	s := scnServMgr.get(id)
	if s == nil {
		return nil
	}

	if isCross {
		msg.WarzoneId = scnServMgr.warZoneID

		// 如果SceneID是跨服场景，且战区ID和当前不一致
		// 替换SceneID里的战区ID
		sceneId = TryReplaceSceneIDSuffixWithWarzoneID(sceneId, scnServMgr.warZoneID)
	}

	return s.sendSync(ctx, sceneId, msg)
}

func Send2SceneServ(mapId int32, msg *pb.SceneMsg) bool {
	if scnServMgr == nil {
		return false
	}

	id, isCross := GetSceneSrvID(mapId)
	s := scnServMgr.get(id)
	if s == nil {
		return false
	}

	if isCross {
		msg.WarzoneId = scnServMgr.warZoneID

		// 如果SceneID是跨服场景，且战区ID和当前不一致
		// 替换SceneID里的战区ID
		msg.SceneId = TryReplaceSceneIDSuffixWithWarzoneID(msg.SceneId, scnServMgr.warZoneID)
	}

	return s.send(mapId, msg)
}

// Deprecated: Use Send2SceneSerSyncWithCtx instead.
func Send2SceneSerSync(mapId int32, sceneId string, msg *pb.SceneMsg) *pb.SceneMsg {
	return Send2SceneSerSyncWithCtx(nil, nil, mapId, sceneId, msg)
}

func Send2SceneSerSyncWithCtx(ctx context.Context, fatherSpan opentracing.Span, mapId int32, sceneId string, msg *pb.SceneMsg) *pb.SceneMsg {
	if scnServMgr == nil {
		return nil
	}
	var ret *pb.SceneMsg
	id, isCross := GetSceneSrvID(mapId)
	s := scnServMgr.get(id)
	if s != nil {
		if ctx == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			defer cancel()
		} else {
			// 通过context的value带过去span的信息
			if fatherSpan == nil {
				span, _ := opentracing.StartSpanFromContext(ctx, "Send2SceneSerSyncWithCtx")
				defer span.Finish()
				ctx = tracing.InjectSpanContext(ctx, opentracing.GlobalTracer(), span)
			} else {
				span := opentracing.GlobalTracer().StartSpan("Send2SceneSerSyncWithCtx", opentracing.ChildOf(fatherSpan.Context()))
				defer span.Finish()
				ctx = tracing.InjectSpanContext(ctx, opentracing.GlobalTracer(), span)
			}
		}
		if isCross {
			msg.WarzoneId = scnServMgr.warZoneID

			// 如果SceneID是跨服场景，且战区ID和当前不一致
			// 替换SceneID里的战区ID
			sceneId = TryReplaceSceneIDSuffixWithWarzoneID(sceneId, scnServMgr.warZoneID)
		}
		ret = s.sendSync(ctx, sceneId, msg)
	}
	if ret == nil {
		tilogs.L().Errorf("Send2SceneSerSyncWithCtx sceneServ not found, acid %v msg.SceneId %v mapId %d, sceneId %s, id %s isCross %v s != nil %v",
			msg.GetAcids(), msg.GetSceneId(), mapId, sceneId, id, isCross, s != nil)
	}
	return ret
}

func StopSceneMgr() {
	tilogs.L().Infof("StopSceneMgr ...before")
	if scnServMgr != nil {
		if scnServMgr.xMgr != nil {
			scnServMgr.xMgr.stop()
		}
		scnServMgr.stop()
	}
	tilogs.L().Infof("StopSceneMgr ...after")
}

func (scn *sceneServsMgr) get(sceneServId string) *sceneServ {
	scn.lock.RLock()
	defer scn.lock.RUnlock()
	return scn.sceneServs[sceneServId]
}

func GetSceneServ(sceneServId string) *sceneServ {
	if scnServMgr == nil {
		return nil
	}
	s := scnServMgr.get(sceneServId)
	return s
}

// OnAddService 非跨服场景服服务发现
func (scn *sceneServsMgr) OnAddService(info dis.ServiceInfo) {
	// 目前一个shard只支持发现自己服独有的scene
	if scn.shardId != info.SerId.SerId {
		tilogs.L().Errorf("sceneServsMgr OnAddService found other sceneServ, want %s got %s", scn.shardId, info.SerId.SerId)
		return
	}
	scn.lock.Lock()
	old, ok := scn.sceneServs[info.SerId.SerId]
	if ok {
		tilogs.L().Errorf("sceneServsMgr rev new sceneServ addr when old not close, old %s new %s", old, info.PrivateAddr)
		old.stop() // 关闭旧的连接
	}
	s := &sceneServ{
		shardId:      scn.shardId,
		sceneServId:  info.SerId.SerId,
		sceneIP:      info.PrivateAddr,
		scene2Stream: make(map[int32]*scene, sceneDefaultSize),
		gen:          scn.gen,
		wait:         scn.wait,
	}
	scn.sceneServs[info.SerId.SerId] = s
	scn.lock.Unlock()

	scn.servDisCounter++
	if scn.servDisCounter != 1 { // 不等于1表示服务发现Add和Del不是成对出现，有问题
		tilogs.L().Errorf("sceneServsMgr OnAddService servDisCounter %d", scn.servDisCounter)
		scn.servDisCounter = 1
	}

	tilogs.L().Infof("sceneServsMgr OnAddService sceneServ %s, info %+v", info.SerId.SerId, info)
	s.start(0, scn.configSceneIds)
}

// OnDelService 非跨服场景服服务发现
func (scn *sceneServsMgr) OnDelService(serId dis.ServiceId) {
	scn.lock.Lock()
	defer scn.lock.Unlock()
	old, ok := scn.sceneServs[serId.SerId]
	if ok {
		old.stop()
		delete(scn.sceneServs, serId.SerId)
	}

	scn.servDisCounter--
	if scn.servDisCounter != 0 { // 不等于0表示服务发现Add和Del不是成对出现，有问题
		tilogs.L().Errorf("sceneServsMgr OnDelService servDisCounter %d", scn.servDisCounter)
		scn.servDisCounter = 0
	}

	tilogs.L().Infof("sceneServsMgr OnDelService sceneServ %v %v", serId, ok)
}

func (scn *sceneServsMgr) OnChangeExtraInfo(info dis.ServiceInfo) {}

func (scn *sceneServsMgr) stop() {
	scn.lock.Lock()
	if scnServMgr.disMgr != nil {
		scnServMgr.disMgr.Stop()
		scnServMgr.disMgr = nil
	}
	if scn.sceneServs != nil {
		for _, v := range scn.sceneServs {
			v.stop()
		}
		scn.sceneServs = map[string]*sceneServ{}
	}
	scn.lock.Unlock()
	close(scn.quit)
	scn.wait.Wait()
}

// scene服务实例
// 每个scene服务应该有不同的ServId以及地址端口
type sceneServ struct {
	shardId     string
	sceneServId string
	sceneIP     string
	// conn
	conn *grpc.ClientConn

	// client
	client pb.GameSceneClient

	// stream
	scene2Stream map[int32]*scene
	lock         sync.RWMutex
	// close
	closed int32

	gen      GenOnRevSceneServMsg
	wait     *util.WaitGroupWrapper
	onceQuit sync.Once
}

// 对应每个map的实例，主要用于stream管理
type scene struct {
	util.NoCopy
	sendChan chan *pb.SceneMsg
	wait     util.WaitGroupWrapper
	once     sync.Once
	uid      uint64
	ctx      util.CancelCtx
}

func (s *scene) _close() {
	s.once.Do(func() {
		s.ctx.Stop()
		s.wait.Wait()
	})
}

func (s *sceneServ) start(warzoneID int32, sceneIds map[int32]int32) {
	_c, err := s._conn()
	if err != nil {
		tilogs.L().Errorf("sceneServ start _conn fail, sceneServ %v ip %s, err %s", s.sceneServId, s.sceneIP, err.Error())
		return
	}

	s.lock.Lock()
	s.conn = _c
	s.client = pb.NewGameSceneClient(s.conn)
	s.lock.Unlock()

	for sceneId, sceneType := range sceneIds {
		s.StartScene(sceneId, warzoneID, strconv.Itoa(int(sceneId)), strconv.Itoa(int(sceneType)))
	}
}

func (s *sceneServ) StartScene(mapId, warzoneId int32, sceneUniq, sceneType string) {
	scene := &scene{
		sendChan: make(chan *pb.SceneMsg, sceneMsgChanDefaultSize),
		uid:      atomic.AddUint64(&sceneUID, 1),
		ctx:      util.NewCancelCtx(),
	}
	s._addScene(mapId, scene)
	processor := s.gen()
	processor.NotifyPlayerReEnterScene(sceneUniq)

	// stream with retry, 为了应对rpc断开后自动重连的情况
	s.wait.Wrap(func() {
		defer func() {
			s._delScene(mapId, scene.uid)
			scene._close()
			tilogs.L().Infof("sceneServ start stream with retry stop, sceneServ %v, sceneId %s", s.sceneServId, sceneUniq)
		}()

		// 一直重试
		bo := backoff.NewExponentialBackOff()
		// scene支持跨服场景后，需要给xscene留出充裕的时间，处理旧链接断开的问题
		bo.InitialInterval = 1 * time.Second
		bo.MaxElapsedTime = 0

		// 重试直到scene关闭或者gprc conn关闭
		err := backoff.Retry(func() error {
			if s._IsClosed() {
				tilogs.L().Infof("sceneServ %s closed", s.sceneServId)
				return nil
			}

			select {
			case <-scene.ctx.Done():
				// 当场景关闭时, 不需要进行重试了
				tilogs.L().Infof("sceneServ %v, scene %v quit, uid %v", s.sceneServId, mapId, sceneUniq)
				return nil
			default:
			}

			_s, err := s._stream(scene.ctx, mapId, warzoneId, sceneUniq, sceneType)
			if err != nil {
				tilogs.L().Errorf("sceneServ start _stream fail, sceneServ %v, sceneId %s, err %s", s.sceneServId, sceneUniq, err.Error())
				return err
			}

			return s.tryStream(mapId, _s, scene, processor)
		}, bo)
		if err != nil {
			tilogs.L().Errorf("backoff retry failed, err %s", err.Error())
		}

		// for {
		// 	if s._IsClosed() {
		// 		break
		// 	}
		//
		// 	_s, err := s._stream(mapId, warzoneId, sceneUniq, sceneType)
		// 	if err != nil {
		// 		tilogs.L().Errorf("sceneServ start _stream fail, sceneServ %v, sceneId %s, err %s", s.sceneServId, sceneUniq, err.Error())
		// 		continue
		// 	}
		//
		// 	if retry := s._startStream(mapId, _s, scene, processor); retry {
		// 		tilogs.L().Infof("_startStream retry, sceneServ %v, sceneId %s", s.sceneServId, sceneUniq)
		// 		time.Sleep(time.Second + time.Second*time.Duration(rand_pool.Int31n(3))) // 1~4s随机
		// 	} else {
		// 		break
		// 	}
		// }
	})
}

// func (s *sceneServ) _startStream(sceneUniq int32, _s pb.GameScene_OnStreamClient, scene *scene, processor IOnRevSceneServMsg) (retry bool) {
// 	// 向scene发送
// 	scene.wait.Wrap(func() {
// 		tilogs.L().Infof("sceneServ send start, sceneServ %s, sceneId %d", s.sceneServId, sceneUniq)
// 		defer func() {
// 			tilogs.L().Infof("sceneServ send close, sceneServ %s, sceneId %d", s.sceneServId, sceneUniq)
// 		}()
// 		for {
// 			select {
// 			case msg := <-scene.sendChan:
// 				if err := _s.Send(msg); err != nil {
// 					tilogs.L().Errorf("sceneServ Send, shard %s, sceneId %d, ip %s, err %s", s.sceneServId, sceneUniq, s.sceneIP, err.Error())
// 				}
// 			case <-scene.quit:
// 				if err := _s.CloseSend(); err != nil {
// 					tilogs.L().Errorf("sceneServ Send, shard %s, sceneId %d, CloseSend err %s", s.sceneServId, sceneUniq, err.Error())
// 				}
// 				return
// 			}
// 		}
// 	})
//
// 	tilogs.L().Infof("startScene sceneServ %v, sceneId %d", s.sceneServId, sceneUniq)
//
// 	return func() bool {
// 		defer func() {
// 			scene._close()
// 			s._delScene(sceneUniq, scene.uid)
// 			tilogs.L().Infof("sceneServ rev close, sceneServ %s %d", s.sceneServId, sceneUniq)
// 		}()
// 		for {
// 			msg := scene_pool.GetMsg()
// 			err := _s.RecvMsg(msg)
//
// 			// if err != nil && (err == io.EOF ||
// 			// 	strings.Contains(err.Error(), "code = Unavailable desc = transport is closing") ||
// 			// 	strings.Contains(err.Error(), "code = Canceled desc = grpc: the client connection is closing")) {
// 			// 	return false
// 			// }
//
// 			// 通过gprc stream错误判断是否需要重连
// 			// 目前的设计：server正常断开连接，client自身关闭不重连，其他情况进行重连
// 			if err != nil {
// 				if err == io.EOF {
// 					return false
// 				}
//
// 				// 不需要重试的情况
// 				code := status.Code(err)
// 				switch code {
// 				case codes.Canceled, codes.Unavailable, codes.Unimplemented:
// 					tilogs.L().Infof("sceneServ %d Recv stop with grpc code %d, connect break", sceneUniq, code)
// 					return false
// 				}
//
// 				tilogs.L().Errorf("sceneServ Recv, srvID %s, sceneId %d, ip %s, code %d err %s",
// 					s.sceneServId, sceneUniq, s.sceneIP, code, err.Error())
//
// 				s.lock.RLock()
// 				if s.conn.GetState() != connectivity.Shutdown {
// 					s.lock.RUnlock()
// 					// retry
// 					return true
// 				}
// 				s.lock.RUnlock()
//
// 				return false
// 			}
// 			processor.OnRevSceneServMsg(s.sceneServId, sceneUniq, msg)
// 			scene_pool.PutMsg(msg)
// 		}
// 	}()
// }

func (s *sceneServ) tryStream(sceneUniq int32, _s pb.GameScene_OnStreamClient, scene *scene, processor IOnRevSceneServMsg) error {
	quit := make(chan struct{}, 1)

	// 向scene发送
	scene.wait.Wrap(func() {
		tilogs.L().Infof("sceneServ send start, sceneServ %s, sceneId %d", s.sceneServId, sceneUniq)
		defer func() {
			tilogs.L().Infof("sceneServ send close, sceneServ %s, sceneId %d", s.sceneServId, sceneUniq)
		}()
		closeSend := func(reason string) {
			if err := _s.CloseSend(); err != nil {
				tilogs.L().Errorf("sceneServ Send, shard %s, sceneId %d, %s CloseSend err %s", s.sceneServId, sceneUniq, reason, err.Error())
			}
			tilogs.L().Infof("sceneServ Send, shard %s, sceneId %d, %s CloseSend", s.sceneServId, sceneUniq, reason)
		}
		for {
			select {
			case msg := <-scene.sendChan:
				if err := _s.Send(msg); err != nil {
					// 一般错误应该是服务器不可用, 此时会因为读端结束, 关闭quit, 此时写端也可以正常退出
					tilogs.L().Errorf("sceneServ Send, shard %s, sceneId %d, ip %s, err %s", s.sceneServId, sceneUniq, s.sceneIP, err.Error())
				}
			case <-scene.ctx.Done():
				// 服务关闭
				closeSend("scene.quit")

				return
			case <-quit:
				// stream断开
				closeSend("quit")
				return
			}
		}
	})

	tilogs.L().Infof("startScene sceneServ %v, sceneId %d", s.sceneServId, sceneUniq)

	return func() error {
		defer func() {
			close(quit)
			tilogs.L().Infof("sceneServ rev close, sceneServ %s %d", s.sceneServId, sceneUniq)
		}()
		for {
			msg := scene_pool.GetMsg()
			err := _s.RecvMsg(msg)
			// if err != nil && (err == io.EOF ||
			// 	strings.Contains(err.Error(), "code = Unavailable desc = transport is closing") ||
			// 	strings.Contains(err.Error(), "code = Canceled desc = grpc: the client connection is closing")) {
			// 	return false
			// }
			// 通过gprc stream错误判断是否需要重连
			// 目前的设计：server正常断开连接，client自身关闭不重连，其他情况进行重连
			if err != nil {
				tilogs.L().Alarm("sceneServ Recv, srvID %s, sceneId %d, ip %s, err %s", s.sceneServId, sceneUniq, s.sceneIP, err.Error())
				if errors.Is(err, io.EOF) {
					// CloseSend会让Recv获取到一个io.EOF错误
					tilogs.L().Infof("sceneServ %d Recv stop with EOF, connect break", sceneUniq)
					return err
				}

				// 不需要重试的情况
				code := status.Code(err)
				switch code {
				// 当conn连接不上服务器时, 相关的错误码就是 codes.Unavailable
				// 因为启用了grpc自己的重连机制, 所以这种情况应该属于临时的错误
				case codes.Canceled /*codes.Unavailable,*/, codes.Unimplemented:
					tilogs.L().Infof("sceneServ %d Recv stop with grpc code %d, connect break", sceneUniq, code)
					return err
				}

				tilogs.L().Infof("sceneServ Recv, srvID %s, sceneId %d, ip %s, code %d err %s",
					s.sceneServId, sceneUniq, s.sceneIP, code, err.Error())

				s.lock.RLock()
				if s.conn.GetState() != connectivity.Shutdown {
					s.lock.RUnlock()
					// retry
					return err
				}
				s.lock.RUnlock()

				tilogs.L().Infof("sceneServ %d Recv stop with grpc error %v, connect break", sceneUniq, err)
				return status.Error(codes.Unknown, "")

			}
			processor.OnRevSceneServMsg(s.sceneServId, sceneUniq, msg)
			scene_pool.PutMsg(msg)
		}
	}()
}

// func getSceneUniq(mapID string, line int) string {
// 	return fmt.Sprintf("%s:%d", mapID, line)
// }

func getMapBySceneUniq(sceneUniq string) string {
	arr := strings.Split(sceneUniq, ":")
	return arr[0]
}

func (s *sceneServ) send(mapID int32, msg *pb.SceneMsg) bool {
	s.lock.RLock()
	scene, ok := s.scene2Stream[mapID]
	s.lock.RUnlock()
	if !ok {
		tilogs.L().Errorf("sceneServ send s.scene2Stream mapID not found, mapID: %v", mapID)
		return false
	}

	// ctx, cancel := context.WithTimeout(context.Background(), planx.DownTimeOut)
	// defer cancel()
	select {
	case scene.sendChan <- msg:
		return true
	// case <-ctx.Done():
	// 	tilogs.L().Errorf("sceneServ send scene.sendChan timeout, sceneServerID: %v, sceneServerIP: %v, msg: %v", s.sceneServId, s.sceneIP, msg)
	// 	return false
	// }
	default:
		tilogs.L().Errorf("sceneServ send scene.sendChan channel full, sceneServerID: %v, sceneServerIP: %v, msg: %v", s.sceneServId, s.sceneIP, msg)
		return false
	}
}

func (s *sceneServ) sendSync(ctx context.Context, sceneId string, msg *pb.SceneMsg) *pb.SceneMsg {
	if s.client == nil {
		return nil
	}

	msg.SceneId = sceneId
	// conn := pb.NewGameSceneClient(s.conn)

	retMsg, err := s.client.OnCommand(ctx, msg)
	if err != nil {
		tilogs.L().Errorf("scene server oncommand error %v", err)
	}
	return retMsg
}

func (s *sceneServ) sendChangeSceLine(sceneId string, msg *pb.SceneMsg) *pb.SceneMsg {
	msg.SceneId = sceneId

	// 设置访问Scene的超时时间。
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// conn := pb.NewGameSceneClient(s.conn)
	retMsg, err := s.client.OnCommand(ctx, msg)
	if err != nil {
		tilogs.L().Errorf("sceneServ sendChangeSceLine OnCommand error: %v", err)
	}
	return retMsg
}

func (s *sceneServ) stop() {
	atomic.StoreInt32(&s.closed, 1)
	s.lock.Lock()
	defer s.lock.Unlock()
	for k, scene := range s.scene2Stream {
		tilogs.L().Infof("sceneServ stop send close %d", k)
		scene._close()
	}
	if s.conn != nil {
		err := s.conn.Close()
		if err != nil {
			tilogs.L().Errorf("sceneServ %v, %v, %v client stop error %v",
				s.shardId,
				s.sceneServId,
				s.sceneIP, err,
			)
		}
	}
}

func (s *sceneServ) _conn() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, s.sceneIP,
		// grpc.WithInsecure(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(util.UnaryClientRecovery),
		grpc.WithChainStreamInterceptor(util.StreamClientRecovery),
		util.GRPCWithConnectParams(),
		// grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer())),
		// grpc.WithChainStreamInterceptor(otgrpc.OpenTracingStreamClientInterceptor(opentracing.GlobalTracer())),
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (s *sceneServ) _stream(ctx context.Context, mapId, warzoneId int32, sceneUniq, sceneType string) (pb.GameScene_OnStreamClient, error) {
	tilogs.L().Debugf("sceneServ create stream %d zone %d sceneUniq %s sceneType %s",
		mapId, warzoneId, sceneUniq, sceneType)
	md := metadata.Pairs(
		metaKeySceneInfo, strconv.Itoa(int(mapId)),
		metaKeySceneInfo, sceneUniq,
		metaKeySceneInfo, sceneType,
		metaKeyShardInfo, s.shardId,
		metaKeyShardInfo, strconv.Itoa(int(warzoneId)),
	)

	streamCtx := metadata.NewOutgoingContext(ctx, md)

	return s.client.OnStream(streamCtx)
}

func (s *sceneServ) _addScene(sceneUniq int32, scene *scene) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.scene2Stream[sceneUniq] = scene
}

func (s *sceneServ) _delScene(sceneUniq int32, uid uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	sce, ok := s.scene2Stream[sceneUniq]
	if ok && sce.uid == uid {
		delete(s.scene2Stream, sceneUniq)
	}
}

// func (s *sceneServ) ExistsScene(sceneUniq int32) bool {
// 	s.lock.RLock()
// 	defer s.lock.RUnlock()
// 	_, ok := s.scene2Stream[sceneUniq]
// 	return ok
// }

func (s *sceneServ) _IsClosed() bool {
	return atomic.LoadInt32(&s.closed) > 0
}

// AddXSceneSrv 添加跨服场景服
func AddXSceneSrv(info *etcd.XSceneInfo) error {
	if scnServMgr == nil {
		return ErrSceneMgrNotExists
	}

	// 如果cross只想修改xscene结束时间，也会重写key
	// 所以这里需要额外进行一次判断，传入的3个参数是否一致
	scnServMgr.lock.RLock()
	if pre, ok := scnServMgr.sceneServs[info.SerID]; ok {
		if pre.sceneServId == info.SerID &&
			pre.sceneIP == info.Addr &&
			fmt.Sprintf("%v", scnServMgr.xSceneIDs) == fmt.Sprintf("%v", info.SceneIDs) &&
			scnServMgr.warZoneID == int32(info.ZoneID) {
			scnServMgr.lock.RUnlock()
			tilogs.L().Infof("AddXSceneSrv scene %s info same as before", info.SerID)
			return nil
		}
	}
	scnServMgr.lock.RUnlock()

	// 新增
	s := &sceneServ{
		shardId:      scnServMgr.shardId,
		sceneServId:  info.SerID,
		sceneIP:      info.Addr,
		scene2Stream: make(map[int32]*scene, sceneDefaultSize),
		gen:          scnServMgr.gen,
		wait:         new(util.WaitGroupWrapper),
	}

	scnServMgr.lock.Lock()

	// 旧连接
	if scnServMgr.xSceneID != "" {
		pre, ok := scnServMgr.sceneServs[scnServMgr.xSceneID]
		if ok {
			tilogs.L().Warnf("sceneServsMgr rev new xscene addr and close pre, pre %s new %s", pre.sceneIP, s.sceneIP)
			pre.stop() // 关闭旧的连接
			delete(scnServMgr.sceneServs, scnServMgr.xSceneID)
		}
	}

	scnServMgr.sceneServs[info.SerID] = s
	scnServMgr.xSceneID = info.SerID
	scnServMgr.xSceneIDs = info.SceneIDs
	scnServMgr.warZoneID = int32(info.ZoneID)
	// scnServMgr.xSceneIDSuffix = ":" + strconv.Itoa(int(info.ZoneID))
	scnServMgr.lock.Unlock()

	// for sceneID := range info.SceneIDs {
	// 	scnServMgr.xSceneID2srvID.Store(sceneID, info.SceneIDs[sceneID])
	// 	tilogs.L().Debugf("AddXSceneSrv scene %d store as cross scene map", sceneID)
	// }

	tilogs.L().Debugf("sceneServsMgr scene %+v added", info)

	s.start(int32(info.ZoneID), info.SceneIDs)

	return nil
}

// DelXSceneSrv 删除场景服
func DelXSceneSrv() error {
	if scnServMgr == nil {
		return ErrSceneMgrNotExists
	}

	scnServMgr.lock.Lock()
	old, ok := scnServMgr.sceneServs[scnServMgr.xSceneID]
	if ok {
		old.stop()
		delete(scnServMgr.sceneServs, scnServMgr.xSceneID)
		tilogs.L().Debugf("sceneServsMgr sceneServ %v deleted", scnServMgr.xSceneID)
	}
	scnServMgr.xSceneID = ""
	scnServMgr.warZoneID = 0
	// scnServMgr.xSceneIDSuffix = ""
	// oldSceneIDs := scnServMgr.xSceneIDs
	scnServMgr.xSceneIDs = nil
	scnServMgr.lock.Unlock()

	// if oldSceneIDs != nil {
	// 	for sceneID := range oldSceneIDs {
	// 		scnServMgr.xSceneID2srvID.Delete(sceneID)
	// 	}
	// }

	return nil
}

// GetSceneSrvID 返回场景ID对应的场景服ID，以及是否为跨服场景
func GetSceneSrvID(sceneID int32) (string, bool) {
	if scnServMgr == nil {
		return "", false
	}

	_, ok := scnServMgr.configCrossSceneIDs[sceneID]
	if ok {
		return scnServMgr.xSceneID, true
	}
	return scnServMgr.shardId, false
}

// 监听crossx注册的xscene键值，用于控制xscene开启或关闭
func (scn *sceneServsMgr) watchXScene(etcdRoot string, gid, shardId uint, quit chan struct{}, wg *util.WaitGroupWrapper) {
	key := fmt.Sprintf("%s/%d/%s/%d", etcdRoot, gid, etcd.DirXScenes, shardId)

	tilogs.L().Infof("start watch xscene with key %s", key)

	v, rev, err := etcd.GetWithRev(key)
	if err != nil {
		// 有key不存在的可能，这里如果没读到，直接过
		tilogs.L().Warnf("start watch xscene key %s failed, err %s", key, err.Error())
	}

	addXScene := func(info *etcd.XSceneInfo) {
		// 实现xscene重启重连
		// 监听了对应的xscene
		tilogs.L().Infof("watchXScene addXScene for info %+v", info)

		// 已有服务发现且ID不变时，不用动
		if scn.xMgr != nil {
			if info.SerID != scn.xSceneID {
				scn.lock.Lock()
				scn.xMgr.stop()
				scn.xMgr = NewXSceneMgr(etcdRoot, gid, info)
				scn.lock.Unlock()
				go scn.xMgr.loop()
			}
		} else {
			scn.lock.Lock()
			scn.xMgr = NewXSceneMgr(etcdRoot, gid, info)
			scn.lock.Unlock()
			go scn.xMgr.loop()
		}

		err = AddXSceneSrv(info)
		if err != nil {
			tilogs.L().Errorf("add xscene for %+v failed, err %s", info, err.Error())
		}
	}

	if v != "" {
		info := new(etcd.XSceneInfo)
		if err := json.Unmarshal([]byte(v), info); err != nil {
			tilogs.L().Warnf("failed to parse xcene info from %s", v)
		} else {
			addXScene(info)
		}
	}

	etcd.WatchWithRevNoPrevRetry(key, rev, false, quit, wg, func(resp clientv3.WatchResponse) {
		for _, event := range resp.Events {
			if event.Kv == nil || len(event.Kv.Value) == 0 {
				continue
			}

			switch event.Type {
			case clientv3.EventTypePut:
				tilogs.L().Infof("xscene info %s put", string(event.Kv.Value))
				info := new(etcd.XSceneInfo)
				if err := json.Unmarshal(event.Kv.Value, info); err != nil {
					tilogs.L().Warnf("failed to parse xscene info from %s", string(event.Kv.Value))
					continue
				}

				addXScene(info)
			case clientv3.EventTypeDelete:
				tilogs.L().Infof("xscene info deleted")
				if scn.xMgr != nil {
					scn.xMgr.stop()
				}
				err = DelXSceneSrv()
				if err != nil {
					tilogs.L().Errorf("del xscene failed, err %s", err.Error())
				}
			}
		}
	})
}

func TryReplaceSceneIDSuffixWithWarzoneID(sceneID string, warzoneID int32) string {
	warzoneIDStr := strconv.Itoa(int(warzoneID))
	if strings.Count(sceneID, ":") == 2 && !strings.HasSuffix(sceneID, warzoneIDStr) {
		tilogs.L().Infof("TryReplaceSceneIDSuffixWithWarzoneID pre sceneID %s cur warzoneID %d", sceneID, warzoneID)
		sceneID = sceneID[:strings.LastIndex(sceneID, ":")+1] + warzoneIDStr
	}
	return sceneID
}
