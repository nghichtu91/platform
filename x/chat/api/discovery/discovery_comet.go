package discovery

type CometServerExtraInfo struct {
	ServerStartTime int64  `json:"ServerStartTime"`
	GrpcServerIp    string `json:"GrpcServerIp"`
}

type CometServerExtraTickInfo struct {
	ConnNum int32 `json:"ConnNum"`
	IsOpen  bool  `json:"IsOpen"`
}

type CometServerInfo struct {
	PrivateAddr string // 内网的ip和端口
	StartTime   int64  //聊天服的启动时间
	IsOpen      bool   //是否对外服务
}

// 发现comet 回调通知job
type HandleCometDiscovery interface {
	OnServerAdd(string, *CometServerInfo)
	OnServerDel(string)
}
