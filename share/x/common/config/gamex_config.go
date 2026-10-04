package config

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/nghichtu91/platform/share/planx/signalhandler"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

// config from etcd3
type ConfigFromEtcd struct {
	*GidConfig
	Shard *ShardConfig
	*GmToolConfig
	MergeConfig // 需要用默认值
}

func LoadConfigFromEtcd(w *util.WaitGroupWrapper, etcdRoot, etcdServer string, gid, sid uint) *ConfigFromEtcd {
	retConfig := &ConfigFromEtcd{}
	retConfig.GidConfig = InitLoadGidConfig(etcdRoot, gid)
	if retConfig.GidConfig == nil {
		return nil
	}
	// cross scene不读相关配置
	if sid < consts.XSceneStartID {
		retConfig.Shard = LoadShardConfig(etcdRoot, *retConfig.GidConfig, sid)
		if retConfig.Shard == nil {
			return nil
		}
		// retConfig.GmToolConfig = LoadGmToolConfig(etcdServer, gid)
		mcfg := LoadMergeConfig(etcdServer, gid, sid)
		if mcfg != nil {
			retConfig.MergeConfig = *mcfg
		}
	}

	if w != nil {
		retConfig.GidConfig.StartWatch(w, etcdRoot, gid)
		signalhandler.SignalKillFunc(func() { retConfig.GidConfig.StopConfigBattleCheck() })
	}

	return retConfig
}

func (c *ConfigFromEtcd) GetShardConfig() ShardConfig {
	return *c.Shard
}

type GmToolConfig struct {
	Apis map[string]string
}

func LoadGmToolConfig(etcdServer string, gid uint) *GmToolConfig {
	keyRoot := fmt.Sprintf("%s/%d/gm_tools/apis", etcdServer, gid)
	apis := make(map[string]string, 256)
	err := etcd.Bind(keyRoot, apis)
	if err != nil {
		tilogs.L().Errorf("", err)
	}
	rt := &GmToolConfig{
		Apis: apis,
	}
	tilogs.L().Infof("load gm_tool api %v", rt)
	return rt
}

// MergeConfig 合服相关的配置
// server/gid/merges/sid
type MergeConfig struct {
	MergedShardsStr string `etcd3:"merged_shards"`
	MergedShardIDs  []uint // 合服区间的uint形式，遍历用
	MergedUID       int    `etcd3:"merged_uid"` // 合服uid，用于合服后shardid相关数据读写处理
}

func LoadMergeConfig(etcdServer string, gid, sid uint) *MergeConfig {
	key := fmt.Sprintf("%s/%d/%s/%d", etcdServer, gid, etcd.Merges, sid)
	cfg := &MergeConfig{}
	err := etcd.Bind(key, cfg)
	if err != nil {
		// 没有进行过合服，返回空
		tilogs.L().Warnf("load merge config from etcd err, %s", err.Error())
		cfg.MergedShardIDs = make([]uint, 0, 1)
	}

	tilogs.L().Infof("load merge config from etcd %+v", cfg)
	return cfg
}

func GetShardEtcdKey(etcdRoot string, gid uint, sid uint) string {
	return fmt.Sprintf("%s/%d/shards/%d", etcdRoot, gid, sid)
}

func GetShardRunState(etcdRoot string, gid uint, sid uint) string {
	return fmt.Sprintf("%s/%d/shards/%d/runState", etcdRoot, gid, sid)
}

func GetShardCCUKey(etcdRoot string, gid uint, sid uint) string {
	return fmt.Sprintf("%s/%d/shards/%d/ccu", etcdRoot, gid, sid)
}

func GetShardsKey(etcdRoot string, gid uint) string {
	return fmt.Sprintf("%s/%d/shards", etcdRoot, gid)
}
