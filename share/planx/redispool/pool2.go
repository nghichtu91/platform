package redispool

import (
	"sync/atomic"

	"github.com/gomodule/redigo/redis"
)

/*
	使用redigo/redis，多线程安全，redis连接池，可增可减
*/

type Pool struct {
	Name             string
	resPool          *redis.Pool
	conf             Config
	dbAddrForMetrics string

	snmp Snmp // 统计信息
}

type Snmp struct {
	Get   uint64 // 从池子中获取连接计数
	Put   uint64 // 释放连接到池子中计数
	Dial  uint64 // 建立新连接计数
	Close uint64 // 关闭旧连接计数
}

// AddGet 增加获取连接计数
func (s *Snmp) AddGet() { atomic.AddUint64(&s.Get, 1) }

// AddPut 增加释放连接计数
func (s *Snmp) AddPut() { atomic.AddUint64(&s.Put, 1) }

// AddDial 增加建立新连接计数
func (s *Snmp) AddDial() { atomic.AddUint64(&s.Dial, 1) }

// AddClose 增加关闭旧连接计数
func (s *Snmp) AddClose() { atomic.AddUint64(&s.Close, 1) }

// Clone 复制一个新的Snmp
func (s *Snmp) Clone() Snmp {
	return Snmp{
		Get:   atomic.LoadUint64(&s.Get),
		Put:   atomic.LoadUint64(&s.Put),
		Dial:  atomic.LoadUint64(&s.Dial),
		Close: atomic.LoadUint64(&s.Close),
	}
}

// Snmp 返回当前的Snmp
func (p *Pool) Snmp() Snmp {
	return p.snmp.Clone()
}

func (p *Pool) IsNil() bool {
	return p.resPool == nil
}

func (p *Pool) IsClosed() bool {
	return p.resPool == nil
}

func (p *Pool) Close() {
	if p.IsNil() {
		return
	}
	p.resPool.Close()
	p.resPool = nil
}

func (p *Pool) Get() RedisPoolConn {
	return p.GetDBConn()
}

func (p *Pool) GetDBConn() RedisPoolConn {
	p.snmp.AddGet()
	return RedisPoolConn{VrConn{p.resPool.Get()}, p, p.dbAddrForMetrics}
}

func (p *Pool) CloseConn(c RedisPoolConn) {
	p.snmp.AddPut()
	c.VrConn.Close()
}

func (p *Pool) Stats() (active, idle, maxCap, maxIdle int64) {
	s := p.resPool.Stats()
	return int64(s.ActiveCount), int64(s.IdleCount), int64(p.resPool.MaxActive), int64(p.resPool.MaxIdle)
}
