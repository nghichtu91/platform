package cmds

import (
	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/ntsdk"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	authConfig "github.com/nghichtu91/platform/share/x/auth/config"
	cconfig "github.com/nghichtu91/platform/share/x/common/config"
)

func InitAuthConfig(cfgName string) bool {
	cfgApp := config.NewConfigToml(cfgName, &authConfig.Cfg)
	if cfgApp == nil {
		tilogs.L().Errorf("Config Read Error\n")
		return false
	}
	tilogs.SetTiLogger(tilogs.L().With(tilogs.TagServerId, authConfig.Cfg.CommonCfg.ServerId).With(tilogs.TagGid, authConfig.Cfg.CommonCfg.Gid))
	tilogs.L().Infof("load config %s", authConfig.Cfg.String())
	// etcd
	if err := etcd.InitEtcd(authConfig.Cfg.CommonCfg.EtcdEndPoint); err != nil {
		tilogs.L().Errorf("etcd.InitEtcd err %s", err.Error())
		return false
	}
	tilogs.L().Infof("etcd start finish")
	authConfig.GidCfg = cconfig.InitLoadGidConfig(
		authConfig.Cfg.CommonCfg.EtcdRoot,
		authConfig.Cfg.CommonCfg.Gid)
	if authConfig.GidCfg == nil {
		tilogs.L().Errorf("etcd LoadGidConfig failed")
		return false
	}

	cfgApp = config.NewConfigToml(cfgName, &ntsdk.SdkNtCfg)
	if cfgApp == nil {
		tilogs.L().Errorf("SdkNtCfg Read Error\n")
		return false
	}
	return true
}
