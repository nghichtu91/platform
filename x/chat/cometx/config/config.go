package config

import (
	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	common_config "github.com/nghichtu91/platform/share/x/common/config"
)

var (

	//Cfg Chat配置
	Cfg ChatConfig

	// etcd config
	EtcdConf common_config.GidConfig

	// run mode
	RunMode = "dev" // TODO 改为从etcd拿
)

func IsDevelopMode() bool {
	return RunMode == "dev"
}

func LoadConfig(configName string) bool {
	config.NewConfigToml(configName, &Cfg)
	tilogs.SetTiLogger(tilogs.L().With(tilogs.TagServerId, Cfg.ServerId).With(tilogs.TagGid, Cfg.Gid))
	tilogs.L().Infof("chat config: %s", Cfg.String())
	if err := etcd.InitEtcd(Cfg.EtcdEndPoint); err != nil {
		tilogs.L().Errorf("etcd.InitEtcd err %s", err.Error())
		return false
	}
	return loadConfigFromEtcd()
}

func loadConfigFromEtcd() bool {
	pConfig := common_config.InitLoadGidConfig(Cfg.EtcdDevops, Cfg.Gid)
	if pConfig == nil {
		return false
	}
	EtcdConf = *pConfig
	return true
}
