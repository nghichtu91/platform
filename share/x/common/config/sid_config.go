package config

import (
	"fmt"
	"strconv"

	"github.com/nghichtu91/platform/share/planx/redispool"

	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/redis_helper"
)

type ShardConfig struct {
	Sid                  uint
	ServerLogicStartTime string `etcd3:"serstarttime"`
	AccountDb            int    `etcd3:"account_db"`
	RankDb               int    `etcd3:"rank_db"`
	ListenPostAddr       string `etcd3:"listen_post_port"`
	GrpcAddr             string
	DebugAddr            string `etcd3:"debug_addr"` // cheat的对外监听端口，prod无效
	PikaSentinelEnable   string `etcd3:"PikaSentinelEnable"`
	PikaMasterName       string `etcd3:"PikaMasterName"`
	AccountPikaDBUrl     string `etcd3:"AccountPikaDBUrl"`
	AccountPikaDBPwd     string `etcd3:"AccountPikaDBPwd"`
	AccountDBUrl         string
	AccountDBPwd         string
	GameDbCfg            redispool.RedisSimpleCfg
}

type GrpcConfig struct {
	Address string
}

type AwsConfig struct {
}

func LoadShardsConfig(etcdRoot string, gidCfg GidConfig, sids []uint) []ShardConfig {
	shards := make([]ShardConfig, 0)
	for _, sid := range sids {
		shardConfig := &ShardConfig{}
		shardConfig.Sid = sid
		shardKey := fmt.Sprintf("%s/%d/gamex/%d", etcdRoot, gidCfg.Gid, sid)
		err := etcd.Bind(shardKey, shardConfig)
		if err != nil {
			tilogs.L().Errorf("load config from etcd err sid %d %v", sid, err)
			return nil
		}
		if !shardConfig.postShardLoad(sid, gidCfg) {
			tilogs.L().Errorf("load config  from etcd, postShardLoad failed sid %d %v", sid, err)
			return nil
		}
		shards = append(shards, *shardConfig)
		//tilogs.L().Infof("load shard config %v", shardConfig)
	}
	return shards
}

func LoadShardConfig(etcdRoot string, gidCfg GidConfig, sid uint) *ShardConfig {
	shardConfig := &ShardConfig{}
	shardConfig.Sid = sid
	shardKey := fmt.Sprintf("%s/%d/gamex/%d", etcdRoot, gidCfg.Gid, sid)
	err := etcd.Bind(shardKey, shardConfig)
	if err != nil {
		tilogs.L().Errorf("load config from etcd err sid %d %v", sid, err)
		return nil
	}
	if !shardConfig.postShardLoad(sid, gidCfg) {
		tilogs.L().Errorf("load config  from etcd, postShardLoad failed sid %d %v", sid, err)
		return nil
	}
	//tilogs.L().Infof("load shard config %+v", shardConfig)
	return shardConfig
}

func (cfg *ShardConfig) postShardLoad(sid uint, gidCfg GidConfig) bool {
	cfg.Sid = sid
	accountPikaUrl := gidCfg.AccountPikaDBUrl
	accountPikaPwd := gidCfg.AccountPikaDBPwd
	pikaSentinelEnable := gidCfg.PikaSentinelEnable
	pikaMasterName := gidCfg.PikaMasterName
	accountDB := gidCfg.AccountDb
	if cfg.AccountPikaDBUrl != "" {
		accountPikaUrl = cfg.AccountPikaDBUrl
		accountPikaPwd = cfg.AccountPikaDBPwd
		pikaSentinelEnable = cfg.PikaSentinelEnable
		pikaMasterName = cfg.PikaMasterName
		accountDB = cfg.AccountDb
	}
	if accountPikaUrl == "" {
		return false
	}
	cfg.GameDbCfg = redis_helper.GenRedisPoolCfg(accountPikaUrl, accountDB, accountPikaPwd, accountPikaUrl,
		gidCfg.RedisSentinelEnable, gidCfg.RedisMasterName, pikaSentinelEnable, pikaMasterName)

	cfg.ListenPostAddr = ":" + cfg.ListenPostAddr
	return true
}

func LoadShardConfigInServer(etcdRoot string, gid uint, sid uint) *consts.ShardInfoInServer {
	key := fmt.Sprintf("%s/%d/shards/%d", etcdRoot, gid, sid)
	cfg := &consts.ShardInfoInServer{}
	err := etcd.Bind(key, cfg)
	if err != nil {
		tilogs.L().Errorf("get shard config from server %v", err)
		return nil
	}
	return cfg
}

func LoadAllShardConfigInServer(etcdRoot string, gid uint) map[uint]consts.ShardInfoInServer {
	serverMap := make(map[string]consts.ShardInfoInServer, 1024)
	key := fmt.Sprintf("%s/%d/shards", etcdRoot, gid)
	err := etcd.Bind(key, serverMap)
	if err != nil {
		tilogs.L().Errorf("bind server map ")
		return nil
	}
	ret := make(map[uint]consts.ShardInfoInServer, len(serverMap))
	for k, v := range serverMap {
		sid, err := strconv.Atoi(k)
		if err != nil {
			tilogs.L().Warnf("parse server sid err, may not sid dir, %v", err)
			continue
		}
		ret[uint(sid)] = v
	}
	return ret
}
