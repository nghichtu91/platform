package config

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type CrossConfig struct {
	CrossDbAddr    string `etcd3:"CrossPikaDbAddr"`
	CrossDbPwd     string `etcd3:"CrossPikaDbPwd"`
	CrossDbNum     int    `etcd3:"CrossPikaDbNum"`
	CrossMemberNum int    `etcd3:"cross_member_num"`
	Serverid       int    `etcd3:"serverid"`
}

func LoadCrossConfigFromEtcd(etcdRoot string, gid uint, crossIdx int, isProd bool) *CrossConfig {
	crossKey := fmt.Sprintf("%s/%d/crossx/crossx%d", etcdRoot, gid, crossIdx)
	if isProd {
		crossKey = fmt.Sprintf("%s/%d/crossx/crossx0%d", etcdRoot, gid, crossIdx)
	}
	crossConfig := &CrossConfig{}
	err := etcd.Bind(crossKey, crossConfig)
	if err != nil {
		tilogs.L().Errorf("load cross config from etcd err: %v", err)
		return nil
	}
	if crossIdx == 0 && crossConfig.CrossMemberNum <= 0 || crossIdx > 0 && crossConfig.Serverid <= 0 {
		if isProd {
			crossKey = fmt.Sprintf("%s/%d/crossx/crossx00%d", etcdRoot, gid, crossIdx)
		}
	}
	err = etcd.Bind(crossKey, crossConfig)
	if err != nil {
		tilogs.L().Errorf("load cross config from etcd err: %v", err)
		return nil
	}
	tilogs.L().Infof("load cross config %v", crossConfig)
	return crossConfig
}
