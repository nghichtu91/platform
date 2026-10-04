package savedbwrapper

import (
	"sync"
)

var (
	usingKeys sync.Map
)

// CheckKeyDuplicate 检查当前goroutine中key是否重复
// 如果当前key不重复，将key加入到全局变量中
func (save *SaveDB) CheckKeyDuplicate(key string) bool {
	_, ok := usingKeys.Load(key)

	// 未使用过，是合法的key
	if !ok {
		usingKeys.Store(key, struct{}{})
		save.bindKeys = append(save.bindKeys, key)
	}
	return ok
}

func (save *SaveDB) RemoveKeysOnQuit() {
	for _, key := range save.bindKeys {
		usingKeys.Delete(key)
	}
}
