package scene

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/msg/scene/pb"
)

const (
	NormalOperate = 0 // 正常运行状态。
	SelfDeleted   = 1 // 已自我删除。
)

const (
	MainCity = 1 // 主城类型的场景。
	XScene   = 5 // 跨服场景
)

const (
	InnerReqTypeNone = iota
	InnerReqTypeNumInfoAndDeleteTrigger
	InnerReqTypeSeizeSeat
	InnerReqTypeIsSeizeSeat
)

var (
	// sceneFullResp = &Resp{curNumInfo: []int{MaxPlayerScene, 0, 0, 0}}
	// numFailResp   = &NumInfoAndDeleteTriggerResp{
	//	NumInfo:       []int{MaxPlayerScene, 0, 0, 0},
	//	DeleteTrigger: false,
	// }
	// seizeSeatFailResp = &SeizeSeatRet{
	//	NumInfo: []int{MaxPlayerScene, 0, 0, 0},
	//	Success: false,
	// }
	respEmptySceneMsg = &pb.SceneMsg{}
)

// 和logicScene绑定的最小场景
type scene struct { // sceneLine
	sceneId      string // 场景线ID。(sceneLineUniq MapID:LineID)
	logicScene   LogicScene
	inChan       chan Req
	innerReqChan chan InnerReq // 内部消息
	// outChan       chan *pb.SceneMsg
	// sceLineChan     chan NumInfoAndDeleteTriggerReq
	// seizeSeatChan   chan SeizeSeat
	// isSeizeSeatChan chan IsSeizeSeat

	tick1sCounter int64 // 每秒计数器
	tick1mCounter int64 // 每分钟计数器

	canSelfDelete bool // 能否执行自我删除。

	wait util.WaitGroupWrapper
	// quit chan struct{}
	once sync.Once

	gid     uint
	shardId uint
	uID     int64

	sm *sceneMgr

	logger tilogs.TiLogger
}

type InnerReq struct {
	ReqType int

	*NumInfoAndDeleteTriggerReq
	*SeizeSeat
	*IsSeizeSeat
}

type NumInfoAndDeleteTriggerReq struct {
	ret chan NumInfoAndDeleteTriggerResp
}

type NumInfoAndDeleteTriggerResp struct {
	NumInfo       []int
	DeleteTrigger bool
}

// 玩家切线占位请求。
type SeizeSeat struct {
	ACID string // 玩家的ACID。
	Ret  chan SeizeSeatRet
}

type SeizeSeatRet struct {
	NumInfo []int // 当前数量信息。
	Success bool  // 占位是否成功。
}

// 玩家切线占位请求。
type IsSeizeSeat struct {
	ACID string // 玩家的ACID。
	Ret  chan bool
}

type Req struct {
	ctx context.Context
	msg *pb.SceneMsg
	ret chan *Resp
}

type Resp struct {
	msg        *pb.SceneMsg
	curNumInfo []int
}

// RoleAct 玩家移动帧相关信息
type RoleAct struct {
	RoleId  string
	ActType int32
	PosX    int32
	PosY    int32
	PosZ    int32
	DirX    int32
	DirZ    int32
}

// GetLastRoleExitSceneTimeStamp 获取最后一次角色退出场景的时间戳。
func (s *scene) GetLastRoleExitSceneTimeStamp() int64 {
	return s.logicScene.GetLastRoleExitSceneTimeStamp()
}

func (s *scene) in(msg *pb.SceneMsg) {
	ctx, cancel := context.WithTimeout(context.Background(), planx.UpTimeOut)
	defer cancel()
	select {
	case s.inChan <- Req{msg: msg}:
	case <-ctx.Done():
		tilogs.L().Errorf("scene in req timeout, sceneID %v, msg %v", s.sceneId, msg)
	}
}

func (s *scene) inSync(ctx context.Context, msg *pb.SceneMsg) *Resp {
	req := Req{
		msg: msg,
		ctx: ctx,
		ret: make(chan *Resp, 1),
	}

	// Req
	select {
	case s.inChan <- req:
	case <-ctx.Done():
		tilogs.L().Errorf("scene inSync req timeout, sceneID: %v, msg: %v", s.sceneId, msg)
		// 场景线处理超时时，应尽量阻止玩家进入此场景线。
		return &Resp{curNumInfo: []int{getMaxPlayerScene(s.logicScene.GetMapId()), 0, 0, 0}}
	}

	// Resp
	select {
	case resp := <-req.ret:
		return resp
	case <-ctx.Done():
		tilogs.L().Errorf("scene inSync resp timeout, sceneID: %v, msg: %v", s.sceneId, msg)
		// 场景线处理超时时，应尽量阻止玩家进入此场景线。
		return &Resp{curNumInfo: []int{getMaxPlayerScene(s.logicScene.GetMapId()), 0, 0, 0}}
	}
}

func (s *scene) inAsync(ctx context.Context, msg *pb.SceneMsg) {
	req := Req{
		msg: msg,
		ctx: ctx,
	}

	// Req
	select {
	case s.inChan <- req:
	case <-ctx.Done():
		tilogs.L().Errorf("scene inAsync req timeout, sceneID: %v, msg: %v", s.sceneId, msg) // 写入超时报错
		return
	}
}

// func (s *scene) out() chan *pb.Req {
//	return s.outChan
// }

// start 场景线启动。
func (s *scene) start() {
	s.logger = tilogs.L().WithUser(s.sceneId)

	s.wait.Wrap(func() {
		defer func() {
			tilogs.L().Infof("sceneID: %v handle stop", s.sceneId)
		}()
		startTimeStamp := time.Now().Unix()

		// 暴露一个协程内接口方法给逻辑场景,方便扩展场景内逻辑
		s.logicScene.OnCreate()

		// 场景线会将玩家发送的消息暂存 FrameMsgSend 时长后下发。
		// tick := time.NewTicker(FrameMsgSend)
		// defer tick.Stop()
		fTicker := timeutil.Timer50MS.After(FrameMsgSend)

		// 场景线每隔 SelfCheckRecycleInterval 时间检查是否可以执行自我删除和是否有玩家保活超时。
		// tickRecycle := time.NewTicker(SelfCheckRecycleInterval)
		// tickRecycle := timeutil.TimerMin.After(SelfCheckRecycleInterval)
		// defer tickRecycle.Stop()

		// 场景线会每隔 SeizeSeatCancelCheckInterval 时间检查是否有玩家霸座并作出相应的处理。
		// tickSeizeSeat := time.NewTicker(SeizeSeatCancelCheckInterval)
		// tickSeizeSeat := timeutil.TimerSec.After(SeizeSeatCancelCheckInterval)
		// defer tickSeizeSeat.Stop()

		for {
			select {
			case r, ok := <-s.inChan:
				if !ok {
					tilogs.L().Infof("scene rev close, sceneId %s", s.sceneId)
					return
				}
				// 场景线收到请求消息。
				func(_r Req) {
					// l := tilogs.L().WithUser(s.sceneId)
					defer tilogs.PanicCatcherWithInfo(s.logger, "Scene Panic, sceneID: %v, msgId: %v, msgType: %v",
						s.sceneId, _r.msg.MsgId, _r.msg.MsgType)
					timeNow := time.Now()

					// tracing
					if _r.ctx != nil && opentracing.SpanFromContext(_r.ctx) != nil {
						span, _ := opentracing.StartSpanFromContext(_r.ctx, "RevSceneReq-"+strconv.Itoa(int(_r.msg.MsgId)))
						defer span.Finish()
					}

					msg := s.logicScene.HandleSceneMsg(_r.msg) // 场景线的逻辑内核处理消息请求。

					// ret := &Resp{
					// 	msg:        s.logicScene.HandleSceneMsg(_r.msg), // 场景线的逻辑内核处理消息请求。
					// 	curNumInfo: s.logicScene.GetSceneNum(),
					// }
					//

					if msg != nil && _r.ret != nil {
						select {
						case _r.ret <- &Resp{
							msg:        msg,
							curNumInfo: s.logicScene.GetSceneNum(),
						}:
						default:
							tilogs.L().Errorf("Scene intent ret msg error, msg: %v", _r.msg)
						}
					}

					// 处理时间过长的请求需要被记录。
					if time.Now().Sub(timeNow) > 100*time.Millisecond {
						tilogs.L().Warnf("Scene s.inChan cost: %v too long, msgID: %v, msgType: %v", time.Now().Sub(timeNow), _r.msg.GetMsgId(), _r.msg.GetMsgType())
						k4stat := strconv.Itoa(int(_r.msg.MsgId)) + "req"
						recordTime(graphiteName(s.sm.metricsPrefix, k4stat), k4stat, timeNow.UnixNano())
					}
				}(r)
			case r := <-s.innerReqChan:
				// 内部消息
				switch r.ReqType {
				case InnerReqTypeNumInfoAndDeleteTrigger:
					func(_r NumInfoAndDeleteTriggerReq) {
						// l := tilogs.L().WithUser(s.sceneId)
						defer tilogs.PanicCatcherWithInfo(s.logger, "scene sceLineChan NumInfoAndDeleteTriggerReq panic, sceneID %v", s.sceneId)
						timeNow := time.Now()

						select {
						case _r.ret <- NumInfoAndDeleteTriggerResp{
							NumInfo:       s.logicScene.GetSceneNum(),
							DeleteTrigger: s.CheckMainCitySceneRecycled(startTimeStamp),
						}:
						default:
							tilogs.L().Errorf("push ret msg error")
						}

						// 处理时间过长的请求需要被记录。
						if ct := time.Now().Sub(timeNow); ct > 100*time.Millisecond {
							tilogs.L().Warnf("Scene s.sceLineChan cost too long %v", ct)
						}
					}(*r.NumInfoAndDeleteTriggerReq)
				case InnerReqTypeSeizeSeat:
					// 场景线收到预先占座的同步请求消息。
					func(ss SeizeSeat) {
						// logger := tilogs.L().WithUser(s.sceneId)
						defer tilogs.PanicCatcherWithInfo(s.logger, "SeizeSeat Handle Panic, sceneID: %v", s.sceneId)
						timeNow := time.Now()

						select {
						case ss.Ret <- SeizeSeatRet{
							NumInfo: s.logicScene.GetSceneNum(),         // 获取人数的信息。
							Success: s.logicScene.TrySeizeSeat(ss.ACID), // 玩家尝试预占位。
						}:
						default:
							tilogs.L().Errorf("TrySeizeSeat intent ret error")
						}

						// 处理时间过长的请求需要被记录。
						if time.Now().Sub(timeNow) > 100*time.Millisecond {
							tilogs.L().Warnf("Scene s.seizeSeatChan cost too long")
						}
					}(*r.SeizeSeat)
				case InnerReqTypeIsSeizeSeat:
					// 场景线收到预先占座的同步请求消息。
					func(ss IsSeizeSeat) {
						// logger := tilogs.L().WithUser(s.sceneId)
						defer tilogs.PanicCatcherWithInfo(s.logger, "IsSeizeSeat Handle Panic, sceneID: %v", s.sceneId)
						timeNow := time.Now()

						select {
						case ss.Ret <- s.logicScene.IsSeizeSeat(ss.ACID):
						default:
							tilogs.L().Errorf("IsSeizeSeat intent ret error")
						}

						// 处理时间过长的请求需要被记录。
						if time.Now().Sub(timeNow) > 100*time.Millisecond {
							tilogs.L().Warnf("Scene s.IsSeizeSeatChan cost too long")
						}
					}(*r.IsSeizeSeat)
				default:
					s.logger.Errorf("invalid inner request %v", r)
				}
			// case r := <-s.sceLineChan:
			// 	func(_r NumInfoAndDeleteTriggerReq) {
			// 		// l := tilogs.L().WithUser(s.sceneId)
			// 		defer tilogs.PanicCatcherWithInfo(s.logger, "scene sceLineChan NumInfoAndDeleteTriggerReq panic, sceneID %v", s.sceneId)
			// 		timeNow := time.Now()
			//
			// 		select {
			// 		case _r.ret <- NumInfoAndDeleteTriggerResp{
			// 			NumInfo:       s.logicScene.GetSceneNum(),
			// 			DeleteTrigger: s.CheckMainCitySceneRecycled(startTimeStamp),
			// 		}:
			// 		default:
			// 			tilogs.L().Errorf("push ret msg error")
			// 		}
			//
			// 		// 处理时间过长的请求需要被记录。
			// 		if ct := time.Now().Sub(timeNow); ct > 100*time.Millisecond {
			// 			tilogs.L().Warnf("Scene s.sceLineChan cost too long %v", ct)
			// 		}
			// 	}(r)
			// case req := <-s.seizeSeatChan:
			// 	// 场景线收到预先占座的同步请求消息。
			// 	func(ss SeizeSeat) {
			// 		// logger := tilogs.L().WithUser(s.sceneId)
			// 		defer tilogs.PanicCatcherWithInfo(s.logger, "SeizeSeat Handle Panic, sceneID: %v", s.sceneId)
			// 		timeNow := time.Now()
			//
			// 		select {
			// 		case ss.Ret <- SeizeSeatRet{
			// 			NumInfo: s.logicScene.GetSceneNum(),         // 获取人数的信息。
			// 			Success: s.logicScene.TrySeizeSeat(ss.ACID), // 玩家尝试预占位。
			// 		}:
			// 		default:
			// 			tilogs.L().Errorf("TrySeizeSeat intent ret error")
			// 		}
			//
			// 		// 处理时间过长的请求需要被记录。
			// 		if time.Now().Sub(timeNow) > 100*time.Millisecond {
			// 			tilogs.L().Warnf("Scene s.seizeSeatChan cost too long")
			// 		}
			// 	}(req)
			// case req := <-s.isSeizeSeatChan:
			// 	// 场景线收到预先占座的同步请求消息。
			// 	func(ss IsSeizeSeat) {
			// 		// logger := tilogs.L().WithUser(s.sceneId)
			// 		defer tilogs.PanicCatcherWithInfo(s.logger, "IsSeizeSeat Handle Panic, sceneID: %v", s.sceneId)
			// 		timeNow := time.Now()
			//
			// 		select {
			// 		case ss.Ret <- s.logicScene.IsSeizeSeat(ss.ACID):
			// 		default:
			// 			tilogs.L().Errorf("IsSeizeSeat intent ret error")
			// 		}
			//
			// 		// 处理时间过长的请求需要被记录。
			// 		if time.Now().Sub(timeNow) > 100*time.Millisecond {
			// 			tilogs.L().Warnf("Scene s.IsSeizeSeatChan cost too long")
			// 		}
			// 	}(req)
			case <-fTicker:
				s.logicScene.Tick()
				s.Tick1s()
				s.Tick1m()
				fTicker = timeutil.Timer50MS.After(FrameMsgSend)
				// case <-tickRecycle:
				// 	// 检查动态类型场景线是否可以执行自我删除。
				// 	s.CheckDynamicSceneRecycled()
				//
				// 	// 检查场景线内是否有玩家保活超时，需移除。
				// 	s.CleanUpKeepAliveTimeout()
				//
				// 	// 清理场景线内空的位置信息
				// 	s.CleanUpScreenIdx()
				// 	tickRecycle = timeutil.TimerMin.After(SelfCheckRecycleInterval)
				// case <-tickSeizeSeat:
				// 	s.CleanUpForciblyOccupySeat()
				// 	tickSeizeSeat = timeutil.TimerSec.After(SeizeSeatCancelCheckInterval)
				// case <-s.quit:
				// 	tilogs.L().Infof("scene rev close, sceneId %s", s.sceneId)
				// 	return
			}
		}
	})
}

func (s *scene) Tick1s() {
	s.tick1sCounter++

	// 每1.15秒执行一次
	// 使用质数避免1s和1m有太多公倍数
	if s.tick1sCounter >= SelfCheckRecycleIntervalTickInterval {
		s.tick1sCounter = 0

		s.CleanUpForciblyOccupySeat()
	}
}

func (s *scene) Tick1m() {
	s.tick1mCounter++

	// 每1分钟执行一次
	// 使用质数避免1s和1m有太多公倍数
	if s.tick1mCounter >= SeizeSeatCancelCheckTickInterval {
		s.tick1mCounter = 0

		// 检查动态类型场景线是否可以执行自我删除。
		s.CheckDynamicSceneRecycled()

		// 检查场景线内是否有玩家保活超时，需移除。
		s.CleanUpKeepAliveTimeout()

		// 清理场景线内空的位置信息
		s.CleanUpScreenIdx()
	}
}

func (s *scene) stop() {
	s.once.Do(func() {
		// close(s.quit)
		close(s.inChan)
		s.wait.Wait()
	})
}

func (s *scene) selfStopWithoutWait() {
	s.once.Do(func() {
		// close(s.quit)
		close(s.inChan)
		tilogs.L().Infof("scene selfStopWithoutWait sceneID: %v", s.sceneId)
	})
}

// func (s *scene) getGoneChan() <-chan struct{} {
// 	return s.quit
// }

// getSceneNumInfoAndDeleteTrigger 获取场景线当前的数量信息和是否触发删除的标记位。
//
// Note: int[0]-场景线当前玩家数量; int[1]-场景线当前精英怪数量;
// Note: int[2]-场景线当前普通怪数量; int[3]-场景线当前预占位玩家数量。
func (s *scene) getSceneNumInfoAndDeleteTrigger() NumInfoAndDeleteTriggerResp {
	requestMsg := NumInfoAndDeleteTriggerReq{}
	retChan := make(chan NumInfoAndDeleteTriggerResp, 1)
	requestMsg.ret = retChan

	// Req
	ctx, cancel := context.WithTimeout(context.Background(), planx.UpTimeOut)
	defer cancel()
	select {
	// case s.sceLineChan <- requestMsg:
	case s.innerReqChan <- InnerReq{
		ReqType:                    InnerReqTypeNumInfoAndDeleteTrigger,
		NumInfoAndDeleteTriggerReq: &requestMsg,
	}:
	case <-ctx.Done():
		tilogs.L().Errorf("getSceneNumInfoAndDeleteTrigger req timeout, sceneID %v", s.sceneId)

		/**
		若查询失败
		1.返回最大人数，防止场景被删除或者进入新玩家。
		2.返回没有删除触发，防止SceneLineManager的信息被清掉。
		*/
		return NumInfoAndDeleteTriggerResp{
			NumInfo:       []int{getMaxPlayerScene(s.logicScene.GetMapId()), 0, 0, 0},
			DeleteTrigger: false,
		}
	}

	// Resp
	select {
	case retMsg := <-retChan:
		return retMsg
	case <-ctx.Done():
		tilogs.L().Errorf("getSceneNumInfoAndDeleteTrigger resp timeout, sceneID %v", s.sceneId)
	}

	/**
	若查询失败
	1.返回最大人数，防止场景被删除或者进入新玩家。
	2.返回没有删除触发，防止SceneLineManager的信息被清掉。
	*/
	return NumInfoAndDeleteTriggerResp{
		NumInfo:       []int{getMaxPlayerScene(s.logicScene.GetMapId()), 0, 0, 0},
		DeleteTrigger: false,
	}
}

// trySeizeSeat 尝试为玩家在目标场景线占位（占位成功后，确保玩家可以成功进入目标场景线）。
func (s *scene) trySeizeSeat(acid string) SeizeSeatRet {
	req := SeizeSeat{
		ACID: acid,
		Ret:  make(chan SeizeSeatRet, 1),
	}

	// 向目标场景线发起占位请求。
	ctx, cancel := context.WithTimeout(context.Background(), planx.UpTimeOut)
	defer cancel()
	select {
	// case s.seizeSeatChan <- req:
	case s.innerReqChan <- InnerReq{
		ReqType:   InnerReqTypeSeizeSeat,
		SeizeSeat: &req,
	}:
	case <-ctx.Done():
		tilogs.L().Errorf("trySeizeSeat req timeout, acid: %v, sceneID: %v", acid, s.sceneId)
		return SeizeSeatRet{
			NumInfo: []int{getMaxPlayerScene(s.logicScene.GetMapId()), 0, 0, 0},
			Success: false,
		}
	}

	// 等待目标场景线的反馈信息。
	select {
	case resp := <-req.Ret:
		return resp
	case <-ctx.Done():
		tilogs.L().Errorf("trySeizeSeat resp timeout, acid: %v, sceneID: %v", acid, s.sceneId)
		return SeizeSeatRet{
			NumInfo: []int{getMaxPlayerScene(s.logicScene.GetMapId()), 0, 0, 0},
			Success: false,
		}
	}
}

// trySeizeSeat 尝试为玩家在目标场景线占位（占位成功后，确保玩家可以成功进入目标场景线）。
func (s *scene) IsSeizeSeat(acid string) bool {
	req := IsSeizeSeat{
		ACID: acid,
		Ret:  make(chan bool, 1),
	}

	// 向目标场景线发起占位请求。
	ctx, cancel := context.WithTimeout(context.Background(), planx.UpTimeOut)
	defer cancel()
	select {
	// case s.isSeizeSeatChan <- req:
	case s.innerReqChan <- InnerReq{
		ReqType:     InnerReqTypeIsSeizeSeat,
		IsSeizeSeat: &req,
	}:
	case <-ctx.Done():
		tilogs.L().Errorf("IsSeizeSeat req timeout, acid: %v, sceneID: %v", acid, s.sceneId)
		return false
	}

	// 等待目标场景线的反馈信息。
	select {
	case resp := <-req.Ret:
		return resp
	case <-ctx.Done():
		tilogs.L().Errorf("IsSeizeSeat resp timeout, acid: %v, sceneID: %v", acid, s.sceneId)
		return false
	}
}

// CheckMainCitySceneRecycled 检查主城类型场景线能否自我删除，如果可以则执行。
func (s *scene) CheckMainCitySceneRecycled(lineStartTime int64) bool {
	if s.logicScene.GetSceneType() != MainCity && s.logicScene.GetSceneType() != XScene {
		return false
	}

	nowTimeStamp := time.Now().Unix()

	// 主城类型场景线自我删除检查。
	if !s.canSelfDelete {
		// 检查场景线是否允许自我删除。
		return false
	}

	// 如果当前场景线内仍有玩家，或有预先占座的玩家则不做删除。
	sceneNumInfo := s.logicScene.GetSceneNum()
	if sceneNumInfo[IdxLinePlayerNum]+sceneNumInfo[IdxLinePlayerSeizeSeat] > 0 {
		return false
	}

	// 如果当前场景线的存在时间还未满1分钟不作删除。
	if nowTimeStamp-lineStartTime <= NewSceneMaxExist {
		return false
	}

	// 如果当前场景线举例最后一次角色离开还未满1分钟不做删除。
	if nowTimeStamp-s.GetLastRoleExitSceneTimeStamp() <= LastRoleLeave {
		return false
	}

	return true
}

// CheckDynamicSceneRecycled 检查动态类型场景能否自我删除，如果可以则执行。
func (s *scene) CheckDynamicSceneRecycled() {
	if s.logicScene.GetSceneType() == MainCity || s.logicScene.GetSceneType() == XScene {
		return
	}

	// 如果当前场景线内仍有玩家则不做删除。
	num := s.logicScene.GetSceneNum()
	if num[0] > 0 {
		return
	}

	// 从场景管理中将自己移除（场景管理会将场景线终止）。
	s.sm.sceneSelfDel(s.sceneId)
}

func (s *scene) CleanUpForciblyOccupySeat() {
	s.logicScene.CleanUpForciblyOccupySeat()
}

func (s *scene) CleanUpKeepAliveTimeout() {
	s.logicScene.CleanUpKeepAliveTimeout()
}

func (s *scene) CleanUpScreenIdx() {
	s.logicScene.CleanUpScreens()
}
