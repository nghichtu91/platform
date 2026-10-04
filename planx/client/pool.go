package client

import (
	"sync"
)

var (
	// 处理server-client头，长度13
	headerPool = sync.Pool{New: func() interface{} {
		return make([]byte, 13)
	}}
)
