package scene

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx/scene_pool"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/msg/scene/pb"
)

// 某个mapID的所有分线的集合，以uID为唯一主键区分
// 同个mapID关联不同shard在这层实现
type sceneLineManager struct {
	uID       int64
	mapID     int32
	zoneID    int32
	sceneType string
	lineUniq  []sceneLineNum
	// uniq -> num
	linePlayerNum map[string]int

	lineChan chan *ParamSceneLine

	// 该mapID场景统一回game shard的chan
	outChan chan *pb.SceneMsg

	// 跨服场景返回不同game shard的接口
	// 同时会传递指针到下层logicScene
	// [uint32]chan *pb.SceneMsg
	outChans *sync.Map

	// 为兼容之前设计，这里保留上层操作scene
	sm   *sceneMgr
	quit chan bool
	w    util.WaitGroupWrapper
	lock sync.RWMutex
	once sync.Once

	AllLineNpc []*pb.AllLineNpcInfo // 全图npc信息缓存

	logger tilogs.TiLogger
}

type sceneLineNum struct {
	sceneLineUniq string
	lineNum       int // 序号
	startTime     int64
}

type ParamSceneLine struct {
	msg *pb.SceneMsg
	ctx context.Context
	ret chan *pb.SceneMsg
}

// type ParamSceneLineMsg struct {
// 	msg *pb.SceneMsg
// 	Ret chan *pb.SceneMsg
// }

// type RetSceneLine struct {
// 	msg *pb.SceneMsg
// }

func (sl *sceneLineManager) start() {
	sl.w.Wrap(sl.logic)
}

func (sl *sceneLineManager) stop() {
	sl.once.Do(func() {
		close(sl.quit)
		sl.w.Wait()
	})
}

func (sl *sceneLineManager) logic() {
	defer tilogs.L().Infof("sceneLineManager %v logic close", sl.uID)

	sl.logger = tilogs.L().WithUser(strconv.Itoa(int(sl.uID)))

	// checkTicker := time.NewTicker(5 * time.Second)
	checkTicker := timeutil.TimerSec.After(5 * time.Second)
	// defer checkTicker.Stop()

	for {
		select {
		case param := <-sl.lineChan:
			func() {
				// l := tilogs.L().WithUser(strconv.Itoa(int(sl.uID)))
				defer tilogs.PanicCatcherWithInfo(sl.logger, "sceneLineManager %v lineChan panic", sl.uID)

				timeNow := time.Now()
				ret := sl.dealLogic(param)
				// 处理时间过长的请求需要被记录。
				if time.Now().Sub(timeNow) > 200*time.Millisecond {
					tilogs.L().Warnf("sceneLineManager sl.lineChan cost too long, msgID: %v, param: %v, costTime: %v", param.msg.GetMsgId(), param.msg, time.Now().Sub(timeNow))
				}
				if time.Now().Sub(timeNow) > 100*time.Millisecond {
					k4stat := strconv.Itoa(int(param.msg.MsgId)) + "SlmReq"
					recordTime(graphiteName(sl.sm.metricsPrefix, k4stat), k4stat, timeNow.UnixNano())
				}
				if ret != nil && param.ret != nil {
					select {
					case param.ret <- ret:
					default:
						tilogs.L().Errorf("sceneLineManager %v, intent ret param failed", sl.uID)
					}
				}
			}()
		case <-checkTicker:
			func() {
				// l := tilogs.L().WithUser(fmt.Sprintf("%d", sl.uID))
				defer tilogs.PanicCatcherWithInfo(sl.logger, "sceneLineManager %v, timer tick panic", sl.uID)

				// 从管理的场景线中更新每条场景线的当前人数（场景线中的玩家+预占位玩家）。
				// 这一同步请求可能触发场景线的自我删除，在触发后会作出相应的清理操作。
				sl.updateSceneNumInfo()

				// 检查是否需要预开场景线（当前场景线负载较高时，预开场景线）。
				sl.sceneCheck()
			}()
			checkTicker = timeutil.TimerSec.After(5 * time.Second)
		case <-sl.quit:
			// 场景线管理对象终止，被管理场景线均应终止。
			for sceneUniqID := range sl.linePlayerNum {
				tilogs.L().Infof("sceneLineManager logic <-sl.quit delLine sceneUniqID: %v", sceneUniqID)
				sl.delLine(sceneUniqID)
			}
			return
		}
	}
}

// updateSceneNumInfo 更新场景线的人数信息。
//
// Note: 人数信息——场景线内玩家和场景线内预占位玩家。
// Note: 这一同步请求可能触发场景线的自我删除，在触发后会作出相应的清理操作。
func (sl *sceneLineManager) updateSceneNumInfo() {
	for index := 0; index < len(sl.lineUniq); index++ {
		sceneLineUniq := sl.lineUniq[index].sceneLineUniq
		scene := sl.sm.getScene(sceneLineUniq)
		if scene == nil {
			tilogs.L().Errorf("updateSceneNumInfo sl.sm.getScene scene %v is nil", sceneLineUniq)
			continue
		}

		/*
			获取目标场景线的数量信息（0-场景线中的玩家; 3-场景线中预占位的玩家）。
			获取数量信息同步请求时，如果触发了目标场景线的自我删除，需要做出相应的清理操作。
		*/
		info := scene.getSceneNumInfoAndDeleteTrigger()
		nums := info.NumInfo
		if info.DeleteTrigger {
			// 触发了目标场景线的自我删除。
			tilogs.L().Infof("sceneLineManager updateSceneNumInfo scene.getSceneNumInfoAndDeleteTrigger trigger delete, sceneUniqID: %v", sceneLineUniq)

			// 终止场景线。
			// tilogs.L().Infof("sceneLineManager updateSceneNumInfo DeleteTrigger delLine sceneLineUniq: %v", sceneLineUniq)
			sl.delLine(sceneLineUniq)

			// 监控统计（不包含预占位玩家的数量）。上线后删除
			// allmetrics.UpdateSceneMetric(sceneLineUniq, int64(0))

			index--
		} else {
			// 没有触发目标场景线的自我删除。
			sl.linePlayerNum[sceneLineUniq] = nums[IdxLinePlayerNum] + nums[IdxLinePlayerSeizeSeat]

			// 监控统计（不包含预占位玩家的数量）。上线后删除
			// allmetrics.UpdateSceneMetric(sceneLineUniq, int64(nums[0]))
		}
	}
}

func (sl *sceneLineManager) dealLogic(param *ParamSceneLine) *pb.SceneMsg {
	if param.ctx != nil && opentracing.SpanFromContext(param.ctx) != nil {
		span, _ := opentracing.StartSpanFromContext(param.ctx, "dealLogic-"+pb.MsgType_name[int32(param.msg.GetMsgType())])
		defer span.Finish()
	}

	var msg *pb.SceneMsg

	switch param.msg.GetMsgType() {
	case pb.MsgType_EnterScene:
		// 玩家意图进入场景线处理。
		msg = sl.enterScene(param.ctx, param.msg.GetSceneId(), param.msg, false)
	case pb.MsgType_ForceEnterScene:
		// 玩家意图强制进入场景线处理。
		msg = sl.enterScene(param.ctx, param.msg.GetSceneId(), param.msg, true)
	case pb.MsgType_GetSceneLine:
		// 玩家意图获取各场景分线的数量信息。
		msg = sl.getSceneLine()
	case pb.MsgType_GetSceneLineNotRealTime:
		// 玩家意图获取各场景分线的数量信息（非实时）。
		msg = sl.getSceneLineNotRealTime()
	case pb.MsgType_ChangeSceneLine:
		// 玩家意图切线前预占位。
		msg = sl.changeSceneLine(param.msg.GetSceneId(), param.msg.GetAcids()[0])
	case pb.MsgType_NpcEnterScene:
		msg = sl.npcEnterScene(param.msg)
	case pb.MsgType_NpcLeaveScene:
		msg = sl.npcLeaveScene(param.msg)
	case pb.MsgType_GetNpcList:
		msg = sl.GetNpcList()
	default:
		msg = sl.inSyncReqScene(param.ctx, param.msg.GetSceneId(), param.msg)
	}

	return msg
}

func (sl *sceneLineManager) sceneCheck() {
	sl.sceneCheckAdd()
}

func (sl *sceneLineManager) delLine(sceneUniqID string) {
	// 通知相应的场景线终止执行。
	sl.sm.delScene(sceneUniqID)

	// 从 sl.lineUniq 和 sl.linePlayerNum 中删除相应的管理项。
	sl.delLineUniq(sceneUniqID)
	delete(sl.linePlayerNum, sceneUniqID)
}

func (sl *sceneLineManager) delLineUniq(sceneID string) {
	indexFlag := -1
	for index, lineInfo := range sl.lineUniq {
		if lineInfo.sceneLineUniq == sceneID {
			indexFlag = index
			break
		}
	}

	if indexFlag != -1 {
		sl.lineUniq = append(sl.lineUniq[:indexFlag], sl.lineUniq[indexFlag+1:]...)
	}
}

func (sl *sceneLineManager) sceneCheckAdd() string {
	// 检查当前场景线负载，判断是否需要预开场景线。
	needNew := true
	for _, playerNum := range sl.linePlayerNum {
		if playerNum < getNeedNewScene(sl.mapID) {
			needNew = false
			break
		}
	}
	if !needNew {
		return ""
	}

	// 需要开启新线程
	line := StartSceneLine
	for _, lineUniq := range sl.lineUniq {
		if lineUniq.lineNum > line {
			break
		}
		line++
	}
	sceneUniq := getSceneUniq(sl.mapID, line, sl.zoneID)

	// deep copy alllineNpc
	allLineNpc := make(map[string]*pb.AllLineNpcInfo, len(sl.AllLineNpc))
	for _, value := range sl.AllLineNpc {
		allLineNpc[value.GetUniqId()] = &(*value)
	}
	tilogs.L().Infof("sceneCheckAdd mapId:%v npcs :%v ", sl.uID, len(sl.AllLineNpc))
	// 创建场景
	if !sl.sm.addScene(sl.mapID, sl.zoneID, sceneUniq, sl.sceneType, sl.outChan, sl.outChans, AllowSelfDelete, allLineNpc) {
		return sceneUniq // 不应该出现，若出现了，说明此分线已存在，则直接返回此分线
	}

	if line > len(sl.lineUniq) {
		newLineUniq := sceneLineNum{
			sceneLineUniq: sceneUniq,
			lineNum:       line,
			startTime:     time.Now().Unix(),
		}
		sl.lineUniq = append(sl.lineUniq, newLineUniq)
	} else {
		insertRank := []sceneLineNum{{
			sceneLineUniq: sceneUniq,
			lineNum:       line,
			startTime:     time.Now().Unix(),
		}}
		newlineUniq := make([]sceneLineNum, len(sl.lineUniq)+1)
		tmpIndex := copy(newlineUniq, sl.lineUniq[:line-1])
		tmpIndex += copy(newlineUniq[tmpIndex:], insertRank)
		copy(newlineUniq[tmpIndex:], sl.lineUniq[line-1:])
		sl.lineUniq = newlineUniq
	}

	sl.linePlayerNum[sceneUniq] = 0
	return sceneUniq
}

func (sl *sceneLineManager) getSceneOutChan() chan *pb.SceneMsg {
	return sl.outChan
}

// inSync 向sceneLineManager发起同步处理请求。
func (sl *sceneLineManager) inSync(ctx context.Context, msg *pb.SceneMsg) *pb.SceneMsg {
	requestMsg := &ParamSceneLine{
		msg: msg,
		ctx: ctx,
		ret: make(chan *pb.SceneMsg, 1),
	}

	// Req
	select {
	case sl.lineChan <- requestMsg:
	default:
		tilogs.L().Errorf("sceneLineManager inSync Req channel full, uID: %v, sceneID: %v, msg: %v", sl.uID, msg.GetSceneId(), msg)
		return nil
	}

	// Resp
	select {
	case retMsg := <-requestMsg.ret:
		return retMsg
	case <-ctx.Done():
		tilogs.L().Errorf("sceneLineManager inSync Resp timeout, uID: %v, sceneID: %v, msg: %v", sl.uID, msg.GetSceneId(), msg)
		return nil
	}
}

func (sl *sceneLineManager) enterScene(ctx context.Context, sceneID string, msg *pb.SceneMsg, force bool) *pb.SceneMsg {
	msgCopy := proto.Clone(msg).(*pb.SceneMsg)
	if lineNum, ok := sl.linePlayerNum[sceneID]; ok {
		if force || lineNum < getMaxPlayerScene(sl.mapID) {
			// 强制进入目标场景线或目标场景线未满。
			return sl.inSyncReqScene(ctx, sceneID, msgCopy)
		} else {
			// 检查是否有占位
			scen := sl.sm.getScene(sceneID)
			if scen != nil {
				has := scen.IsSeizeSeat(msg.Acids[0])
				if has {
					tilogs.L().Debugf("enterScene by SeizeSeat acid:%v sceneID:%v", msg.Acids, sceneID)
					return sl.inSyncReqScene(ctx, sceneID, msgCopy)
				}
			} else {
				tilogs.L().Infof("enterScene but not find scene acid:%v sceneID:%v", msg.Acids, sceneID)
			}
		}
	} else {
		tilogs.L().Infof("enterScene but not find linePlayerNum acid:%v sceneID:%v", msg.Acids, sceneID)
	}

	// 目标场景线已满，需要找其他场景线（按线号从小到大）。
	for _, lineInfo := range sl.lineUniq {
		if sl.linePlayerNum[lineInfo.sceneLineUniq] < getMaxPlayerScene(sl.mapID) {
			msgCopy.SceneId = lineInfo.sceneLineUniq
			return sl.inSyncReqScene(ctx, lineInfo.sceneLineUniq, msgCopy)
		}
	}

	// 所有场景线已满，创建新的场景线。
	newSceneLineUniq := sl.sceneCheckAdd()
	msgCopy.SceneId = newSceneLineUniq

	// 怀疑有低概率场景线和场景线玩家数量不一致，导致新场景线为空
	// 加日志排查
	if newSceneLineUniq == "" {
		sl.logger.Errorf("enterScene newSceneLineUniq is empty, acid:%v sceneID:%v, getMaxPlayerScene(sl.mapID):%d, getNeedNewScene(sl.mapID):%d, len of sl.lineUniq:%d, len of sl.linePlayerNum:%d",
			msg.Acids, sceneID, getMaxPlayerScene(sl.mapID), getNeedNewScene(sl.mapID), len(sl.lineUniq), len(sl.linePlayerNum))
		sl.logger.Errorf("sl.lineUniq:%v, sl.linePlayerNum:%v", sl.lineUniq, sl.linePlayerNum)
	}

	return sl.inSyncReqScene(ctx, newSceneLineUniq, msgCopy)
}

func (sl *sceneLineManager) removeDeadNpc(uniqId string) {
	ts := timeutil.Now().Unix()
	for i := 0; i < len(sl.AllLineNpc); i++ {
		npc := sl.AllLineNpc[i]
		if npc.GetUniqId() == uniqId || (npc.GetDeadTime() > 0 && ts > npc.GetDeadTime()) {
			sl.AllLineNpc = append(sl.AllLineNpc[:i], sl.AllLineNpc[i+1:]...)
		}
	}
}

// 所有线种npc
func (sl *sceneLineManager) npcEnterScene(msg *pb.SceneMsg) *pb.SceneMsg {

	if msg.IsAllLineNpc {
		sl.AllLineNpc = append(sl.AllLineNpc, msg.GetAllLineNpc())
		sl.removeDeadNpc("")
		tilogs.L().Debugf("npcEnterScene %v", msg.GetAllLineNpc())
		for _, lineInfo := range sl.lineUniq {
			sl.activeNpc(lineInfo.sceneLineUniq, msg)
		}
	} else {
		sl.activeNpc(msg.GetSceneId(), msg)
	}

	return respEmptySceneMsg
}

// 获取npclist
func (sl *sceneLineManager) GetNpcList() *pb.SceneMsg {
	res := scene_pool.GetMsg()
	res.AllLineNpcList = make([]string, 0, len(sl.AllLineNpc))

	sl.removeDeadNpc("")
	// 删除超时的npc
	for _, npcInfo := range sl.AllLineNpc {
		res.AllLineNpcList = append(res.AllLineNpcList, npcInfo.GetUniqId())
	}
	return res
}

// 所有线删npc
func (sl *sceneLineManager) npcLeaveScene(msg *pb.SceneMsg) *pb.SceneMsg {
	tilogs.L().Debugf("npcLeaveScene %v", msg.NpcUniqId)
	if msg.IsAllLineNpc {
		sl.removeDeadNpc(msg.NpcUniqId)
		for _, lineInfo := range sl.lineUniq {
			// msg.SceneId = lineInfo.sceneLineUniq
			sl.deleteNpc(lineInfo.sceneLineUniq, msg)
		}
	} else {
		sl.deleteNpc(msg.SceneId, msg)
	}
	return respEmptySceneMsg
}

func (sl *sceneLineManager) activeNpc(sceneID string, msg *pb.SceneMsg) {
	scene := sl.sm.getScene(sceneID)
	if scene == nil {
		tilogs.L().Errorf("sceneLineManager activeNpc sl.sm.getScene: %v is nil", sceneID)
	}
	scene.in(msg)
}

func (sl *sceneLineManager) deleteNpc(sceneID string, msg *pb.SceneMsg) {
	scene := sl.sm.getScene(sceneID)
	if scene == nil {
		tilogs.L().Errorf("sceneLineManager deleteNpc sl.sm.getScene: %v is nil", sceneID)
	}
	scene.in(msg)
}

func (sl *sceneLineManager) inSyncReqScene(ctx context.Context, sceneID string, msg *pb.SceneMsg) *pb.SceneMsg {
	scene := sl.sm.getScene(sceneID)
	if scene == nil {
		tilogs.L().Warnf("sceneLineManager inSyncReqScene sl.sm.getScene: %s is nil, msgID %d, acids %s", sceneID, msg.MsgId, msg.Acids)
		msg.ErrDes = ErrSceneMapNotExists.Error()
		return msg
	}
	resp := scene.inSync(ctx, msg)
	sl.linePlayerNum[sceneID] = resp.curNumInfo[IdxLinePlayerNum] + resp.curNumInfo[IdxLinePlayerSeizeSeat]
	return resp.msg
}

// 获取场景的各个分线的人数信息（非实时）。
func (sl *sceneLineManager) getSceneLineNotRealTime() *pb.SceneMsg {
	ret := &pb.SceneMsg{}
	ret.SceLineRoleNum = make([]*pb.SceneLineRoleNum, 0, len(sl.lineUniq))
	for _, lineInfo := range sl.lineUniq {
		ret.SceLineRoleNum = append(ret.SceLineRoleNum, &pb.SceneLineRoleNum{
			SceLine:       uint32(lineInfo.lineNum),
			SceRoleNum:    uint32(sl.linePlayerNum[lineInfo.sceneLineUniq]),
			SceRoleMaxNum: uint32(getMaxPlayerScene(sl.mapID)),
		})
	}
	return ret
}

// 获取场景的各个分线的数量信息。
func (sl *sceneLineManager) getSceneLine() *pb.SceneMsg {
	ret := scene_pool.GetMsg()
	ret.SceLineRoleNum = make([]*pb.SceneLineRoleNum, 0, len(sl.lineUniq))
	for _, lineInfo := range sl.lineUniq {
		scene := sl.sm.getScene(lineInfo.sceneLineUniq)
		if scene == nil {
			tilogs.L().Errorf("sceneLineManager getSceneLine sceneID: %v is nil", lineInfo.sceneLineUniq)
			continue
		}

		nums := scene.getSceneNumInfoAndDeleteTrigger().NumInfo
		extra := make([]uint32, 0, len(nums)-1)
		for _, num := range nums[1:] {
			extra = append(extra, uint32(num))
		}

		ret.SceLineRoleNum = append(ret.SceLineRoleNum, &pb.SceneLineRoleNum{
			SceLine:       uint32(lineInfo.lineNum),
			SceRoleNum:    uint32(nums[IdxLinePlayerNum] + nums[IdxLinePlayerSeizeSeat]),
			SceRoleMaxNum: uint32(getMaxPlayerScene(sl.mapID)),
			Extra:         extra,
		})
	}
	tilogs.L().Debugf("sceneLineManager getSceneLine SceLineRoleNum: %v", ret.SceLineRoleNum)
	return ret
}

// 玩家切线前尝试预占位。
func (sl *sceneLineManager) changeSceneLine(sceneID string, acid string) *pb.SceneMsg {
	ret := &pb.SceneMsg{
		Code: pb.TrySceneSeizeSeat_Success,
	}

	// 获取目标场景线。
	scene := sl.sm.getScene(sceneID)
	if scene == nil {
		ret.Code = pb.TrySceneSeizeSeat_NotExit
		return ret
	}

	if sl.linePlayerNum[sceneID] >= getMaxPlayerScene(sl.mapID) {
		// 尝试为玩家预占位失败。
		ret.Code = pb.TrySceneSeizeSeat_Fail
		return ret
	}

	// 尝试玩家预占位。
	result := scene.trySeizeSeat(acid)
	sl.linePlayerNum[sceneID] = result.NumInfo[IdxLinePlayerNum] + result.NumInfo[IdxLinePlayerSeizeSeat]
	if !result.Success {
		// 尝试为玩家预占位失败。
		ret.Code = pb.TrySceneSeizeSeat_Fail
		return ret
	}

	return ret
}

// 将场景服的消息转发回gamex
// TODO 如果性能影响较大，后续需要实现新的Scene，支持多个sendChan
// func (sl *sceneLineManager) forward2shards() {
// 	for {
// 		select {
// 		case msg, ok := <-sl.outChan:
// 			if !ok {
// 				tilogs.L().Infof("sceneLineManager %d forward2shards quit", sl.uID)
// 				return
// 			}
//
// 			tilogs.L().Debugf("forward2shards receive msg to shard %d", msg.ShardId)
//
// 			v, ok := sl.outChans.Load(int(msg.ShardId))
// 			if ok {
// 				select {
// 				case v.(chan *pb.SceneMsg) <- msg:
// 				default:
// 					tilogs.L().Errorf("sceneLineManager %d forward2shards to shard %d failed, channel full", sl.uID, msg.ShardId)
// 				}
// 			}
// 		}
// 	}
// }
