package chat

import (
	"strconv"

	"github.com/nghichtu91/platform/share/planx/redis_helper"
	common_config "github.com/nghichtu91/platform/share/x/common/config"

	"github.com/nghichtu91/platform/share/planx/metrics"

	"github.com/gomodule/redigo/redis"
	"github.com/nghichtu91/platform/share/planx/redispool"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
)

func NewRedisPool(pooName, server, password string, dbselectCfg string, dbAddrForMetrics string, gidCfg *common_config.GidConfig, capacity int) redispool.IPool {
	db_id, err := strconv.Atoi(dbselectCfg)
	if err != nil {
		log.L().Warnf("newRedisPool can't read ", dbselectCfg, ". default is 0")
		db_id = 0
	}
	log.L().Debugf("newRedisPool read %s it is %d", dbselectCfg, db_id)
	return redispool.NewSimpleRedisPool(pooName, redis_helper.GenRedisPoolCfg(
		server, db_id, password, dbAddrForMetrics,
		gidCfg.RedisSentinelEnable, gidCfg.RedisMasterName, gidCfg.PikaSentinelEnable, gidCfg.PikaMasterName), capacity)
}

func SetPlayerToken(chatDb redispool.IPool, userId, token string) error {
	_db := chatDb.Get()
	defer _db.Close()

	key := GetTokenKey(userId)
	_, err := redis.String(_db.Do(metrics.GetDBStatPrefix("chat", "SetPlayerToken", "SET"), "SET", key, token))
	if err != nil {
		return err
	}

	return nil
}

func DelPlayerToken(chatDb redispool.IPool, userId string) error {
	_db := chatDb.Get()
	defer _db.Close()

	key := GetTokenKey(userId)
	_, err := redis.String(_db.Do(metrics.GetDBStatPrefix("chat", "DelPlayerToken", "DEL"), "DEL", key))
	if err != nil {
		return err
	}

	return nil
}

func ExpirePlayerToken(chatDb redispool.IPool, userId string, expireTime int32) error {
	_db := chatDb.Get()
	defer _db.Close()

	key := GetTokenKey(userId)
	_, err := redis.Int64(_db.Do(metrics.GetDBStatPrefix("chat", "ExpirePlayerToken", "EXPIRE"), "EXPIRE", key, expireTime))
	if err != nil {
		return err
	}

	return nil
}
