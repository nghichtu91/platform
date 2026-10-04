package servers

import (
	"io"
	"net"
	"time"

	log "github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/gatex/config"
)

// 处理负载均衡的健康检查
func serveLSB(conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(
		time.Second * time.Duration(config.Cfg.GateConfig.LSBDeadLine)))

	buf := make([]byte, 4)
	n, err := io.ReadFull(conn, buf)
	log.L().Debugf("serve lsb connection, n:%d, err:%v", n, err)
}
