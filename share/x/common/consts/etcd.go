package consts

import (
	"fmt"
)

const (
	DirHotData                     = "hotdata"
	KeyIp                          = "ip"
	KeyInternalIp                  = "internalip"
	PathGame4Auth                  = "game4auth"
	KeySName                       = "sn"
	KeyDName                       = "dn"
	KeyState                       = "state"
	KeyShowState                   = "ss"
	KeyServerStartTime             = "serstarttime"     // gm配置的开服时间
	KeyServerUsedStartTime         = "serusedstarttime" // 服务器用的开服时间
	KeyServerLaunchTime            = "serlaunchtime"    // gamex启动时间
	KeyEndPoint                    = "gonggao_for_auth"
	KeyEndPoint_Whitelistpwd       = "whitelistpwd"
	KeyEndPoint_maintain_starttime = "maintain_starttime"
	KeyEndPoint_maintain_endtime   = "maintain_endtime"
	KeyMergedShard                 = "mergedshard"
	KeyShardVersion                = "version" // 服务器版本号
	KeyTeamAB                      = "teamAB"
	StateOnline                    = "online"
	MultiLang                      = "multilang"
	Language                       = "language"
	KeyPayInnerPort                = "pay_inner_http_port"
	ServerListVersion              = "serverlistversion"
	DirMarketActivity              = "marketactivity"
	DirAnticheat                   = "anticheat"
	KeySceneCount                  = "scene_count"
	KeyChatIllegalWordsHashPath    = "chat_illegal_words_hash_path"
	KeyChatConfigPath              = "chat_config_path"
	KeyChatStoreTimePath           = "chat_store_time"
	KeyShardAutoStartRoleCount     = "shardautostartrolecount"
	KeyShardAutoStartServer        = "shardautostartserver"
	AUTH_URL_SUFFIX                = "auth/publicIp"
	KeyShardCCU                    = "shard_ccu"
	KeyCometElbAddr                = "cometx_elb_address"
	KeyServerPlayerCount           = "server_player_count" // 服务器注册人数
)

const (
	ShowStateNotStart       = iota // 0 半启动
	ShowStateNewSer                // 1 新服
	ShowStateHot                   // 2 火爆(锁服)
	ShowStateMaintenanceNew        // 3 新服维护，玩家看得到，但进不去
	ShowStateMaintenanceHot        // 4 锁服维护，玩家看得到，但进不去
	ShowStateHide                  // 5 隐藏
	ShowStateLockedAdd      = 10   // 本服锁服增加值
	ShowStateMaintenanceAdd = 2    // 维护状态增加值
	// 暂时不需要

	ShowStateCrowd      = iota // 8 拥挤
	ShowStateExperience        // 9 体验
	ShowStateSuper             // 10 玩家看不到
	ShowStateCount
)

const (
	Driver_Mongo  = "MongoDB"
	Driver_Dynamo = "DynamoDB"
)

type ShardInfo struct {
	Machine           string
	Gid               uint
	Sid               uint
	Ip                string
	SName             string
	DName             string
	State             string
	ShowState         string
	StartTime         int64 // 对外开服时间
	LaunchTime        string
	RealStartTime     string
	Version           string
	MultiLang         string
	Language          string
	Recommend         int
	RecommendLanguage string
	GroupId           int32
	NameExtra         string
}

type GamexEtcdInternalInfo struct {
	Gid           uint
	Sid           uint
	Addr          string
	IsMaintaining bool // 单服维护状态
	PhysicsSid    uint // 物理服Sid
}

func DefaultGamexEtcdInternalInfo(gid, sid uint) GamexEtcdInternalInfo {
	return GamexEtcdInternalInfo{
		Gid:        gid,
		Sid:        sid,
		PhysicsSid: sid,
	}
}

type ShardInfoInDevops struct {
}

type ShardInfoInServer struct {
	ShowState         int    `etcd3:"showState"`
	OpenServerTime    int64  `etcd3:"openServerTime"`
	DisplayName       string `etcd3:"displayName"`
	SerLaunchTime     int64  `etcd3:"serLaunchTime"`
	PublicServerTime  int64  `etcd3:"publicServerTime"`
	SName             string `etcd3:"sn"`
	Ip                string `etcd3:"ip"`
	State             string `etcd3:"state"`
	RunState          string `etcd3:"runState"`
	RecommendValue    int    `etcd3:"recommendValue"`
	RecommendLanguage string `etcd3:"recommendLanguage"`
	OldShowState      int    `etcd3:"oldShowState"`
	CheatAddr         string `etcd3:"cheataddr"`
	MaxActivePower    int64  `etcd3:"maxActivePower"`
	GroupId           int32  `etcd3:"serverGroup"`
	NameExtra         string `etcd3:"nameExtra"`
}

type ShardInfoAll struct {
	ShardInfoInDevops
	ShardInfoInServer
}

func GetNoticeWatchUpdateKey(etcdServer string, gid uint) string {
	return fmt.Sprintf("%s/%d/notice/update", etcdServer, gid)
}

func GetSceneCountKey(etcdServer string, gid uint) string {
	return fmt.Sprintf("%s/%d/%s", etcdServer, gid, KeySceneCount)
}

// Deprecated: JWS2-42263 gate检查gamex状态改为使用service_be_discovery_alive
// func GetGameInternalIp4Auth(etcdServer string, gid, sid uint) string {
// 	return fmt.Sprintf("%s/%d/%s/%d/%s",
// 		etcdServer, gid, PathGame4Auth, sid, KeyInternalIp)
// }

type CrossInfoInDevops struct {
	ServerId uint `etcd3:"serverid"`
}
