package msg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	MaxHandShakePackageSize = 1024            // 握手包最大长度
	HandShakeTimeout        = 8 * time.Second // 一次握手操作的最大读写超时时长
	HandShakeResultOK       = "ok"
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, MaxHandShakePackageSize))
		},
	}

	ErrNilConn = errors.New("nil conn")
)

func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func putBuffer(buffer *bytes.Buffer) {
	if buffer == nil {
		return
	}
	buffer.Reset()
	bufferPool.Put(buffer)
}

func WriteHandShakeReq(writer io.Writer, loginToken, gzipLimit string) error {
	var info = RobotClientHandshakeInfo(loginToken, gzipLimit)
	var buffer = getBuffer()
	defer putBuffer(buffer)
	info.Marshal(buffer)
	_, err := buffer.WriteTo(writer)
	return err
}

func ReadHandshakeResp(read io.Reader) (resp HandShakeResp, err error) {
	var buffer = getBuffer()
	defer putBuffer(buffer)
	// 读取返回的结果
	buffer.Grow(MaxHandShakePackageSize)
	var buf = buffer.Bytes()[:MaxHandShakePackageSize]
	var n int
	n, err = read.Read(buf)
	if err != nil {
		return
	}
	buffer.Write(buf[:n])
	err = resp.Unmarshal(buffer)
	return
}

func GateHandShake(conn net.Conn, loginToken, gzipLimit string) (acid string, err error) {
	if conn == nil {
		return "", ErrNilConn
	}
	// 设置读写超时
	err = conn.SetDeadline(time.Now().Add(HandShakeTimeout))
	if err != nil {
		err = fmt.Errorf("set deadline error %w", err)
		return
	}
	defer func() {
		// 必须要重置定时器, 否则如果在超出定时器时间后 读写 会超时
		err = conn.SetDeadline(time.Time{})
	}()
	// 写入握手数据
	writeErr := WriteHandShakeReq(conn, loginToken, gzipLimit)
	if writeErr != nil {
		err = fmt.Errorf("handshake write error %w", writeErr)
		return
	}
	// 读取握手的返回值
	resp, respErr := ReadHandshakeResp(conn)
	if respErr != nil {
		err = fmt.Errorf("handshake read resp error %w", respErr)
		return
	}
	// 检查返回结果
	if resp.Result != HandShakeResultOK {
		err = fmt.Errorf("handshake resp is %+v, not valid", resp)
		return
	}
	return resp.AccountID, nil
}
