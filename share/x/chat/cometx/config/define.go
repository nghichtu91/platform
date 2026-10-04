package config

import (
	"encoding/json"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

// ChatConfig配置
type ChatConfig struct {
	Gid             uint     `toml:"gid"`
	ServerId        string   `toml:"serverid"`
	ChatxRedisAddr  string   `toml:"ChatxRedisAddr"`
	ChatxRedisDb    string   `toml:"ChatxRedisDb"`
	ChatxRedisDbPwd string   `toml:"ChatxRedisDbPwd"`
	EtcdEndPoint    []string `toml:"etcd_endpoint"`
	EtcdDevops      string   `toml:"etcd_devops"`
	EtcdServer      string   `toml:"etcd_server"`
	PublicIP        string   `toml:"publicip"`
	ElbCometxAddr   string   `toml:"elb_cometx_addr"`
	InternalIp      string   `toml:"internalip"`
	Listen          string   `toml:"listen"`
	LogicId         string   `toml:"logicid"`
	ElbAddr         []string `toml:"elbaddr"`

	RPCServer struct {
		Network           string `toml:"Network"`
		Addr              string `toml:"Addr"`
		Timeout           int    `toml:"Timeout"`
		IdleTimeout       int    `toml:"IdleTimeout"`
		MaxLifeTime       int    `toml:"MaxLifeTime"`
		ForceCloseWait    int    `toml:"ForceCloseWait"`
		KeepAliveInterval int    `toml:"KeepAliveInterval"`
		KeepAliveTimeout  int    `toml:"KeepAliveTimeout"`
	} `toml:"RPCServer"`
	RPCClient struct {
		Addr string `toml:"Addr"`
	} `toml:"RPCClient"`
	Bucket struct {
		Size          int `toml:"Size"`
		Channel       int `toml:"Channel"`
		Room          int `toml:"Room"`
		RoutineAmount int `toml:"RoutineAmount"`
		RoutineSize   int `toml:"RoutineSize"`
	} `toml:"Bucket"`
	TCP struct {
		Bind         []string `toml:"Bind"`
		Sndbuf       int      `toml:"Sndbuf"`
		Rcvbuf       int      `toml:"Rcvbuf"`
		KeepAlive    bool     `toml:"KeepAlive"`
		Reader       int      `toml:"Reader"`
		ReadBuf      int      `toml:"ReadBuf"`
		ReadBufSize  int      `toml:"ReadBufSize"`
		Writer       int      `toml:"Writer"`
		WriteBuf     int      `toml:"WriteBuf"`
		WriteBufSize int      `toml:"WriteBufSize"`
		DeadLine     int      `toml:"DeadLine"`
		LSBDeadLine  int      `toml:"LSBDeadLine"`
	} `toml:"TCP"`
	Protocol struct {
		Timer            int `toml:"Timer"`
		TimerSize        int `toml:"TimerSize"`
		SvrProto         int `toml:"SvrProto"`
		CliProto         int `toml:"CliProto"`
		HandshakeTimeout int `toml:"HandshakeTimeout"`
	} `toml:"Protocol"`
	Room struct {
		Cd int `toml:"Cd"`
	} `toml:"Room"`
}

func (ac ChatConfig) String() string {
	jsonBytes, err := json.Marshal(ac)
	if err != nil {
		tilogs.L().Errorf("json marshal error %v", err)
		return ""
	}
	return string(jsonBytes)
}
