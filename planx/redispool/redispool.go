package redispool

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FZambia/sentinel"
	"github.com/gomodule/redigo/redis"

	"github.com/nghichtu91/platform/share/planx/metrics"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type IPool interface {
	IsNil() bool
	IsClosed() bool
	Close()
	Get() RedisPoolConn
	GetDBConn() RedisPoolConn
	CloseConn(RedisPoolConn)
	Stats() (active, idle, maxCap, maxIdle int64)
	Snmp() Snmp
}

type RedisPoolConn struct {
	VrConn
	p                IPool
	dbAddrForMetrics string
}

var NilRPConn = RedisPoolConn{VrConn: NilConn}

func (rpc RedisPoolConn) RawConn() redis.Conn {
	return rpc.VrConn.RawConn()
}

func (rpc RedisPoolConn) Close() {
	if rpc.IsNil() {
		tilogs.L().Warnf("RedisPoolConn send Nil conn back to pool[1] %v", rpc)
		return
	}
	if rpc.p == nil {
		tilogs.L().Warnf("RedisPoolConn send Nil conn back to pool[2] %v", rpc)
		return
	}
	rpc.p.CloseConn(rpc)
}

func (rpc RedisPoolConn) Do(metricKey, commandName string, args ...interface{}) (reply interface{}, err error) {
	// 统计时间
	if metricKey != "" {
		name := fmt.Sprintf("dbstat.%s.%s", rpc.dbAddrForMetrics, metricKey)
		defer metrics.TimeStatisticsWithThreshold(name, metricKey, time.Now().UnixNano(), int64(time.Millisecond))
		metrics.CountStatistics(name)
	}
	return rpc.Conn.Do(commandName, args...)
}

func (rpc RedisPoolConn) DoCmdBuffer(metricKey string, transaction bool) (reply interface{}, err error) {
	// 统计时间
	if metricKey != "" {
		name := fmt.Sprintf("dbstat.%s.%s", rpc.dbAddrForMetrics, metricKey)
		defer metrics.TimeStatisticsWithThreshold(name, metricKey, time.Now().UnixNano(), int64(time.Millisecond))
		metrics.CountStatistics(name)
	}
	if transaction {
		return rpc.Conn.Do("EXEC")
	} else {
		return rpc.Conn.Do("")
	}
}

// ResourceConn adapts a Redigo connection to a Vitess Resource.
type VrConn struct {
	redis.Conn
}

var NilConn = VrConn{Conn: nil}

func (r VrConn) IsNil() bool {
	return r.Conn == nil
}

func (r VrConn) Close() {
	if r.IsNil() {
		return
	}
	r.Conn.Close()
}

func (r VrConn) RawConn() redis.Conn {
	return r.Conn
}

func NewRedisPool(cfg Config, name string) (p IPool) {
	tilogs.L().Debugf("NewRedisPool cfg %v", cfg)
	_p := &Pool{
		Name:             name,
		conf:             cfg,
		dbAddrForMetrics: cfg.DbAddrForMetrics,
	}
	if cfg.IsSentinel {
		s := &sentinel.Sentinel{
			Addrs:      strings.Split(cfg.RedisSimpleCfg.RedisServer, ","),
			MasterName: cfg.RedisSimpleCfg.SentinelMasterName,
			Dial: func(addr string) (redis.Conn, error) {
				c, err := createRedisSentinelConn(addr, cfg)
				if err != nil {
					return nil, err
				}
				return c, nil
			},
		}
		_p.resPool = &redis.Pool{
			MaxIdle:     cfg.Capacity,
			MaxActive:   cfg.MaxCapacity,
			Wait:        true,
			IdleTimeout: cfg.IdleTimeout,
			Dial: func() (redis.Conn, error) {
				masterAddr, err := s.MasterAddr()
				if err != nil {
					return nil, err
				}
				c, err := createRedisConn(masterAddr, cfg)
				if err != nil {
					return nil, err
				}
				_p.snmp.AddDial()
				return _p.WarpConn(c), nil
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				if !sentinel.TestRole(c, "master") {
					return errors.New("Role check failed")
				} else {
					return nil
				}
			},
		}
		_p.dbAddrForMetrics = cfg.SentinelMasterName
	} else {
		_p.resPool = &redis.Pool{
			Dial: func() (redis.Conn, error) {
				c, err := createRedisConn(cfg.RedisSimpleCfg.RedisServer, cfg)
				if err != nil {
					return nil, err
				}
				_p.snmp.AddDial()
				return _p.WarpConn(c), err
			},
			MaxIdle:     cfg.Capacity,    // Capacity
			MaxActive:   cfg.MaxCapacity, // MaxCapacity
			IdleTimeout: cfg.IdleTimeout, // idleTimeout
			Wait:        true,
		}
	}
	p = _p
	tilogs.L().Infof("New RedisPool %s[%d] %v Valid", cfg.RedisServer, cfg.DbSelected, cfg.IsSentinel)
	return p
}

func createRedisConn(serverAddr string, cfg Config) (redis.Conn, error) {
	c, err := redis.Dial("tcp", serverAddr,
		redis.DialConnectTimeout(cfg.ConnectTimeout),
		redis.DialReadTimeout(cfg.ReadTimeout),
		redis.DialWriteTimeout(cfg.WriteTimeout),
		redis.DialDatabase(cfg.RedisSimpleCfg.DbSelected),
		redis.DialPassword(cfg.RedisSimpleCfg.DbPwd),
	)
	return c, err
}

func createRedisSentinelConn(serverAddr string, cfg Config) (redis.Conn, error) {
	c, err := redis.Dial("tcp", serverAddr,
		redis.DialConnectTimeout(cfg.ConnectTimeout),
		redis.DialReadTimeout(cfg.ReadTimeout),
		redis.DialWriteTimeout(cfg.WriteTimeout),
		// redis.DialDatabase(cfg.RedisSimpleCfg.DbSelected),
		// redis.DialPassword(cfg.RedisSimpleCfg.DbPwd),
	)
	return c, err
}

func DialRedisConn(cfg Config) redis.Conn {
	addr := cfg.RedisServer
	if cfg.IsSentinel {
		s := &sentinel.Sentinel{
			Addrs:      strings.Split(cfg.RedisServer, ","),
			MasterName: cfg.SentinelMasterName,
			Dial: func(addr string) (redis.Conn, error) {
				c, err := createRedisSentinelConn(addr, cfg)
				if err != nil {
					return nil, err
				}
				return c, nil
			},
		}
		masterAddr, err := s.MasterAddr()
		if err != nil {
			tilogs.L().Errorf("sentinel MasterAddr err %v, config %v", err, cfg)
			return nil
		}
		addr = masterAddr
	}
	tilogs.L().Debugf("DialRedisConn cfg.IsSentinel:%v addr:%v", cfg.IsSentinel, addr)
	c, err := createRedisConn(addr, cfg)
	if err != nil {
		tilogs.L().Errorf("DialRedisConn createRedisConn err %v, config %v", err, cfg)
		return nil
	}
	return c

}

// 默认使用从库的函数
func DialRedisSlaveConn(cfg Config) redis.Conn {
	addr := cfg.RedisServer
	if cfg.IsSentinel {
		s := &sentinel.Sentinel{
			Addrs:      strings.Split(cfg.RedisServer, ","),
			MasterName: cfg.SentinelMasterName,
			Dial: func(addr string) (redis.Conn, error) {
				c, err := createRedisSentinelConn(addr, cfg)
				if err != nil {
					return nil, err
				}
				return c, nil
			},
		}
		SlaveAddrs, err := s.SlaveAddrs()
		if err != nil || len(SlaveAddrs) <= 0 {
			tilogs.L().Errorf("sentinel MasterAddr err %v, config %v %v", err, cfg, SlaveAddrs)
			return nil
		}
		//默认选第一个
		addr = SlaveAddrs[0]
	}
	tilogs.L().Debugf("DialRedisSlaveConn cfg.IsSentinel:%v addr:%v", cfg.IsSentinel, addr)
	c, err := createRedisConn(addr, cfg)
	if err != nil {
		tilogs.L().Errorf("DialRedisConn createRedisConn err %v, config %v", err, cfg)
		return nil
	}
	return c

}

// WarpConn warp a redis.Conn to VrConn
func (p *Pool) WarpConn(c redis.Conn) redis.Conn {
	return &statisticConn{Conn: c, onClose: p.OnConnClose}
}

// OnConnClose set on close callback
func (p *Pool) OnConnClose(c redis.Conn) {
	_ = c
	p.snmp.AddClose()
}

type statisticConn struct {
	redis.Conn
	onClose func(redis.Conn)
}

// Close on close callback
func (sc *statisticConn) Close() error {
	if sc.onClose != nil {
		sc.onClose(sc.Conn)
	}
	return sc.Conn.Close()
}
