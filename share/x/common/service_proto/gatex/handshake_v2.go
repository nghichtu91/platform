package gatex

import (
	"bytes"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/x/common/msg"
)

var (
	buffPool = msg.NewBuffPool(MaxHandShakePackageSize)                                            // buff池
	bsPool   = sync.Pool{New: func() interface{} { return make([]byte, MaxHandShakePackageSize) }} // byte slice池，后续可以考虑用ring buff
)

var (
	ErrClientHandShakeInfoIsNil = errors.New("client hand shake info is nil")
	ErrHandShakeResultFail      = errors.New("hand shake resp result is fail")
)

const (
	MaxHandShakePackageSize = msg.MaxHandShakePackageSize // 握手包最大长度
	HandShakeTimeout        = msg.HandShakeTimeout        // 握手超时
	HandShakeResultOK       = msg.HandShakeResultOK
)

// SendHandShakeV2 向gatex发送TCP握手请求
func SendHandShakeV2(ip string, tlsConfig *tls.Config, info *msg.ClientHandShakeInfo) (token string, conn *tls.Conn, err error) {
	if info == nil {
		err = ErrClientHandShakeInfoIsNil
		return
	}

	conn, err = tls.Dial("tcp", ip, tlsConfig)
	if err != nil {
		return
	}

	err = conn.SetReadDeadline(time.Now().Add(HandShakeTimeout * time.Second))
	if err != nil {
		return
	}

	buff := buffPool.Get().(*bytes.Buffer)
	buff.Reset()
	defer buffPool.Put(buff)

	// 发握手信息
	info.Marshal(buff)
	_, err = buff.WriteTo(conn)
	if err != nil {
		return
	}
	// tilogs.L().Debugf("handshake info sent")

	// 收握手信息
	bs := bsPool.Get().([]byte)
	bs = make([]byte, MaxHandShakePackageSize)
	defer bsPool.Put(bs)

	// 由于msg来自gate，去掉限制
	n, err := conn.Read(bs)
	if err != nil {
		return
	}
	// tilogs.L().Debugf("handshake resp received")

	buff.Reset()
	buff.Write(bs[:n])
	resp := &msg.HandShakeResp{}
	err = resp.Unmarshal(buff)
	if err != nil {
		return
	}

	// tilogs.L().Debugf("resp %+v", resp)

	if resp.Result != HandShakeResultOK {
		err = ErrHandShakeResultFail
		return
	}

	token = resp.AccountID

	return
}

// ReceiveHandShakeV2 接收客户端的握手信息
// 只简化了处理原始数据的逻辑，其他逻辑需要自行实现
func ReceiveHandShakeV2(conn net.Conn) (*msg.ClientHandShakeInfo, error) {
	// 收握手信息
	bs := bsPool.Get().([]byte)
	bs = make([]byte, MaxHandShakePackageSize)
	defer bsPool.Put(bs)

	// 这里已经限制了读取的最大长度，就不需要limit reader了
	n, err := conn.Read(bs)
	if err != nil {
		return nil, err
	}

	buff := buffPool.Get().(*bytes.Buffer)
	buff.Reset()
	defer buffPool.Put(buff)

	buff.Write(bs[:n])
	info := &msg.ClientHandShakeInfo{}
	err = info.Unmarshal(buff)
	if err != nil {
		return nil, err
	}

	return info, nil
}
