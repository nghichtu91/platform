package config

import (
	"encoding/json"

	"github.com/nghichtu91/platform/share/x/common/config"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type Config struct {
	Gid      uint   `toml:"gid"`
	ServerId string `toml:"serverid"`

	// etcd
	EtcdEndPoint []string `toml:"etcd_endpoint"`
	EtcdDevOps   string   `toml:"etcd_devops"`
	EtcdServer   string   `toml:"etcd_server"`

	// https
	EnableHttpTLS bool   `toml:"EnableHttpTLS"`
	HttpsAddress  string `toml:"HttpsAddress"`
	HttpsCertFile string `toml:"HttpsCertFile"`
	HttpsKeyFile  string `toml:"HttpsKeyFile"`
	// http
	HttpAddress string `toml:"httpAddress"`
	InternalIp  string `toml:"internalip"`
}

func (ac Config) String() string {
	jsonBytes, err := json.Marshal(ac)
	if err != nil {
		tilogs.L().Errorf("json marshal error %v", err)
		return ""
	}
	return string(jsonBytes)
}

var (
	Cfg     Config
	GidCfg  *config.GidConfig
	RunMode string
	Proj    string
)
