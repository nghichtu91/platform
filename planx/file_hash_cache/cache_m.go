package file_hash_cache

import (
	"sync"
)

// MCache 支持并发的缓存，M代表使用mutex
// 性能要求不高但又有并发需求，可以用这个
type MCache struct {
	*Cache

	mutex sync.RWMutex
}

func NewMCache(path string) *MCache {
	return &MCache{
		Cache: NewCache(path),
	}
}

func (mc *MCache) Save() error {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.Cache.Save()
}

func (mc *MCache) Load() error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	return mc.Cache.Load()
}

func (mc *MCache) UpdateFiles(filenames ...string) (changed bool, err error) {
	// TODO 锁的粒度有点大，先这样吧
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	return mc.Cache.UpdateFiles(filenames...)
}

func (mc *MCache) GetChangedInfo() *ChangeInfo {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	return mc.Cache.GetChangedInfo()
}
