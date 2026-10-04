package config

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/config"
)

var (
	Cfg Config // Gift起服起服配置。
)

// LoadGiftConfigAndInit 加载Gift服务起服所需配置并执行相关初始化。
func LoadGiftConfigAndInit(c *cli.Context) error {
	cfgPath := c.String(ConfigFlagFull)
	// logicLogPath := c.String("logiclog")

	// 从指定路径获取配置文件内容。
	loadInfo := config.NewConfigToml(cfgPath, &Cfg)
	if loadInfo == nil {
		return fmt.Errorf("LoadGiftConfigAndInit read file error: %s", cfgPath)
	}
	tilogs.SetTiLogger(tilogs.L().With(tilogs.TagServerId, Cfg.GiftCfg.ServerID))

	// 配置文件Json化为字符串。
	cfgJson, jsonErr := Cfg.ToString()
	if jsonErr != nil {
		return fmt.Errorf("LoadGiftConfigAndInit %v", jsonErr)
	} else {
		tilogs.L().Infof("LoadGiftConfigAndInit load config: %s", cfgJson)
	}

	// Gift无需服务发现，所以无需启动ETCD。

	// TODO 有埋点需求时，需要加入埋点。
	return nil
}

// ToString 获取配置的Json化输出。
func (cfg Config) ToString() (string, error) {
	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("ToString json error: %v", err)
	}
	return string(jsonBytes), nil
}
