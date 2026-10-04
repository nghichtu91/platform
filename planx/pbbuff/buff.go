package pbbuff

import (
	"bytes"
	"io"
	"sync"

	"github.com/nghichtu91/platform/share/planx/math"
	"google.golang.org/protobuf/proto"
)

// 专门为protobuf设计的buffer池

const (
	defMaxSize  = 1024 * 1024
	defInitSize = 16 * 1024
)

var (
	// buffer池
	pool sync.Pool

	// marshal option
	pmo = new(proto.MarshalOptions)

	// buffer初始化大小
	initBuffSize int

	// buffer最大长度，超过此长度会被丢弃
	maxBuffSize int
)

func init() {
	initBuffSize = defInitSize
	maxBuffSize = defMaxSize
}

// Init 初始化pbbuff
// 如果不带参数，则使用默认的buffer大小
// 第一个参数为buffer初始化大小，第二个参数为buffer最大长度
func Init(opts ...int) {
	n := len(opts)
	if n > 1 {
		initBuffSize = math.Round2N(opts[0])
	}
	if n > 2 {
		maxBuffSize = math.Round2N(opts[1])
	}

	pool = sync.Pool{New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, initBuffSize))
	}}
}

// GetBuffer 获取buffer
func GetBuffer() *bytes.Buffer {
	buff := pool.Get().(*bytes.Buffer)
	buff.Reset()
	return buff
}

// PutBuffer 返还buffer
// 由于protobuf Marshal没有直接使用buffer内的slice
// 需要将Marshal后的内容传入，以确定是否需要丢弃buffer
func PutBuffer(buff *bytes.Buffer, pbData *[]byte) {
	size := math.Round2N(len(*pbData))
	if size <= maxBuffSize {
		if buff.Cap() < size {
			buff.Grow(size - buff.Cap())
		}
		pool.Put(buff)
	}
}

// MarshalTo 封装buff的Marshal方法
func MarshalTo(w io.Writer, msg proto.Message) error {
	var (
		outData []byte
		err     error
	)
	buff := GetBuffer()
	defer PutBuffer(buff, &outData)

	outData, err = pmo.MarshalAppend(buff.Bytes(), msg)
	if err != nil {
		return err
	}

	_, err = w.Write(outData)
	if err != nil {
		return err
	}

	return nil
}
