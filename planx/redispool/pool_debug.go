package redispool

import (
	"strings"

	"github.com/gomodule/redigo/redis"
)

type PoolDebug struct {
}

func (p *PoolDebug) IsNil() bool {
	return false
}
func (p *PoolDebug) IsClosed() bool {
	return false
}
func (p *PoolDebug) Close() {

}
func (p *PoolDebug) Get() RedisPoolConn {
	return RedisPoolConn{
		VrConn: VrConn{&redisConnDebug{}},
		p:      p,
	}
}
func (p *PoolDebug) GetDBConn() RedisPoolConn {
	return RedisPoolConn{
		VrConn: VrConn{&redisConnDebug{}},
		p:      p,
	}
}
func (p *PoolDebug) CloseConn(RedisPoolConn) {

}
func (p *PoolDebug) Stats() (capacity, available, maxCap, waitCount int64) {
	return 0, 0, 0, 0
}
func (p *PoolDebug) Snmp() Snmp { return Snmp{} }

type redisConnDebug struct {
}

func (conn *redisConnDebug) Close() error {
	return nil
}
func (conn *redisConnDebug) Err() error {
	return nil
}
func (conn *redisConnDebug) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	cmd := strings.ToUpper(commandName)
	switch cmd {
	case "EXISTS":
		return int64(0), nil
	case "HGET":
		return nil, nil
	case "HGETALL":
		return []interface{}{}, nil
	}
	return nil, nil
}
func (conn *redisConnDebug) DoCmdBuffer(cb redis.Conn, transaction bool) (reply interface{}, err error) {
	return nil, nil
}
func (conn *redisConnDebug) Send(commandName string, args ...interface{}) error {
	return nil
}
func (conn *redisConnDebug) Flush() error {
	return nil
}
func (conn *redisConnDebug) Receive() (reply interface{}, err error) {
	return nil, nil
}
