package models

import (
	"strconv"

	"github.com/nghichtu91/platform/share/planx/redis_helper"

	"github.com/nghichtu91/platform/share/x/common/config"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/redispool"
)

func newRedisPool(pooName, server, password string, dbselectCfg string, dbAddrForMetrics string, gidCfg *config.GidConfig) redispool.IPool {
	db_id, err := strconv.Atoi(dbselectCfg)
	if err != nil {
		tilogs.L().Warnf("newRedisPool can't read ", dbselectCfg, ". default is 0")
		db_id = 0
	}
	tilogs.L().Debugf("newRedisPool read %s it is %d", dbselectCfg, db_id)
	return redispool.NewSimpleRedisPool(pooName, redis_helper.GenRedisPoolCfg(server, db_id, password, dbAddrForMetrics,
		gidCfg.RedisSentinelEnable, gidCfg.RedisMasterName, gidCfg.PikaSentinelEnable, gidCfg.PikaMasterName),
		redispool.DefaultRedisPoolCapacity)
}
