package redispool

import (
	"sync"
	"testing"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
)

func Test_Pool2(t *testing.T) {
	zaplog.InitDebugLog()
	p := createPool()
	var waitter sync.WaitGroup
	for i := 0; i < 1000; i++ {
		waitter.Add(1)
		go func(i int) {
			defer waitter.Done()
			c := p.Get()
			ac, _, _, _ := p.Stats()
			snmp := p.Snmp()
			if c.IsNil() {
				t.Logf("%d nil", i)
			} else {
				t.Logf("%d {%d, %+v }", i, ac, snmp)
			}
			time.Sleep(time.Second)
			c.Close()
		}(i)
	}
	waitter.Wait()

	time.Sleep(time.Millisecond * 300)
	ac, _, _, _ := p.Stats()
	t.Log(ac, p.Snmp())

	time.Sleep(time.Millisecond * 300)
	ac, _, _, _ = p.Stats()
	t.Log(ac, p.Snmp())

	time.Sleep(time.Millisecond * 300)
	ac, _, _, _ = p.Stats()
	t.Log(ac, p.Snmp())

	p.Close()
}

func createPool() IPool {
	return NewRedisPool(Config{
		RedisSimpleCfg: RedisSimpleCfg{
			RedisServer:        "127.0.0.1:6379",
			DbSelected:         10,
			DbPwd:              "",
			DbAddrForMetrics:   "",
			IsSentinel:         false,
			SentinelMasterName: "",
		},
		Capacity:    10,
		MaxCapacity: 100,
		IdleTimeout: 1 * time.Second,

		ConnectTimeout: 1 * time.Second,
		ReadTimeout:    1 * time.Second,
		WriteTimeout:   1 * time.Second,
	}, "test")
}
