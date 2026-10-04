package scene

import (
	"time"
)

const (
	MaxPlayerScene        = 64       // 单个场景线人数上限，由于队长将队员拉到场景线不受此限制，理论上单条场景线实际人数应该为这个数字乘3
	NeedNewScene          = 48       // 场景线多少人时开下一个场景线
	StartSceneLine        = 1        // 场景线起始数字
	NewSceneMaxExist      = 60       // 场景线删除前保留时间
	LastRoleLeave         = 5*60 + 5 // 扩充到5分钟
	DefaultNPCNum         = 16       // 预计的默认NPC数量
	DefaultXSceneShardNum = 16       // 预计一个跨服场景会连接几个shard
	DefaultSceneNum       = 64       // 一个manager预计有几个scene
	DefaultSceneLingNum   = 16       // 预计有几个带场景线的地图
	DefaultScreenNum      = 2048     // 预计屏幕数量
	InMsgChanSize         = 256      // 每个场景接收消息的队列长度
	MgrLineReqChanSize    = 2048     // 场景管理同步请求队列长度
	MgrOutMsgChanSize     = 32768    // 场景管理对外队列长度
	AllNPCNum             = 128      // 所有地图NPC唯一数量预计
	MaxObjectsInView      = 64       // 玩家视野内最大对象数量，一般建议和 MaxPlayerScene 一致
)

const (
	FrameMsgSend                         = 50 * time.Millisecond
	TickInterval                         = 1 * time.Second // 场景服执行timeutil.Now间隔时间
	SelfCheckRecycleIntervalTickInterval = 23              // 场景线自我删除检查Tick间隔。使用质数避免和其他Tick有更小公倍数
	SeizeSeatCancelCheckTickInterval     = 1201            // 场景线预占位取消检查Tick间隔。使用质数避免和其他Tick有更小公倍数
)

// 场景线返回结果数组相关
const (
	IdxLinePlayerNum       = iota // 玩家数量下标
	IdxLineMobElite               // 精英怪数量下标
	IdxLineMobNormal              // 普通怪数量下标
	IdxLinePlayerSeizeSeat        // 占位玩家数量下标
	LineSliceLen                  // slice 长度
)

const (
	GetMaxPlayerScene = 1
	GetNeedNewScene   = 2
)

const (
	DynamicSceneModule = "dynamic_scene_mgr"
)
