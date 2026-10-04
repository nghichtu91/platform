package protogen

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/nghichtu91/platform/share/planx/util"
)

var (
	raw = []byte{7, 11, 15, 9, 23, 112, 232, 12, 15, 15, 15, 9, 0, 223, 111, 9, 7, 12, 16,
		22, 11, 11, 22, 0, 13, 0, 11, 11, 11, 11, 11, 12, 15, 15, 15, 15}
	compressed = []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 98, 231, 230, 231, 20, 47, 120, 193, 195, 207, 207, 207, 201, 112, 63, 159, 147, 157, 71, 64, 140, 155, 91, 140, 129, 151, 129, 27, 4, 64, 226, 252, 128, 0, 0, 0, 255, 255, 168, 251, 139, 242, 36, 0, 0, 0}
)

func genP() *ClientPackage {
	return &ClientPackage{
		MessageId: proto.Uint32(uint32(ChatOperation_OpRawData)),
		RawData: []byte{7, 11, 15, 9, 23, 112, 232, 12, 15, 15, 15, 9, 0, 223, 111, 9, 7, 12, 16,
			22, 11, 11, 22, 0, 13, 0, 11, 11, 11, 11, 11, 12, 15, 15, 15, 15},
	}
}

func TestClientPackage_CompressAndDecompress(t *testing.T) {
	t.Run("TryCompress", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			before := genP()
			before.TryCompressOld()
			after := genP()
			after.TryCompress()

			assert.True(t, bytes.Equal(before.GetRawData(), after.GetRawData()), fmt.Sprintf("\nbefore %v\nafter %v", before.GetRawData(), after.GetRawData()))
		}
	})

	t.Run("Compress and Decompress", func(t *testing.T) {
		c, dc := new(ClientPackage), new(ClientPackage)
		c.RawData = make([]byte, 0, MaxBodySize)
		dc.RawData = make([]byte, 0, MaxBodySize)

		reset := func() {
			c.RawData = c.RawData[:0]
			c.RawData = append(c.RawData, raw...)
		}

		resetDC := func() {
			dc.RawData = dc.RawData[:0]
			dc.RawData = append(dc.RawData, compressed...)
		}

		oldCompressed := util.CompressProto(raw)
		oldRaw := util.UnCompressProto(compressed)

		// 验证新旧压缩和解压得到的数据一致
		reset()
		c.Compress()
		assert.True(t, bytes.Equal(oldCompressed, c.GetRawData()))

		resetDC()
		c.Decompress()
		assert.True(t, bytes.Equal(oldRaw, c.GetRawData()))

		for i := 0; i < 100; i++ {
			reset()
			c.Compress()

			assert.True(t, bytes.Equal(compressed, c.GetRawData()))

			resetDC()
			dc.Decompress()

			assert.True(t, bytes.Equal(raw, dc.GetRawData()))
		}
	})
}

func BenchmarkClientPackage_CompressAndDecompress(b *testing.B) {
	p := new(ClientPackage)
	p.MessageId = proto.Uint32(uint32(ChatOperation_OpRawData))
	p.RawData = make([]byte, 0, MaxBodySize)

	reset := func(p *ClientPackage, bs []byte) {
		p.RawData = p.RawData[:len(bs)]
		copy(p.RawData, bs)
	}

	b.Run("TryCompressOld", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			reset(p, raw)
			p.TryCompressOld()
		}
	})

	b.Run("TryCompress", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			reset(p, raw)
			p.TryCompress()
		}
	})

	b.Run("CompressOld", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			reset(p, raw)
			p.RawData = util.CompressProto(p.RawData)
		}
	})

	b.Run("Compress", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			reset(p, raw)
			p.Compress()
		}
	})

	b.Run("DeCompressOld", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			reset(p, compressed)
			p.RawData = util.UnCompressProto(p.RawData)
		}
	})

	b.Run("DeCompress", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			reset(p, compressed)
			p.Decompress()
		}
	})
}

func BenchmarkAppendAndCopy(b *testing.B) {
	p := new(ClientPackage)
	p.MessageId = proto.Uint32(uint32(ChatOperation_OpRawData))
	p.RawData = make([]byte, 0, MaxBodySize)

	resetCopy := func(p *ClientPackage, bs []byte) {
		p.RawData = p.RawData[:len(bs)]
		copy(p.RawData, bs)
	}

	resetAppend := func(p *ClientPackage, bs []byte) {
		p.RawData = p.RawData[:0]
		p.RawData = append(p.RawData, bs...)
	}

	b.Run("Append", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			resetAppend(p, raw)
		}
	})

	b.Run("Copy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resetCopy(p, raw)
		}
	})
}
