package config

import (
	"crypto/tls"

	"github.com/nghichtu91/platform/share/planx/config"
)

// LoadTlsConfig 使用读取config.toml文件一样的逻辑，读取对应目录下server.crt和server.key
func LoadTlsConfig() (*tls.Config, error) {
	crt, key := config.NewConfigPath(tlsCertFilename), config.NewConfigPath(tlsKeyFilename)
	cert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}, nil
}
