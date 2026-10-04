package protogen

import (
	gb "bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"syscall"

	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx/metrics"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/chat/pkg/bufio"
	"github.com/nghichtu91/platform/share/x/chat/pkg/bytes"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

const (
	// MaxBodySize max proto body size
	MaxBodySize = uint32(1024 * 10)
)

const (
	// size
	_packSize      = 4
	_headerSize    = 4
	_compressSize  = 2
	_rawHeaderSize = _packSize + _headerSize + _compressSize
	_maxPackSize   = MaxBodySize + uint32(_rawHeaderSize)
	// offset
	_packOffset     = 0
	_headerOffset   = _packOffset + _packSize
	_compressOffset = _headerOffset + _headerSize
	_bodyOffset     = _compressOffset + _compressSize

	// compressLimit
	_compressLimit = 1024
)

var (
	// ErrProtoPackLen proto packet len error
	ErrProtoPackLen = errors.New("default server codec pack length error")
	// ErrProtoHeaderLen proto header len error
	ErrProtoHeaderLen = errors.New("default server codec header length error")
)

var (
	// ProtoReady proto ready
	ProtoReady = &ClientPackageWithSpan{
		ClientPackage: &ClientPackage{MessageId: proto.Uint32(uint32(ChatOperation_OpProtoReady))},
	}

	// ProtoFinish proto finish
	ProtoFinish = &ClientPackageWithSpan{
		ClientPackage: &ClientPackage{MessageId: proto.Uint32(uint32(ChatOperation_OpProtoFinish))},
	}
)

var (
	zipSetPool = sync.Pool{
		New: func() interface{} {
			buf := gb.NewBuffer(make([]byte, 0, MaxBodySize))
			return &zipSet{
				buf: buf,
				r:   new(gzip.Reader),
				w:   gzip.NewWriter(buf),
			}
		},
	}
)

// 可复用的gzip结构，处理不超过chat聊天限制的内容
type zipSet struct {
	buf *gb.Buffer
	w   *gzip.Writer
	r   *gzip.Reader
}

// GetZipSetR 从对象池获取一个用于解压的ZipSet
func GetZipSetR() *zipSet {
	zs := zipSetPool.Get().(*zipSet)
	return zs
}

// GetZipSetW 从对象池获取一个用于解压的ZipSet
func GetZipSetW() *zipSet {
	zs := zipSetPool.Get().(*zipSet)
	zs.w.Reset(zs.buf)
	return zs
}

// PutZipSetR 重置Reader并放回
func PutZipSetR(zs *zipSet) {
	zs.buf.Reset()
	zs.r.Close()
	zipSetPool.Put(zs)
}

// PutZipSetW 重置Writer并放回
func PutZipSetW(zs *zipSet) {
	zs.buf.Reset()
	zipSetPool.Put(zs)
}

// Zip 对data进行压缩，并返回压缩后的buffer
func (zs *zipSet) Zip(data []byte) *gb.Buffer {
	zs.w.Write(data)
	zs.w.Close()
	return zs.buf
}

// UnZip 对data解压，并返回解压后的buffer
func (zs *zipSet) UnZip(data []byte) *gb.Buffer {
	zs.r.Reset(gb.NewReader(data))
	io.Copy(zs.buf, zs.r)
	return zs.buf
}

type ClientPackageWithSpan struct {
	Span opentracing.Span
	*ClientPackage
}

type PushMsgReqWithSpan struct {
	Span opentracing.Span
	*PushMsgReq
}

type BroadcastRoomReqWithSpan struct {
	Span opentracing.Span
	*BroadcastRoomReq
}

// WriteTo write a proto to bytes writer.
func (p *ClientPackage) WriteTo(b *bytes.Writer) {
	var (
		packLen  uint32
		compress uint16
		buf      = b.Peek(_rawHeaderSize)
	)

	// body := p.RawData
	// if len(body) > _compressLimit {
	//	compress = 1
	//	body = chat.CompressProto(body)
	// }

	packLen = uint32(_rawHeaderSize) + uint32(len(p.RawData))

	binary.LittleEndian.PutUint32(buf[_packOffset:], packLen)
	binary.LittleEndian.PutUint32(buf[_headerOffset:], p.GetMessageId())
	binary.LittleEndian.PutUint16(buf[_compressOffset:], compress)
	if len(p.RawData) > 0 {
		b.Write(p.RawData)
	}
}

// 如果读端关闭--》 写端再写报错 broken pipe
// 如果写端关闭--》 读端读取会收到EOF
// 如果己方关闭socket后继续 读写 use of closed network connection
// ReadTCP read a proto from TCP reader.
func (p *ClientPackage) ReadTCP(rr *bufio.Reader) (err error) {
	var (
		bodyLen int
		packLen uint32
		buf     []byte
	)

	// 读取tcp的包的头4个字节 包大小
	if buf, err = rr.Pop(_rawHeaderSize); err != nil {
		if !errors.Is(err, syscall.ECONNRESET) {
			log.L().Warnf("read tcp error: %v", err)
		}
		return
	}

	// 之前用的小端 还是用之前的吧
	packLen = binary.LittleEndian.Uint32(buf[_packOffset:_packSize])
	if packLen > _maxPackSize {
		p.MessageId = proto.Uint32(binary.LittleEndian.Uint32(buf[_headerOffset:_compressOffset]))
		log.L().Warnf("read buf, packLen:%d, p.MessageId:%d", packLen, p.GetMessageId())
		return ErrProtoPackLen
	}

	p.MessageId = proto.Uint32(binary.LittleEndian.Uint32(buf[_headerOffset:_compressOffset]))
	compress := false
	if binary.LittleEndian.Uint16(buf[_compressOffset:_bodyOffset]) == 1 {
		compress = true
	}

	// 读取包体
	// p.RawData = p.RawData[:0]
	if bodyLen = int(packLen - uint32(_rawHeaderSize)); bodyLen > 0 {
		// p.RawData, err = rr.Pop(bodyLen)
		// if compress {
		// 	p.RawData = util.UnCompressProto(p.RawData)
		// }
		buf, err = rr.Pop(bodyLen)
		if err != nil {
			return err
		}
		if cap(p.RawData) >= len(buf) {
			p.RawData = p.RawData[0:len(buf)]
		} else {
			p.RawData = make([]byte, len(buf))
		}

		copy(p.RawData, buf)
		// p.RawData = append(p.RawData, buf...)
		if compress {
			p.Decompress()
		}
	}

	return
}

// WriteTCP write a proto to TCP writer.
func (p *ClientPackage) WriteTCP(wr *bufio.Writer) (err error) {
	if p.GetMessageId() == uint32(ChatOperation_OpRawData) { // jobx已经压缩过的协议
		// write without buffer, job contact proto into raw buffer
		_, err = wr.WriteRaw(p.RawData)
		metrics.ReqCountAndSizeStatistics(allmetrics.CometPrefix, ChatOperation_name[int32(p.GetMessageId())], len(p.RawData))
		return
	}

	var (
		buf      []byte
		packLen  uint32
		compress uint16
	)

	if len(p.RawData) > _compressLimit {
		compress = 1
		p.Compress()
	}

	packLen = uint32(_rawHeaderSize) + uint32(len(p.RawData))
	if buf, err = wr.Peek(_rawHeaderSize); err != nil {
		return
	}

	binary.LittleEndian.PutUint32(buf[_packOffset:], packLen)
	binary.LittleEndian.PutUint32(buf[_headerOffset:], p.GetMessageId())
	binary.LittleEndian.PutUint16(buf[_compressOffset:], compress)
	if len(p.RawData) > 0 {
		_, err = wr.Write(p.RawData)
	}

	metrics.ReqCountAndSizeStatistics(allmetrics.CometPrefix, ChatOperation_name[int32(p.GetMessageId())], int(packLen))
	return
}

// WriteTCPHeart write TCP heartbeat with room online.
func (p *ClientPackage) WriteTCPHeart(wr *bufio.Writer, online int32) (err error) {
	/*
		var (
			buf     []byte
			packLen int
		)
		packLen = _rawHeaderSize + _heartSize
		if buf, err = wr.Peek(packLen); err != nil {
			return
		}
		// header
		binary.LittleEndian.PutInt32(buf[_packOffset:], int32(packLen))
		binary.LittleEndian.PutInt16(buf[_headerOffset:], int16(_rawHeaderSize))
		binary.LittleEndian.PutInt16(buf[_verOffset:], int16(p.Ver))
		binary.LittleEndian.PutInt32(buf[_opOffset:], p.Op)
		binary.LittleEndian.PutInt32(buf[_seqOffset:], p.Seq)
		// body
		binary.LittleEndian.PutInt32(buf[_heartOffset:], online)
	*/
	return
}

/*
对S2CHistoryMsg 2004 /OpRawData 3002 两种协议进行压缩
*/
func (p *ClientPackage) TryCompressOld() {
	if p.GetMessageId() != uint32(ChatOperation_OpRawData) {
		return
	}

	body := util.CompressProto(p.RawData)
	packLen := uint32(_rawHeaderSize) + uint32(len(body))
	buf := make([]byte, _rawHeaderSize)

	binary.LittleEndian.PutUint32(buf[_packOffset:], packLen)
	binary.LittleEndian.PutUint32(buf[_headerOffset:], p.GetMessageId())
	binary.LittleEndian.PutUint16(buf[_compressOffset:], uint16(1))
	buf = append(buf, body...)
	p.RawData = buf
}

// TryCompress 对S2CHistoryMsg 2004 /OpRawData 3002 两种协议进行压缩
func (p *ClientPackage) TryCompress() {
	if p.GetMessageId() != uint32(ChatOperation_OpRawData) {
		return
	}

	zs := GetZipSetW()
	defer PutZipSetW(zs)

	buff := zs.Zip(p.RawData)
	packLen := uint32(_rawHeaderSize) + uint32(buff.Len())
	if cap(p.RawData) >= int(packLen) {
		p.RawData = p.RawData[0:packLen]
	} else {
		p.RawData = make([]byte, packLen)
	}
	binary.LittleEndian.PutUint32(p.RawData[_packOffset:], packLen)
	binary.LittleEndian.PutUint32(p.RawData[_headerOffset:], p.GetMessageId())
	binary.LittleEndian.PutUint16(p.RawData[_compressOffset:], uint16(1))
	copy(p.RawData[_bodyOffset:packLen], buff.Bytes())
}

// Compress 将p.RawData内容压缩并放回p.RawData
func (p *ClientPackage) Compress() {
	zs := GetZipSetW()
	defer PutZipSetW(zs)

	buff := zs.Zip(p.RawData)
	if cap(p.RawData) >= buff.Len() {
		p.RawData = p.RawData[0:buff.Len()]
	} else {
		p.RawData = make([]byte, buff.Len())
	}

	copy(p.RawData, buff.Bytes())
}

// Decompress 将p.RawData内容解压并放回p.RawData
// 注意这一步可能突破MaxBodySize的限制
func (p *ClientPackage) Decompress() {
	zs := GetZipSetR()
	defer PutZipSetR(zs)

	buff := zs.UnZip(p.RawData)
	if cap(p.RawData) >= buff.Len() {
		p.RawData = p.RawData[0:buff.Len()]
	} else {
		p.RawData = make([]byte, buff.Len())
	}

	copy(p.RawData, buff.Bytes())
	// p.RawData = append(p.RawData, buff.Bytes()...)
}
