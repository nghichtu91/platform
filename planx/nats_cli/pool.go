package nats_cli

import (
	"sync"

	"github.com/nghichtu91/platform/share/planx/nats_cli/pb"
)

var (
	pbPool = sync.Pool{New: func() interface{} {
		return new(pb.NatsWrapper)
	}}
)

// GetNatsWrapper 从缓存池中获取NatsWrapper
func GetNatsWrapper(w *pb.NatsWrapper) *pb.NatsWrapper {
	w = pbPool.Get().(*pb.NatsWrapper)
	return w
}

// PutNatsWrapper 将NatsWrapper放回
func PutNatsWrapper(w *pb.NatsWrapper) {
	w.Reset()
	pbPool.Put(w)
}
