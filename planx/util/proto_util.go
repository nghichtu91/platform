package util

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func CompressProto(data []byte) []byte {
	b := ByteBuffPool.Get().(*bytes.Buffer)
	w := GZipWPool.Get().(*gzip.Writer)
	b.Reset()
	w.Reset(b)
	w.Write(data)
	w.Close()
	defer ByteBuffPool.Put(b)
	defer GZipWPool.Put(w)

	copyRet := make([]byte, len(b.Bytes()))
	copy(copyRet, b.Bytes())
	return copyRet
}

func UnCompressProto(data []byte) []byte {
	b := ByteBuffPool.Get().(*bytes.Buffer)
	defer ByteBuffPool.Put(b)
	b.Reset()
	b.Write(data)
	r := GZipRPool.Get().(*gzip.Reader)
	defer GZipRPool.Put(r)
	err := r.Reset(b)
	if err != nil {
		tilogs.L().Errorf("ungzip NewReader error %v", err)
		return nil
	}

	unData, err := ioutil.ReadAll(r)
	if err != nil {
		tilogs.L().Errorf("ungzip ReadAll error %v", err)
		return nil
	}
	return unData
}
