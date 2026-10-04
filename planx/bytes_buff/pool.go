package bytes_buff

import (
	"bytes"
	"sync"
)

var buffPool = sync.Pool{New: func() interface{} {
	return bytes.NewBuffer(make([]byte, 0, 8196))
}}

func Get() *bytes.Buffer {
	return buffPool.Get().(*bytes.Buffer)
}

func Put(b *bytes.Buffer) {
	b.Reset()
	buffPool.Put(b)
}

// ReadOutAndPut 使用这个接口将buf缓存内容取出，并放回
func ReadOutAndPut(b *bytes.Buffer) []byte {
	data := make([]byte, len(b.Bytes()))
	copy(data, b.Bytes())

	b.Reset()
	buffPool.Put(b)

	return data
}
