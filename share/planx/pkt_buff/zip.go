package pkt_buff

import (
	"bytes"
	"compress/gzip"
)

// ZipBuff 用于压缩和解压
type ZipBuff struct {
	b *bytes.Buffer
	r *gzip.Reader
	w *gzip.Writer
}

func NewZipBuff() *ZipBuff {
	b := bytes.NewBuffer(make([]byte, 0, defaultBuffSize))
	return &ZipBuff{
		b: b,
		w: gzip.NewWriter(b),
	}
}

func GetZipBuff() *ZipBuff {
	return zipPool.Get().(*ZipBuff)
}

func PutZipWBuff(zb *ZipBuff) {
	zb.b.Reset()
	zb.w.Reset(zb.b)
	zipPool.Put(zb)
}

func (zb *ZipBuff) Zip(b []byte) error {
	var (
		err error
	)
	_, err = zb.w.Write(b)
	if err != nil {
		return err
	}

	err = zb.w.Close()
	if err != nil {
		return err
	}

	return nil
}
