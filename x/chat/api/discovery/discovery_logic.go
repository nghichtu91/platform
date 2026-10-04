package discovery

type LogicServerExtraInfo struct {
	ServerStartTime int64  `json:"ServerStartTime"`
	HttpServerIp    string `json:"HttpServerIp"`
}

type LogicServerExtraTickInfo struct {
	ConnNum int32 `json:"ConnNum"`
	IsOpen  bool  `json:"IsOpen"`
}

type LogicServerInfo struct {
	PrivateAddr string // grpc ip port
	HttpAddr    string // http ip port
	StartTime   int64  // 聊天服的启动时间
	IsOpen      bool   // 是否对外服务
}

// 发现logic 回调
type HandleLogicDiscovery interface {
	OnServerAdd(string, *LogicServerInfo)
	OnServerDel(string)
}
