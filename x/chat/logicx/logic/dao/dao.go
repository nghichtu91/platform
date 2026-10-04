package dao

import (
	"github.com/nghichtu91/platform/share/planx/redispool"
	chat_common "github.com/nghichtu91/platform/share/planx/servers/chat"
	"github.com/nghichtu91/platform/share/x/chat/logicx/config"
)

// Dao dao.
type Dao struct {
	redis redispool.IPool
}

// New new a dao and return.
func New() *Dao {
	return &Dao{
		redis: newRedis(),
	}
}

func newRedis() redispool.IPool {
	return chat_common.NewRedisPool("chatredis",
		config.EtcdConf.ChatxRedisAddr,
		config.EtcdConf.ChatxRedisDbPwd,
		config.EtcdConf.ChatxRedisDb,
		config.EtcdConf.ChatxRedisAddr, &config.EtcdConf, redispool.DefaultRedisPoolCapacity)
}
