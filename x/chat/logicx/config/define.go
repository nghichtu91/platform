package config

import (
	"encoding/json"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

// ChatConfig 配置
type ChatConfig struct {
	Gid          uint     `toml:"gid"`
	ServerId     string   `toml:"serverid"`
	EtcdEndPoint []string `toml:"etcd_endpoint"`
	EtcdDevops   string   `toml:"etcd_devops"`
	EtcdServer   string   `toml:"etcd_server"`
	PublicIP     string   `toml:"pulbicip"`
	InternalIp   string   `toml:"internalip"`
	Listen       string   `toml:"listen"`
	NatsUrl      string   `toml:"NatsUrl"`
	GmServerUrl  string   `toml:"gmserver"`

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
	Personal struct {
		MaxPersonSaveCount   int `toml:"MaxPersonSaveCount"`
		MaxPersonKeySaveTime int `toml:"MaxPersonKeySaveTime"`
	} `toml:"Personal"`
	Room struct {
		MaxRoomSaveCount   int `toml:"MaxRoomSaveCount"`
		MaxRoomKeySaveTime int `toml:"MaxRoomKeySaveTime"`
	} `toml:"Room"`
	Auth struct {
		MaxAuthTimeOut int `toml:"MaxAuthTimeOut"`
	} `toml:"Auth"`
	GmServer struct {
		ConnectTimeout   int `toml:"ConnectTimeout"`
		ReadWriteTimeout int `toml:"ReadWriteTimeout"`
	} `toml:"GmServer"`
	YiDun struct {
		Enable     bool    `toml:"Enable"`
		SecretId   string  `toml:"SecretId"`
		SecretKey  string  `toml:"SecretKey"`
		BusinessId string  `toml:"BusinessId"`
		Version    string  `toml:"Version"`
		Url        string  `toml:"Url"`
		Labels     []lable `toml:"Labels"`
		Score      uint64  `toml:"score"`
	} `toml:"YiDun"`
	FeiShuRobot struct {
		FeiShuEnable    bool   `toml:"FeiShuEnable"`
		FeiShuUrl       string `toml:"FeiShuUrl"`
		FeiShuSercetKey string `toml:"FeiShuSercetKey"`
		FeiShuCtxType   string `toml:"FeiShuCtxType"`
		FeiShuMsgType   string `toml:"FeiShuMsgType"`
		FeiShuCnUrl     string `toml:"FeiShuCnUrl"`
		FeiShuCn100Url  string `toml:"FeiShuCn100Url"`
		FeiShuCn200Url  string `toml:"FeiShuCn200Url"`
		FeiShuCn500Url  string `toml:"FeiShuCn500Url"`
	} `toml:"FeiShuRobot"`
	Translate struct {
		TranslateRedisAddr     []string `toml:"TranslateRedisAddr"`
		TranslateRedisLanguage []string `toml:"TranslateRedisLanguage"`
		TranslateRedisDb       []string `toml:"TranslateRedisDb"`
		TranslateRedisPwd      []string `toml:"TranslateRedisPwd"`
		LRUMaxMemory           []string `toml:"LRUMaxMemory"`
	} `toml:"Translate"`
}

type lable struct {
	Code int64  `toml:"code"`
	Desc string `toml:"desc"`
}

func (ac ChatConfig) String() string {
	jsonBytes, err := json.Marshal(ac)
	if err != nil {
		tilogs.L().Errorf("json marshal error %v", err)
		return ""
	}
	return string(jsonBytes)
}
