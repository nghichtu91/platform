package config

import (
	"encoding/json"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

//ChatConfig配置
type ChatConfig struct {
	Gid          uint     `toml:"gid"`
	ServerId     string   `toml:"serverid"`
	EtcdEndPoint []string `toml:"etcd_endpoint"`
	EtcdDevops   string   `toml:"etcd_devops"`
	EtcdServer   string   `toml:"etcd_server"`
	InternalIp   string   `toml:"internalip"`
	Listen       string   `toml:"listen"`
	NatsUrl      string   `toml:"NatsUrl"`

	Room struct {
		Batch  int `toml:"Batch"`
		Signal int `toml:"Signal"`
		Idle   int `toml:"Idle"`
	} `toml:"Room"`
	Comet struct {
		RoutineChan int `toml:"RoutineChan"`
		RoutineSize int `toml:"RoutineSize"`
	} `toml:"Comet"`
}

func (ac ChatConfig) String() string {
	jsonBytes, err := json.Marshal(ac)
	if err != nil {
		tilogs.L().Errorf("json marshal error %v", err)
		return ""
	}
	return string(jsonBytes)
}
