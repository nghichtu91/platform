package promemetrics

/*
	各个服务的metrics的前缀都定义在这里
	最终展示的标识为：项目名+服务器类型前缀+大区号+区服号
*/
const (
	AuthPrefix     = "auth"
	BattlePrefix   = "battle"
	ClientBIPrefix = "clientbilog"
	CrossxPrefix   = "crossx"
	GamePrefix     = "gamex"
	GatePrefix     = "gatex"
	LoginPrefix    = "login"
	ModulexPrefix  = "modulex"
	NoticePrefix   = "notice"
	RedisPrefix    = "redis"
	ScenePrefix    = "scene"
	CometPrefix    = "cometx"
	LogicPrefix    = "logicx"
	JobPrefix      = "jobx"
	PayxPrefix     = "payx"
	GiftPrefix     = "gift"
	StressxPrefix  = "stressx"
)
