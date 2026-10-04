package allmetrics

/*
	各个服务的metrics的前缀都定义在这里
	在Grafana中，就用此前缀和此包中其他文件中定义的具体名称进行拼接
*/
const (
	NoticePrefix        = "notice"
	AuthPrefix          = "auth"
	GatePrefix          = "gatex"
	GamePrefix          = "gamex"
	ScenePrefix         = "scene"
	BattlePrefix        = "battle"
	ModulexPrefix       = "modulex"
	CrossxPrefix        = "crossx"
	ClientBIPrefix      = "clientbilog"
	StressxPrefix       = "stressx"
	CometPrefix         = "cometx"
	LogicPrefix         = "logicx"
	JobPrefix           = "jobx"
	FriendXPrefix       = "friendx"
	PayxPrefix          = "payx"
	GMServerPrefix      = "gmserver"
	GMChatPrefix        = "gmchat"
	MergerPrefix        = "merger"
	ServerCheckPrefix   = "servercheck"
	CachexPrefix        = "cachex"
	BattlexAgentPrefix  = "battlexAgent"
	SensitiveWordPrefix = "sensitiveword"
	PingServerPrefix    = "pingserver"
)
