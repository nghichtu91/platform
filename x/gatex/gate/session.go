package gate

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/astaxie/beego/cache"
)

var (
	ErrNotExist = errors.New("key not exist")
	ErrNotImp   = errors.New("not implement")
)

type SessionCache struct {
	cacheMap atomic.Value
}

func NewSessionCache() cache.Cache {
	var sessionCache = &SessionCache{}
	sessionCache.cacheMap.Store(&sync.Map{})
	return sessionCache
}

func (s *SessionCache) Map() *sync.Map {
	m, _ := s.cacheMap.Load().(*sync.Map)
	return m
}

func (s *SessionCache) Get(key string) interface{} {
	val, _ := s.Map().Load(key)
	return val
}

func (s *SessionCache) GetMulti(keys []string) []interface{} {
	var ret = make([]interface{}, 0, len(keys))
	m := s.Map()
	for _, key := range keys {
		val, _ := m.Load(key)
		ret = append(ret, val)
	}
	return ret
}

// Put 这里的key, 不支持超时
func (s *SessionCache) Put(key string, val interface{}, _ time.Duration) error {
	m := s.Map()
	m.Store(key, val)
	return nil
}

func (s *SessionCache) Delete(key string) error {
	m := s.Map()
	_, find := m.LoadAndDelete(key)
	if !find {
		return ErrNotExist
	}
	return nil
}

func (s *SessionCache) Incr(_ string) error {
	return ErrNotImp
}

func (s *SessionCache) Decr(_ string) error {
	return ErrNotImp
}

func (s *SessionCache) IsExist(key string) bool {
	_, find := s.Map().Load(key)
	return find
}

func (s *SessionCache) ClearAll() error {
	s.cacheMap.Store(&sync.Map{})
	return nil
}

func (s *SessionCache) StartAndGC(_ string) error {
	return nil
}
