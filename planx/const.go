package planx

import (
	"strings"
	"time"
)

// runmode
const (
	RunMode_Local_Test = "local_test" // 用于本地服，特点是一个gamex对应一个cross
	RunMode_Local      = "local"
	RunMode_Dev        = "dev"
	RunMode_PrefTest   = "pref_test" // 压测模式
	RunMode_Prod       = "prod"      // 只有正式线上服才会用，渠道测试服都不要用
)

func IsRunProd(runMode string) bool {
	return runMode == RunMode_Prod
}

func IsRunDev(runMode string) bool {
	return runMode == RunMode_Dev
}

// IsRunInLocal 是否是本地环境, 本地服包括Local和Local_Test
func IsRunInLocal(runMode string) bool {
	return IsRunLocalTest(runMode) || IsRunLocal(runMode)
}

func IsRunLocal(runMode string) bool {
	return runMode == RunMode_Local
}

func IsRunLocalTest(runMode string) bool {
	return runMode == RunMode_Local_Test
}

func IsRunPrefTest(runMode string) bool {
	return runMode == RunMode_PrefTest
}

func IsCheatEnable(cheatEnable string) bool {
	return strings.ToLower(cheatEnable) == "true"
}

// 是否镜像服
func IsMirrorEnable(mirrorEnable string) bool {
	return strings.ToLower(mirrorEnable) == "true"
}

const (
	CloudServerName_AWS     = "aws"
	CloudServerName_QCCloud = "qcloud"
	CloudServerName_Docker  = "docker"
	CloudServerName_Tencent = "tencent"
	CloudServerName_Aliyun  = "aliyun"
	CloudServerName_Azure   = "azure"
	CloudServerName_HuaWei  = "huawei"
	CloudServerName_Google  = "gcp"
	CloudServerName_Oracle  = "oracle"
)

// 这是kick协议的enumId，由于协议是逻辑代码，这里只能写死id
const PushCodeKick = 30001

const (
	ClientTimeOut        = 4500 * time.Millisecond // 客户端协议发起方的超时时间
	GateClientTimeOut    = 15 * time.Second        // 客户端gate超时
	UpTimeOut            = 600 * time.Millisecond  // 上行超时, 可能因为服务器内部处理任务较多，所以设置的超时时间较长
	DownTimeOut          = 200 * time.Millisecond  // 下行超时
	InnerTimeOut         = 400 * time.Millisecond  // 服务内部处理超时
	BattleCheckTimeOut   = 1200 * time.Millisecond // 服务内部处理超时
	BattleRoomTimeOut    = time.Second             // 战斗房间相关超时
	BossChallengeTimeOut = 3 * time.Second         // boss挑战战区化时可能需要在请求cross，时间设置长点
	StressTimeOut        = 3 * time.Second         // 压测相关超时，压测使用
	CachexTimeOut        = 4 * time.Second         // cachex处理批量请求的超时时间，比客户端超时减上行超时略短，允许菊花，不允许断线
	CacheXTimeOutCutDown = 2 * time.Second
	GMSTimeOut           = 1600 * time.Millisecond // 部分GMS请求需要gamex拉起profile应答，适当延长超时时间
	GamexBatchReqResp    = 3 * time.Second         // gamex批量请求的超时时间
)

// 存db间隔
const SaveDBInternal = time.Second * 30

const StatisticsOnlineInternal = 5 * time.Minute

const LogSendInterval = 1 * time.Second

const SecondsEveryDay = 3600

const SecondTicInterval = 1 * time.Second

const ShardCCUUpdateTime = 5 * time.Minute

const LoadServerGroup = time.Second * 30 // 加载战区时间间隔

const LoadVersionUpdate = time.Second * 30 // 加载gmServer 版本更新间隔

const LoadGMGlobalMail = time.Second * 30 // 加载gm区域邮件，时间间隔

const ReqCrossTriggerWarZone = time.Second * 5

const LoadGMSilenceSys = time.Second * 30 // 加载禁言信息，时间间隔

const OfflineDealCache = time.Millisecond * 300 // 处理离线模块缓存时间间隔
const OfflineDealMaxPerTime = 100

const TriggerWarZone2GameTime = time.Second * 5

const ClearBattleMemberTIme = time.Second * 30

const GVGPlayerFightingAndRankTime = time.Millisecond * 500
