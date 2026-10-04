package redispool

import (
	"fmt"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	gm "github.com/rcrowley/go-metrics"

	"github.com/nghichtu91/platform/share/planx/metrics"
)

type RedisSimpleCfg struct {
	RedisServer        string
	DbSelected         int
	DbPwd              string
	DbAddrForMetrics   string
	IsSentinel         bool
	SentinelMasterName string // 用于监控统计的redis IP地址或域名
}

type Config struct {
	RedisSimpleCfg
	Capacity       int
	MaxCapacity    int
	IdleTimeout    time.Duration
	ConnectTimeout time.Duration // 建立链接需要的时间
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
}

const (
	DefaultRedisPoolCapacity = 100
	DefaultMaxCapacity       = 3000
	DefaultReadTimeout       = 5 * time.Second
	DefaultWriteTimeout      = 5 * time.Second
)

// TODO 基于pool.WaitCout() WaitTime() 发展趋势统计调整Capacity
func NewSimpleRedisPool(pooName string, simpleCfg RedisSimpleCfg, capacity int) IPool {
	tilogs.L().Infof("DB Setup with cfg %v", simpleCfg)
	// 去掉ip地址的.和:
	addr := strings.ReplaceAll(simpleCfg.DbAddrForMetrics, ".", "_")
	simpleCfg.DbAddrForMetrics = strings.ReplaceAll(addr, ":", "_")
	cfg := Config{
		RedisSimpleCfg: simpleCfg,
		Capacity:       capacity,
		MaxCapacity:    DefaultMaxCapacity,
		IdleTimeout:    5 * time.Minute,
		ConnectTimeout: 2000 * time.Millisecond,
		ReadTimeout:    DefaultReadTimeout,
		WriteTimeout:   DefaultWriteTimeout,
	}

	pool := NewRedisPool(cfg, pooName)
	go poolStatistic(pooName, pool)
	return pool
}

func poolStatistic(poolName string, pool IPool) {
	defer func() {
		if err := recover(); err != nil {
			tilogs.L().Errorf("[RedisPoolStatistic] recover error %v", err)
		}
	}()

	NewGauge := func(name string) gm.Gauge {
		return metrics.NewGauge(fmt.Sprintf("%s.RedisPool.%s", poolName, name))
	}
	poolActive := NewGauge("Active")
	poolIdle := NewGauge("Idle")
	poolMaxCap := NewGauge("MaxCap")
	poolMaxIdle := NewGauge("MaxIdle")
	poolGet := NewGauge("Get")
	poolPut := NewGauge("Put")
	poolDial := NewGauge("Dial")
	poolClose := NewGauge("Close")

	poolStateTick := time.NewTicker(5 * time.Second)
	tilogs.L().Debugf("poolStateTick Start")
	for {
		select {
		case <-poolStateTick.C:
			if pool.IsClosed() {
				tilogs.L().Debugf("poolStateTick stop")
				poolStateTick.Stop()
				return
			}
			active, idle, maxCap, maxIdle := pool.Stats()
			snmp := pool.Snmp()
			poolActive.Update(active)
			poolIdle.Update(idle)
			poolMaxCap.Update(maxCap)
			poolMaxIdle.Update(maxIdle)
			poolGet.Update(int64(snmp.Get))
			poolPut.Update(int64(snmp.Put))
			poolDial.Update(int64(snmp.Dial))
			poolClose.Update(int64(snmp.Close))
		}
	}
}
