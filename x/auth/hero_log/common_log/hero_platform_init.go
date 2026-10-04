package common_log

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	sls "github.com/aliyun/aliyun-log-go-sdk"

	"github.com/nghichtu91/platform/share/planx/config"
)

type HeroPlatformLogConfig struct {
	IsOpen          bool
	BDCIsTestEnv    bool
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	ProjectName     string
	LogStoreName    string
	ServerId        string
	AppKey          string
	Platform        string
	TimeZone        string
}

var (
	Client      *sls.Client
	Cfg         *HeroPlatformLogConfig
	cfgFileName = "herolog.toml"
)

func Start(sid string, isProd bool) {
	// 若为海外服，则bdc不启用

	config.NewConfigToml(cfgFileName, &Cfg)
	if Cfg == nil {
		tilogs.L().Errorf("NewConfigToml fail ,HeroPlatformLogConfig is nil")
		return
	}
	time := timeutil.Now()
	_, offset := time.Zone()
	Cfg.TimeZone = fmt.Sprintf("UTC+%d:00", offset/planx.SecondsEveryDay)
	var heroPlatform = "SERVER"
	if Cfg.BDCIsTestEnv {
		heroPlatform = "SERVER_TEST"
	}

	Cfg.Platform = heroPlatform
	Cfg.ServerId = fmt.Sprint(sid)
	tilogs.L().Debugf("Cfg is %v", Cfg)

	Client = &sls.Client{
		Endpoint:        Cfg.Endpoint,
		AccessKeyID:     Cfg.AccessKeyID,
		AccessKeySecret: Cfg.AccessKeySecret,
	}
	if !timeutil.IsAsiaShanghaiTZ() { // 海外服
		return
	}
	Cfg.IsOpen = true
	start()
}

func Stop() {
	if Cfg == nil {
		return
	}
	quitChan <- true
	wait.Wait()
	tilogs.L().Infof("hero platform closed ok")
}
