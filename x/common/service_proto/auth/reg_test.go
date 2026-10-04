package auth_test

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nghichtu91/platform/share/x/common/msg"
	"github.com/nghichtu91/platform/share/x/common/service_proto/auth"
	"github.com/nghichtu91/platform/share/x/common/service_proto/gatex"
)

func TestRegRobotSDK(t *testing.T) {

	const GateConf = "../../../gatex/conf"

	doConn := func(requireHost bool) {
		token, err := auth.RegRobotSDK("http://127.0.0.1:8081", "testuid")
		assert.Nil(t, err)
		t.Logf("token %s", token)
		assert.NotEmpty(t, token)
		ip, loginToken, err := auth.GetGateRequireHost("http://127.0.0.1:8081", token, "120001", requireHost)
		assert.Nil(t, err)
		t.Logf("ip %s, loginToken %s", ip, loginToken)

		var cfg tls.Config
		cert, err := tls.LoadX509KeyPair(GateConf+"/server.crt", GateConf+"/server.key")
		assert.Nil(t, err)
		cfg.Certificates = []tls.Certificate{cert}
		cfg.InsecureSkipVerify = true

		info := msg.RobotClientHandshakeInfo(loginToken, "1024")

		acid, conn, err := gatex.SendHandShakeV2(ip, &cfg, &info)
		require.NoError(t, err)
		defer func(conn *tls.Conn) {
			err := conn.Close()
			if err != nil {
				t.Logf("close conn err %v", err)
			}
		}(conn)
		t.Logf("requireHost %v acid %s", requireHost, acid)
	}

	doConn(false)
	doConn(true)
}
