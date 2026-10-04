package util

import (
	"bytes"
	"compress/gzip"
	"sync"

	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
)

const (
	packetBuffSize = 1024 * 16 // 设置默认缓冲区为16K
)

var (
	ByteBuffPool      sync.Pool
	GZipWPool         sync.Pool
	GZipRPool         sync.Pool
	SessionPacketPool sync.Pool
	PacketPool        sync.Pool
)

func init() {
	ByteBuffPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
	GZipWPool = sync.Pool{
		New: func() interface{} {
			var b bytes.Buffer
			return gzip.NewWriter(&b)
		},
	}
	GZipRPool = sync.Pool{
		New: func() interface{} {
			return new(gzip.Reader)
		},
	}
	SessionPacketPool = sync.Pool{
		New: func() interface{} {
			return new(pb.SessionPacket)
		},
	}
	PacketPool = sync.Pool{
		New: func() interface{} {
			return &pb.Packet{
				RawData: make([]byte, 0, packetBuffSize),
			}
		},
	}
}
