package tiredis

import (
	"github.com/gomodule/redigo/redis"
)

// RedisDB 简单封装的redis
// 用于快速实现模块功能
type RedisDB struct {
	pool *redis.Pool
}

// RedisDBCfg redis初始化配置
type RedisDBCfg struct {
	Addr string
	Pwd  string
	DB   int
}

// NewRedisConn 生成redis实例
func NewRedisConn(cfg *RedisDBCfg) (*RedisDB, error) {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cfg.Addr)
			if err != nil {
				return nil, err
			}
			if cfg.Pwd != "" {
				if _, err := c.Do("AUTH", cfg.Pwd); err != nil {
					c.Close()
					return nil, err
				}
			}
			if _, err := c.Do("SELECT", cfg.DB); err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
	}

	return &RedisDB{pool: pool}, nil
}

// Close 关闭链接
func (rdb *RedisDB) Close() error {
	if rdb.pool != nil {
		return rdb.pool.Close()
	}
	return nil
}

// Incr 原子性的返回自增的int值
// 如果对应key之前不存在，返回1
//
// 实现IAtomicDB接口
func (rdb *RedisDB) Incr(key string) (int64, error) {
	conn := rdb.pool.Get()
	defer conn.Close()
	return redis.Int64(conn.Do("INCR", key))
}
