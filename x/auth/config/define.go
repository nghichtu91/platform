package config

import (
	"encoding/json"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/limit"
)

type AppConfig struct {
	CommonCfg         CommonConfig         `toml:"CommonConfig"`
	LimitCfg          limit.LimitConfig    `toml:"LimitConfig"`
	HeroSdkAndroidCfg SdkHeroAndroidConfig `toml:"SdkHeroAndroidConfig"`
	HeroSdkIosCfg     SdkHeroIosConfig     `toml:"SdkHeroIosConfig"`
}

func (ac AppConfig) String() string {
	jsonBytes, err := json.Marshal(ac)
	if err != nil {
		tilogs.L().Errorf("json marshal error %v", err)
		return ""
	}
	return string(jsonBytes)
}

type CommonConfig struct {
	EtcdEndPoint []string `toml:"etcd_endpoint"`
	EtcdRoot     string   `toml:"etcd_root"`
	EtcdServer   string   `toml:"etcd_server"`

	Gid      uint   `toml:"gid"`
	ServerId string `toml:"serverid"`

	PublicIP      string `toml:"publicip"`
	EnableHttpTLS bool   `toml:"EnableHttpTLS"`
	HttpsPort     string `toml:"HttpsPort"`
	HttpsCertFile string `toml:"HttpsCertFile"`
	HttpsKeyFile  string `toml:"HttpsKeyFile"`
	Httpport      string `toml:"httpport"`

	InternalIp string `toml:"internalip"`

	ElbAuthAddr string `toml:"elb_auth_addr"` // auth配置的elb地址，如果存在，替换public ip

	// mongo
	MongoAuthDbName string `toml:"MongoAuthDbName"` // 目前只有mongo有db，redis和dynamo都没有db的概念

	// redis
	AuthRedisAddr  string `toml:"AuthRedisAddr"`
	AuthRedisDb    string `toml:"AuthRedisDb"`
	AuthRedisDbPwd string `toml:"AuthRedisDbPwd"`

	LoginRedisAddr  string `toml:"LoginRedisAddr"`
	LoginRedisDb    string `toml:"LoginRedisDb"`
	LoginRedisDbPwd string `toml:"LoginRedisDbPwd"`

	// table
	AuthDeviceTable        string `toml:"AuthDeviceTable"`
	AuthNameTable          string `toml:"AuthNameTable"`
	AuthUserInfoTable      string `toml:"AuthUserInfoTable"`
	AuthGmTable            string `toml:"AuthGmTable"`
	AuthUserShardInfoTable string `toml:"AuthUserShardInfoTable"`
}

type SdkHeroAndroidConfig struct {
	Url          string `toml:"url"`
	UrlLogin     string `toml:"url_login"`
	AppKey       string `toml:"appKey"`
	ProductId    string `toml:"productId"`
	ProjectId    string `toml:"projectId"`
	ClientSecret string `toml:"client_secret"`
	ServerId     string `toml:"serverId"`
}

type SdkHeroIosConfig struct {
	Url          string `toml:"url"`
	UrlLogin     string `toml:"url_login"`
	AppKey       string `toml:"appKey"`
	ProductId    string `toml:"productId"`
	ProjectId    string `toml:"projectId"`
	ClientSecret string `toml:"client_secret"`
	ServerId     string `toml:"serverId"`
}
