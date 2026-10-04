// gm_tools config struct
package config

import (
	//"fmt"
	//"github.com/nghichtu91/platform/share/planx/etcd"
	//"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/config"
)

type Config struct {
	LocalConfig
	config.GidConfig
}

type LocalConfig struct {
	EtcdEndpoint   []string `toml:"etcd_endpoint"`
	ServerEtcdRoot string   `toml:"ServerEtcdRoot"`
	DevopsEtcdRoot string   `toml:"DevopsEtcdRoot"`
	Gid            uint     `toml:"Gid"`
	Port           string   `toml:"Port"`
	DbDriver       string   `toml:"DbDriver"`
	DbUser         string   `toml:"DbUser"`
	DbPwd          string   `toml:"DbPwd"`
	DbDatabase     string   `toml:"DbDatabase"`
	DbUrl          string   `toml:"DbUrl"`
}

var Cfg Config

//type ChatWebEtcdConfig struct {
//	// 端口
//	Port string `toml:"Port"`
//	// 数据库
//	DbDriver   string `etcd3:"DBDriver"`
//	DbUser     string `etcd3:"DbUser"`
//	DbPwd      string `etcd3:"DbPwd"`
//	DbDatabase string `etcd3:"DbDatabase"`
//	DbUrl      string `etcd3:"DbUrl"`
//	// oss
//	CloudDbDriver    string `etcd3:"CloudDbDriver"` // "S3" OR "aliyun"
//	HotDataBucket    string `etcd3:"HotDataBucket"`
//	BattleDataBucket string `etcd3:"BattleDataBucket"`
//	AliOssRegion     string `etcd3:"oss_endpoint"`
//	AliOssAccessKey  string `etcd3:"oss_accesstkey"`
//	AliOssSecretKey  string `etcd3:"oss_secretkey"`
//	// chat server
//	ChatServer string `etcd3:"ChatServer"`
//	// esearch
//	EsIndex string `etcd3:"EsIndex"`
//	EsUrl   string `etcd3:"EsUrl"`
//}

//type ChatServerEtcdConfig struct {
//	HttpUrl string `etcd3:"HttpUrl"`
//	EsIndex string `etcd3:"EsIndex"`
//	EsUrl   string `etcd3:"EsUrl"`
//}

//func LoadChatWebEtcd(etcdRoot string, gid uint) *ChatWebEtcdConfig {
//	key := fmt.Sprintf("%s/%d/defaults", etcdRoot, gid)
//	config := &ChatWebEtcdConfig{}
//	err := etcd.Bind(key, config)
//	if err != nil {
//		tilogs.L().Errorf("load config from etcd err ChatWeb %d, %v", gid, err)
//		return nil
//	}
//	return config
//}

//func LoadChatServerEtcd(etcdRoot string, gid uint, gmId string) *ChatServerEtcdConfig {
//	key := fmt.Sprintf("%s/%d/gmtools/%s", etcdRoot, gid, gmId)
//	config := &ChatServerEtcdConfig{}
//	err := etcd.Bind(key, config)
//	if err != nil {
//		tilogs.L().Errorf("load config from etcd err gid %d, [%v]", gid, err)
//		return nil
//	}
//	return config
//}
