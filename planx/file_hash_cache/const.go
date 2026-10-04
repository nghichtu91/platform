package file_hash_cache

import (
	"errors"
)

var (
	ErrCacheFileNameMissing = errors.New("cache file name missing") // 未设置cache相关文件名
	ErrCCacheCmdSendTimeout = errors.New("cache cmd send time out") // 发送cmd请求超时
	ErrCCacheCmdRetTimeout  = errors.New("cache cmd ret time out")  // 接收cmd请求回复超时
)
