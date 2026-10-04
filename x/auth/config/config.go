package config

import (
	"github.com/nghichtu91/platform/share/x/common/config"
)

var (
	Cfg    AppConfig
	GidCfg *config.GidConfig

	LoginUrl *string // auth的内网地址和端口以及login的url
)
