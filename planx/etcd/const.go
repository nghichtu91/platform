package etcd

import (
	"fmt"

	"github.com/nghichtu91/platform/share/x/common/consts"
)

type ServiceTypeId uint

const (
	_ ServiceTypeId = iota
	Ser_Auth
	Ser_Login
	Ser_Gate
	Ser_Gamex
	Ser_Crossx
	Ser_Match
	Ser_Multiplay
	Ser_Pay
	Ser_ChatLogic
	Ser_ChatComet
	Ser_ChatJob
	Ser_Notice
	Ser_Modulex
	Ser_Scene
	SerXScene // 跨服场景服
	Ser_Battle
	Ser_BattleCheck
	Ser_ClientBiLog
	Ser_Gmtools
	Ser_GmChat
	Ser_GMServer
	Ser_Stressx
	Ser_Friendx
	Ser_KcpRobot
	Ser_Merger // 合服工具
	Ser_ServerCheck
	Ser_Cachex      // 跨服镜像
	Ser_BattleAgent // 战斗代理
	Ser_PingServer  // kcpPing服务

	Ser_SensitiveWord
	// ...
	UnValid
)

const (
	Server_Auth          = "auth"
	Server_Login         = "login"
	Server_Gate          = "gate"
	Server_Gamex         = "gamex"
	Server_Crossx        = "crossx"
	Server_Match         = "match"
	Server_Multiplay     = "multiplay"
	Server_Pay           = "pay"
	Server_ChatLogic     = "logic"
	Server_ChatComet     = "comet"
	Server_ChatJob       = "job"
	Server_Notice        = "notice"
	Server_Modulex       = "modulex"
	Server_Scene         = "scene"
	ServerXScene         = "XScene"
	Server_Battle        = "battle"
	Server_BattleCheck   = "BattleCheck"
	Server_Gmtools       = "Gmtools"
	Server_GmChat        = "GmChat"
	Server_ClientBiLog   = "ClientBiLog"
	ServerSer_GMServer   = "GMServer"
	Server_Stressx       = "stressx"
	Server_Friendx       = "friendx"
	Server_KcpRobot      = "kcprobot"
	Server_Merger        = "merger"
	Server_ServerCheck   = "servercheck"
	Server_Cachex        = "cachex"
	Server_BattleAgent   = "battleagent" // 战斗代理
	Server_SensitiveWord = "sensitiveword"
	Server_PingServer    = "pingServer"
)

var (
	// 添加新的ServiceTypeId，这里一定要记得加！不然启不来
	SerTyp2Name = map[ServiceTypeId]string{
		Ser_Auth:          Server_Auth,
		Ser_Login:         Server_Login,
		Ser_Gate:          Server_Gate,
		Ser_Gamex:         Server_Gamex,
		Ser_Crossx:        Server_Crossx,
		Ser_Match:         Server_Match,
		Ser_Multiplay:     Server_Multiplay,
		Ser_Pay:           Server_Pay,
		Ser_ChatLogic:     Server_ChatLogic,
		Ser_ChatComet:     Server_ChatComet,
		Ser_ChatJob:       Server_ChatJob,
		Ser_Notice:        Server_Notice,
		Ser_Modulex:       Server_Modulex,
		Ser_Scene:         Server_Scene,
		SerXScene:         ServerXScene,
		Ser_Battle:        Server_Battle,
		Ser_BattleCheck:   Server_BattleCheck,
		Ser_ClientBiLog:   Server_ClientBiLog,
		Ser_Gmtools:       Server_Gmtools,
		Ser_GmChat:        Server_GmChat,
		Ser_GMServer:      ServerSer_GMServer,
		Ser_Stressx:       Server_Stressx,
		Ser_Friendx:       Server_Friendx,
		Ser_KcpRobot:      Server_KcpRobot,
		Ser_Merger:        Server_Merger,
		Ser_ServerCheck:   Server_ServerCheck,
		Ser_Cachex:        Server_Cachex,
		Ser_BattleAgent:   Server_BattleAgent,
		Ser_SensitiveWord: Server_SensitiveWord,
		Ser_PingServer:    Server_PingServer,
	}
	SerName2Typ map[string]ServiceTypeId
)

func init() {
	SerName2Typ = make(map[string]ServiceTypeId, len(SerTyp2Name))
	for id, name := range SerTyp2Name {
		SerName2Typ[name] = id
	}
}

const (
	DirSwitch               = "switch"
	KeyGameFunction         = "gamefunction" // 游戏功能开关
	KeyDevDebug             = "devdebug"     // debug开关
	KeyCross                = "cross"        // 跨服玩法开关
	KeyGidSwitch            = "gidswitch"    // 大区开关
	KeySeverForGMAddr       = "forgmaddr"
	DirHotData              = "hotdata"
	DirPatch                = "hotpatch"
	DirPatchWatch           = "hotpatchwatch"
	DirHotWatch             = "hotwatch"
	KeyDebugTimePath        = "debugtime"
	GameVersion             = "version"
	KeyDebugAddr            = "cheataddr"
	GamexDirs               = "shards"
	CrossxDir               = "crossx"
	KeyShardState           = "showState"
	KeyShardOpenTime        = "openServerTime"             // 服务器开服时间，逻辑计算使用
	KeyShardPublicOpenTime  = "publicServerTime"           // 对外开服时间
	KeyShardWarzoneOpenTime = "warZoneOpenTime"            // 战区开服时间，战区相关奖励逻辑使用
	KeyShardHisWZOpenTime   = "hisWZOpenTime"              // 战区历史开服时间，战区调整后计算战区开服时间
	KeyShardMaxActivePower  = "maxActivePower"             // 最高活跃战力
	KeyShardRunStatue       = "runState"                   // 运行状态
	GID2ShardID             = "gid2shardid"                // 大区号到服务器号映射表路径
	MergedInfo              = "mergedinfo"                 // gamex合服信息，同步gamex状态用
	Merges                  = "merges"                     // 配置所在位置
	KeyMergedShards         = "merged_shards"              // 合服信息，gamex启动时读取，gmserver合服写入
	KeyMergedUID            = "merged_uid"                 // 使用此Key的Version作为UID
	DirServerGroups         = "server_groups"              // 分组相关信息
	DirGamexOpenTime        = "gamex_open_time"            // gamex开服时间汇总
	KeyGroup2Member         = "group2member"               // crossx分组ID与member对应关系
	KeySrvNameWatch         = "server_name_watch"          // 服务器名称监控键值
	DirXScenes              = "xscenes"                    // 跨服场景服分配
	WarzoneDir              = "war_zone"                   // 战区
	KeyLargeWarzoneMirror   = "large_warzone_mirror"       // 大战区镜像
	DirCountry              = "country"                    // 国战分组
	LargeWarzoneDir         = "large_war_zone"             // 大战区
	KeyConflict             = "conflict"                   // 玩法互斥状态
	KeyCrossCheatWZGroups   = "cross_cheat_warzone_groups" // 跨服分组cheat状态
	WarzoneRevertKey        = "war_zone_reverted"          // 标记战区状态回退
	VirtualGID              = "virtual_gid"                // 虚拟大区
	Defaults                = "defaults"
	BeMerged                = "be_merged"    // 被合服sid，在合服时以及主服启动时更新
	BattleInfo              = "battle_info"  // 玩法帧数据上传概率
	DirStressRobot          = "stress_robot" // 压测机器人

)

const (
	GameVersion_Bundle = "/bundle"
)

func GetVersionWatchUpdateKey(root string, gid string) string {
	return fmt.Sprintf("%s/%s/%s", root, gid, GameVersion)
}

func GetVersionWLPwdWatchUpdateKey(root string) string {
	return fmt.Sprintf("%s/%s/", root, consts.KeyEndPoint_Whitelistpwd)
}

func GetPprofAddrKey(root, gid, serverType, serverId string) string {
	if serverId != "" {
		return fmt.Sprintf("%s/%s/pprof/%s_%s", root, gid, serverType, serverId)
	} else {
		return fmt.Sprintf("%s/%s/pprof/%s", root, gid, serverType)
	}

}

func GetRunStateKey(root, gid, serverID string) string {
	return fmt.Sprintf("%s/%s/shards/%s/runState", root, gid, serverID)
}

// GetEtcdPrefix 按类型生成对应etcd开关路径
func GetEtcdPrefix(devopsRoot, gid, switchType string) string {
	return fmt.Sprintf("%s/%s/defaults/%s", devopsRoot, gid, switchType)
}

func GetMaxActivePowerKey(root string, gid, serverId uint) string {
	return fmt.Sprintf("%s/%d/shards/%d/maxActivePower", root, gid, serverId)
}

func GetGidSwitchKey(etcdRoot string, gid uint) string {
	return fmt.Sprintf("%s/%d/defaults", etcdRoot, gid)
}

func GetBattleCheckDisable(etcdRoot string, gid uint) string {
	return fmt.Sprintf("%s/%d/defaults/BattleCheckDisable", etcdRoot, gid)
}

func GetBattleCheckVersionKey(etcdRoot string, gid uint) string {
	return fmt.Sprintf("%s/%d/%s/BattleCheck/version", etcdRoot, gid, KeyPathVersion)
}

func GetMarketSwitchPreKey(etcdRoot string, gid uint) string {
	return fmt.Sprintf("%s/%d/%s/%s/", etcdRoot, gid, DirSwitch, KeyDevDebug)
}

func GetBattleInfoSwitchKeyDir(etcdRoot string, gid uint) string {
	return fmt.Sprintf("%s/%d/%s", etcdRoot, gid, BattleInfo)
}

func GetBattleInfoSwitchKey(etcdRoot string, gid uint, switchType string) string {
	return fmt.Sprintf("%s/%d/%s/%s", etcdRoot, gid, BattleInfo, switchType)
}

func GetStressRobotKey(etcdRoot string, serverId string, gid uint) string {
	return fmt.Sprintf("%s/%d/%s/%s", etcdRoot, gid, serverId, DirStressRobot)
}

func GetCheckShardKey(etcdRoot string, gid uint) string {
	return fmt.Sprintf("%s/%d/checkshards", etcdRoot, gid)
}
func GetRobotAccKey(etcdRoot string, gid uint) string {
	return fmt.Sprintf("%s/%d/robotaccount", etcdRoot, gid)
}
