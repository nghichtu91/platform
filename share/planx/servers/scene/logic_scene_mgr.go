package scene

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	
	"github.com/nghichtu91/platform/share/planx/merge_util"
	
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
	
	"github.com/nghichtu91/platform/share/planx/tilogs"
	
	"github.com/nghichtu91/platform/share/x/common/msg/scene/pb"
)

const (
	AllowSelfDelete    = true  // 允许场景线执行自我删除。
	NotAllowSelfDelete = false // 不允许场景线执行自我删除。
)

// 用于nats主题注册
const (
	crossSceneQuizMapId = 602001 // 跨服 场景答题 地图id
	sceneQuizMapId      = 602011 // 单服 场景答题 地图id
)

type GenLogicScene func(mapId int32, sceneUniq, sceneType string,
	outChan chan *pb.SceneMsg, outChans *sync.Map, allLineNpc map[string]*pb.AllLineNpcInfo) LogicScene
type GetSceneType func(mapID int32) int32

type LogicScene interface {
	HandleSceneMsg(param *pb.SceneMsg) *pb.SceneMsg
	GetSceneNum() []int
	TrySeizeSeat(string) bool
	IsSeizeSeat(string) bool
	GetLastRoleExitSceneTimeStamp() int64
	CleanUpForciblyOccupySeat()
	CleanUpKeepAliveTimeout()
	CleanUpScreens()
	GetSceneType() int32
	OnCreate()
	Tick()
	GetMapId() int32
}

func createSceneMgr(gen GenLogicScene, getType GetSceneType, gid, shardId uint) *sceneMgr {
	return &sceneMgr{
		scenes:          make(map[string]*scene, DefaultSceneNum),
		getSceneType:    getType,
		genLogicScene:   gen,
		sceneLine:       make(map[int64]*sceneLineManager, DefaultSceneLingNum),
		dynamicOutChan:  make(map[int32]chan *pb.SceneMsg),
		dynamicOutChans: make(map[int64]*sync.Map),
		dynamicNats:     make(map[int64]*dynamicNatsMgr),
		gid:             gid,
		shardId:         shardId,
		metricsPrefix:   "requests." + strconv.Itoa(int(gid)) + "." + strconv.Itoa(int(shardId)) + ".scene", // requests.%d.%d.scene
	}
}

// sceneMgr 管理所有LogicScene
// 每个scene服务唯一单例
type sceneMgr struct {
	// 场景线 Map[SceneID]Scene
	// 对于动态场景，Key为 mapID:uniqueID
	// 对于静态场景，Key为 mapID:lineID
	// 对于跨服场景，Key为 mapID:lineID:zoneID
	scenes          map[string]*scene
	sceneLine       map[int64]*sceneLineManager // 场景线管理 Map[zoneID+mapID]sceneLineManager。
	dynamicOutChan  map[int32]chan *pb.SceneMsg // 动态场景OutChan Map[mapID]OutChan。
	dynamicOutChans map[int64]*sync.Map         // 动态跨服场景OutChan Map[zoneID+mapID]OutChans
	dynamicNats     map[int64]*dynamicNatsMgr
	
	getSceneType    GetSceneType
	genLogicScene   GenLogicScene
	lock            sync.RWMutex
	sceneLineLock   sync.RWMutex
	dynamicChanLock sync.RWMutex
	
	gid     uint
	shardId uint
	
	metricsPrefix string
}

func (sm *sceneMgr) startSceneLineMgr(mapId, warzoneId int32, sceneType string, sceneNum int) *sceneLineManager {
	// 这里需要尽早锁上，分组变化时可能有好几个gamex同时请求
	sm.sceneLineLock.Lock()
	uid := util.Int32Merge2Int64(warzoneId, mapId)
	
	// 再查一次是否存在
	if mgr, ok := sm.sceneLine[uid]; ok {
		sm.sceneLineLock.Unlock()
		return mgr
	}
	
	sceneLineMgr := &sceneLineManager{
		uID:           uid,
		mapID:         mapId,
		zoneID:        warzoneId,
		sceneType:     sceneType,
		quit:          make(chan bool, 1),
		linePlayerNum: make(map[string]int, 8),
		sm:            sm,
		outChan:       make(chan *pb.SceneMsg, MgrOutMsgChanSize),
		outChans:      new(sync.Map),
		lineChan:      make(chan *ParamSceneLine, MgrLineReqChanSize),
		AllLineNpc:    make([]*pb.AllLineNpcInfo, 0, AllNPCNum),
	}
	
	sm.sceneLine[uid] = sceneLineMgr
	sm.sceneLineLock.Unlock()
	
	now := time.Now().Unix()
	for i := StartSceneLine; i <= sceneNum; i++ {
		sceneLineUniq := getSceneUniq(mapId, i, warzoneId)
		if sm.addScene(mapId, warzoneId, sceneLineUniq, sceneType, sceneLineMgr.outChan, sceneLineMgr.outChans, NotAllowSelfDelete, nil) {
			sceneLineMgr.lineUniq = append(sceneLineMgr.lineUniq, sceneLineNum{
				sceneLineUniq: sceneLineUniq,
				lineNum:       i,
				startTime:     now,
			})
			sceneLineMgr.linePlayerNum[sceneLineUniq] = 0
		}
	}
	tilogs.L().Infof("sceneLineMgr startSceneLineMgr %p mapId %d, warzoneid %d, uid %d, lineUniq %+v, linePlayerNum %+v",
		sceneLineMgr, mapId, warzoneId, uid, sceneLineMgr.lineUniq, sceneLineMgr.linePlayerNum)
	sceneLineMgr.start()
	
	return sceneLineMgr
}

func (sm *sceneMgr) stopSceneLineMgr(uid int64) {
	sm.sceneLineLock.Lock()
	defer sm.sceneLineLock.Unlock()
	oldMgr, ok := sm.sceneLine[uid]
	if ok {
		oldMgr.stop()
		delete(sm.sceneLine, uid)
	}
}

func (sm *sceneMgr) addScene(mapId, zoneID int32, sceneLineUniq, sceneType string,
	outChan chan *pb.SceneMsg, outChans *sync.Map, canSelfDelete bool, allLineNpc map[string]*pb.AllLineNpcInfo) bool {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	_, ok := sm.scenes[sceneLineUniq]
	if ok {
		return true
	}
	
	switch sceneType {
	case StaticSceneType:
	case XSceneType:
	case XDynamicSceneType:
		outChans = sm.getDynamicOutChans(zoneID, sceneLineUniq)
	default:
		/*
			动态场景的outChan在创建scene的锁内再获取,因为极限情况下会出现创建的时候game-scene断掉了
			又瞬间建立连接,重新创建了新的outChan
			断掉会先清除所有的动态outChan,再删除所有的动态scene,保证删掉的scene携带的都是旧的outChan
			如果这里不重新获取,可能出现使用的是旧的outChan,创建新的scene
		*/
		outChan = sm.getDynamicOutChan(sceneLineUniq)
	}
	
	if outChan == nil && outChans == nil {
		retErr := fmt.Errorf("scene_server OnCommand s.sceMgr.getDynamicOutChan sceneID: %v is nil", sceneLineUniq)
		tilogs.L().Errorf(retErr.Error())
		return false
	}
	
	scene := &scene{
		sceneId:      sceneLineUniq,
		logicScene:   sm.genLogicScene(mapId, sceneLineUniq, sceneType, outChan, outChans, allLineNpc),
		inChan:       make(chan Req, InMsgChanSize),
		innerReqChan: make(chan InnerReq, InMsgChanSize),
		// quit:         make(chan struct{}, 1),
		// sceLineChan:     make(chan NumInfoAndDeleteTriggerReq, 32),
		// seizeSeatChan:   make(chan SeizeSeat, 32),
		// isSeizeSeatChan: make(chan IsSeizeSeat, 32),
		canSelfDelete: canSelfDelete,
		gid:           sm.gid,
		shardId:       sm.shardId,
		uID:           util.Int32Merge2Int64(zoneID, mapId),
		sm:            sm,
	}
	// logicScene.CopyNpc()
	sm.AddSceneToNatsMgr(mapId, zoneID, sm.gid, scene)
	scene.start()
	sm.scenes[sceneLineUniq] = scene
	// metrics统计
	allmetrics.AddSceneMapMetric(GetMapBySceneUniq(sceneLineUniq))
	
	// 静态地图场景线需要记录线内人数。上线后删除
	// if sceneType == StaticSceneType {
	//	allmetrics.AddSceneMetric(sceneLineUniq)
	// }
	tilogs.L().Infof("sceneMgr addScene，sceneID: %v", sceneLineUniq)
	return true
}

func (sm *sceneMgr) getScene(sceneUniq string) *scene {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	_, ok := sm.scenes[sceneUniq]
	if !ok {
		sli := make([]string, 0, len(sm.scenes))
		for k := range sm.scenes {
			sli = append(sli, k)
		}
		tilogs.L().Warnf("sceneMgr getScene not find the scene %s curr %v", sceneUniq, sli)
		return nil
	}
	return sm.scenes[sceneUniq]
}

func (sm *sceneMgr) getSceneLineMgr(uid int64) *sceneLineManager {
	sm.sceneLineLock.RLock()
	defer sm.sceneLineLock.RUnlock()
	_, ok := sm.sceneLine[uid]
	if !ok {
		tilogs.L().Warnf("sceneMgr getSceneLineMgr not find the scene uID %d", uid)
		return nil
	}
	return sm.sceneLine[uid]
}

func (sm *sceneMgr) getDynamicOutChan(sceneUniq string) chan *pb.SceneMsg {
	mapID := getMapBySceneUniq(sceneUniq)
	sm.dynamicChanLock.RLock()
	defer sm.dynamicChanLock.RUnlock()
	return sm.dynamicOutChan[mapID]
}

func (sm *sceneMgr) setDynamicOutChan(sceneUniq string, outChan chan *pb.SceneMsg) {
	mapID := getMapBySceneUniq(sceneUniq)
	sm.dynamicChanLock.Lock()
	defer sm.dynamicChanLock.Unlock()
	sm.dynamicOutChan[mapID] = outChan
}

func (sm *sceneMgr) deleteDynamicOutChan(sceneUniq string) {
	mapID := getMapBySceneUniq(sceneUniq)
	sm.dynamicChanLock.Lock()
	defer sm.dynamicChanLock.Unlock()
	delete(sm.dynamicOutChan, mapID)
}

// 注意有可能拿到nil
func (sm *sceneMgr) getDynamicOutChans(zoneID int32, sceneUniq string) *sync.Map {
	mapID := getMapBySceneUniq(sceneUniq)
	uid := util.Int32Merge2Int64(zoneID, mapID)
	sm.dynamicChanLock.RLock()
	defer sm.dynamicChanLock.RUnlock()
	return sm.dynamicOutChans[uid]
}

func (sm *sceneMgr) setDynamicOutChans(zoneID int32, sceneUniq string, shardID uint32, outChan chan *pb.SceneMsg) {
	mapID := getMapBySceneUniq(sceneUniq)
	uid := util.Int32Merge2Int64(zoneID, mapID)
	sm.dynamicChanLock.Lock()
	defer sm.dynamicChanLock.Unlock()
	if _, ok := sm.dynamicOutChans[uid]; !ok {
		sm.dynamicOutChans[uid] = new(sync.Map)
	}
	// 主服所有对应的合服ShardID，也一并放入outChans
	otherShards, err := merge_util.GetSIDsFromReal(int(shardID))
	if err != nil {
		tilogs.L().Errorf("scene_server OnStream get shard %d sub shards failed, err %s", shardID, err.Error())
	}
	for _, sid := range otherShards {
		sm.dynamicOutChans[uid].Store(uint32(sid), outChan)
	}
}

func (sm *sceneMgr) deleteDynamicOutChans(zoneID int32, sceneUniq string, shardID uint32, outChan chan *pb.SceneMsg) {
	mapID := getMapBySceneUniq(sceneUniq)
	uid := util.Int32Merge2Int64(zoneID, mapID)
	sm.dynamicChanLock.Lock()
	defer sm.dynamicChanLock.Unlock()
	
	outChs, ok := sm.dynamicOutChans[uid]
	if !ok || outChs == nil {
		return
	}
	
	v, ok := outChs.Load(shardID)
	if !ok {
		return
	}
	preOutChan, ok := v.(chan *pb.SceneMsg)
	if !ok {
		return
	}
	
	if preOutChan == outChan {
		otherShards, err := merge_util.GetSIDsFromReal(int(shardID))
		if err != nil {
			tilogs.L().Errorf("scene_server OnStream get shard %d sub shards failed, err %s", shardID, err.Error())
		}
		for _, sid := range otherShards {
			outChs.Delete(sid)
		}
	}
}

func (sm *sceneMgr) stopDynamicMapScene(mapID int32) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	for sceneID, scene := range sm.scenes {
		if getMapBySceneUniq(sceneID) == mapID {
			delete(sm.scenes, sceneID)
			sm.DelSceneFromNatsMgr(scene)
			scene.stop()
			allmetrics.ReduceSceneMapMetric(GetMapBySceneUniq(sceneID))
		}
	}
}

func (sm *sceneMgr) delScene(sceneId string) {
	sm.lock.Lock()
	s, ok := sm.scenes[sceneId]
	if !ok {
		sm.lock.Unlock()
		return
	}
	sm.DelSceneFromNatsMgr(s)
	delete(sm.scenes, sceneId)
	sm.lock.Unlock()
	allmetrics.ReduceSceneMapMetric(GetMapBySceneUniq(sceneId))
	s.stop()
}

func (sm *sceneMgr) sceneSelfDel(sceneID string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	
	s, ok := sm.scenes[sceneID]
	if !ok {
		return
	}
	delete(sm.scenes, sceneID)
	sm.DelSceneFromNatsMgr(s)
	allmetrics.ReduceSceneMapMetric(GetMapBySceneUniq(sceneID))
	s.selfStopWithoutWait()
}

func (sm *sceneMgr) stop() {
	sm.stopScene()
	sm.stopSceneLine()
}

func (sm *sceneMgr) stopScene() {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	for sID, s := range sm.scenes {
		s.stop()
		sm.DelSceneFromNatsMgr(s)
		allmetrics.ReduceSceneMapMetric(GetMapBySceneUniq(sID))
	}
}

func (sm *sceneMgr) stopSceneLine() {
	sm.sceneLineLock.Lock()
	defer sm.sceneLineLock.Unlock()
	for _, s := range sm.sceneLine {
		s.stop()
	}
}

func getSceneUniq(mapID int32, line int, zoneID int32) string {
	if zoneID == 0 {
		return strconv.Itoa(int(mapID)) + ":" + strconv.Itoa(line)
	}
	return strconv.Itoa(int(mapID)) + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(int(zoneID))
	// return fmt.Sprintf("%d:%d", uID, line)
}

func getDynamicSceneUniq(sceneId string, zoneID int32) string {
	if zoneID == 0 {
		return sceneId
	}
	strZone := strconv.Itoa(int(zoneID))
	ids := strings.Split(sceneId, ":")
	if len(ids) > 2 {
		return sceneId
	}
	return sceneId + ":" + strZone
}

func getMapBySceneUniq(sceneUniq string) int32 {
	// arr := strings.Split(sceneUniq, ":")
	// mapId, err := strconv.Atoi(arr[0])
	// if err != nil {
	// 	return 0
	// }
	return int32(GetMapIdBySceneString(sceneUniq))
}

func GetMapBySceneUniq(sceneUniq string) string {
	if strings.Contains(sceneUniq, ":") {
		sceneUniq = sceneUniq[:strings.Index(sceneUniq, ":")]
	}
	return sceneUniq
}

// GetSceneLineBySceneUniq 获取场景线
// 这里需要考虑scene id有两部分或者3部分的情况
func GetSceneLineBySceneUniq(sceneUniq string) string {
	// 去掉mapID
	if strings.Contains(sceneUniq, ":") {
		sceneUniq = sceneUniq[strings.Index(sceneUniq, ":")+1:]
	}
	// 如果后面还有冒号，去掉后面的部分
	if strings.Contains(sceneUniq, ":") {
		sceneUniq = sceneUniq[:strings.Index(sceneUniq, ":")]
	}
	return sceneUniq
}

func GetMapIdBySceneString(sceneId string) int {
	mapId, _ := strconv.Atoi(GetMapBySceneUniq(sceneId))
	return mapId
}

func GetLineIdBySceneString(sceneId string) int {
	lineId, _ := strconv.Atoi(GetSceneLineBySceneUniq(sceneId))
	return lineId
}

func (sm *sceneMgr) IsNeedNats(mapId int32) bool {
	for _, v := range mapIdNeedStartNats {
		if v == mapId {
			return true
		}
	}
	return false
}

func (sm *sceneMgr) AddSceneToNatsMgr(mapId int32, zoneId int32, gid uint, sc *scene) {
	// 因为这个增加删除都是跟着Scene的map的创建删除一起的在用的是外围的scene
	
	if !sm.IsNeedNats(mapId) {
		return
	}
	tilogs.L().Debugf("[sceneMgr] AddSceneToNatsMgr now add NatsService for mapId: %v, zoneId: %v", mapId, zoneId)
	mgr, ok := sm.dynamicNats[sc.uID]
	if !ok || mgr == nil {
		// sc.uID是由 zoneId 和 mapId 拼成的int64，非跨服场景因为 zoneId 为0，此时需要赋值sid
		sceneUid := sc.uID
		if zoneId <= 0 {
			mapIdUid := mapId
			// 为了方便nats通信，场景答题 单服地图id 转为 跨服地图id
			if mapIdUid == sceneQuizMapId {
				mapIdUid = crossSceneQuizMapId
			}
			
			sceneUid = util.Int32Merge2Int64(int32(sm.shardId), mapIdUid)
		}
		
		mgr = &dynamicNatsMgr{
			sceneMgr: sm,
			gid:      gid,
			uID:      sceneUid,
			sUC:      make([]*scene, 0),
			quit:     make(chan bool, 1),
			cmdChan:  make(chan iCmd, 2048),
		}
		mgr.start()
		sm.dynamicNats[sc.uID] = mgr
	}
	for _, v := range mgr.sUC {
		if v.sceneId == sc.sceneId {
			return
		}
	}
	mgr.sUC = append(mgr.sUC, sc)
	tilogs.L().Debugf("[%v]AddSceneToNatsMgr add new scene to suc %v now len %v", sc.uID, sc, len(mgr.sUC))
}

func (sm *sceneMgr) DelSceneFromNatsMgr(sc *scene) {
	dn := sm.dynamicNats[sc.uID]
	if dn == nil {
		return
	}
	var index int
	for index = range dn.sUC {
		if dn.sUC[index].sceneId == sc.sceneId {
			break
		}
	}
	dn.sUC = append(dn.sUC[:index], dn.sUC[index+1:]...)
	tilogs.L().Debugf("dynamicNats closeOneScene, now %v", len(dn.sUC))
	if len(dn.sUC) == 0 {
		delete(sm.dynamicNats, sc.uID)
		dn.stop()
	}
}
