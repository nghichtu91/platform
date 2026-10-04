package redis_helper

import (
	"fmt"
	"strings"

	"github.com/nghichtu91/platform/share/planx/redispool"
)

func GenRedisPoolCfg(redisServer string, dbSelected int, dbPwd, dbAddrForMetrics string,
	redisSentinelEnable, redisMasterName, pikaSentinelEnable, pikaMasterName string) redispool.RedisSimpleCfg {
	rss := strings.Split(redisServer, "@")
	if len(rss) == 2 {
		ret := redispool.RedisSimpleCfg{
			RedisServer:      rss[1],
			DbSelected:       dbSelected,
			DbPwd:            dbPwd,
			DbAddrForMetrics: dbAddrForMetrics,
		}
		switch strings.ToLower(rss[0]) {
		case "redis":
			if strings.ToLower(redisSentinelEnable) == "true" {
				ret.IsSentinel = true
				ret.SentinelMasterName = redisMasterName
			}
			return ret
		case "pika":
			if strings.ToLower(pikaSentinelEnable) == "true" {
				ret.IsSentinel = true
				ret.SentinelMasterName = pikaMasterName
			}
			return ret
		default:
			panic(fmt.Sprintf("dbserver db type error, %s", redisServer))
		}
	} else {
		// 临时代码，等各个大区都使用带@的配置方式后，恢复下面的panic
		ret := redispool.RedisSimpleCfg{
			RedisServer:      redisServer,
			DbSelected:       dbSelected,
			DbPwd:            dbPwd,
			DbAddrForMetrics: dbAddrForMetrics,
		}
		return ret
		//panic(fmt.Sprintf("dbserver format error, %s", redisServer))
	}
}

func SetupRedis(cfg redispool.RedisSimpleCfg) redispool.IPool {
	return SetupRedisByCap(cfg, 100)
}

func SetupRedisForSimple(cfg redispool.RedisSimpleCfg) redispool.IPool {
	return SetupRedisByCap(cfg, 1)
}

func SetupRedisByCap(cfg redispool.RedisSimpleCfg, cap int) redispool.IPool {
	return redispool.NewSimpleRedisPool("gamexredis", cfg, cap)
}
