package tracing

import (
	"bytes"
	"sync"
)

var (
	buffPool = sync.Pool{New: func() interface{} {
		return new(bytes.Buffer)
	}}
	readerPool = sync.Pool{New: func() interface{} {
		return new(bytes.Reader)
	}}
)
