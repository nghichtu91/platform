package gatex

import (
	"bufio"
	"crypto/tls"
	"errors"
	"net/textproto"

	"github.com/nghichtu91/platform/share/x/common/check"
)

const (
	HandShakeFail = "fail"
)

var (
	ErrResultIsNotFail = errors.New("handshake result is invalid")
)

var (
	fakeToken = []byte("servercheck,999999\n")
)

// TcpFailHandShake 此次TCP握手一定会失败
// 如果result返回"fail"，说明连通性正常
// 其他情况为连接失败
func TcpFailHandShake(ip string, tlsConfig *tls.Config) (err error) {
	_, err = TcpFailHandshakeEvents(ip, tlsConfig)
	return
}

const (
	EventTLSHandshake = "tls_handshake"
	EventWriteToken   = "write_token"
	EventReadLine     = "read_line"
	EventClose        = "close"
)

func TcpFailHandshakeEvents(ip string, tlsConfig *tls.Config) (events check.Events, err error) {
	var (
		conn   *tls.Conn
		result string
	)

	var eventBuilder = check.CreateEventsBuilder()
	conn, err = tls.Dial("tcp", ip, tlsConfig)
	if err != nil {
		return
	}
	defer func() {
		_ = conn.Close()
		eventBuilder.AddEvent(EventClose)
		events = eventBuilder.Events()
	}()
	eventBuilder.AddEvent(EventTLSHandshake)

	_, err = conn.Write(fakeToken) // loginToken
	if err != nil {
		return
	}
	eventBuilder.AddEvent(EventWriteToken)

	reader := bufio.NewReader(conn)
	tr := textproto.NewReader(reader)
	result, err = tr.ReadLine()
	if err != nil {
		return
	}
	eventBuilder.AddEvent(EventReadLine)

	if result != HandShakeFail {
		err = ErrResultIsNotFail
	}

	return
}
