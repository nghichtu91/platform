package ginhelper

import (
	"github.com/gin-gonic/gin"
	ucfg "github.com/nghichtu91/platform/share/planx/config"
)

func MakeGinGMEngine() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(SentryGinLog())
	return engine
}

// MakeGinEngine 需要记录accesslog的engine，目前auth, notice使用
func MakeGinEngine(configFile string) (engine *gin.Engine, exitfun func()) {
	engine = gin.New()

	engine.Use(gin.Recovery())
	engine.Use(SentryGinLog())
	// engine.Use(MetricHandler())
	//engine.Use(ginhelper.LoggerWithWriter())
	//engine.Use(ginhelper.NginxLoggerWithWriter())
	var gh gin.HandlerFunc
	var exitfunlog func()
	ucfg.NewConfig(configFile, false, func(lcfgname string, cmd ucfg.LoadCmd) error {
		gh, exitfunlog = NgixLoggerToFile(lcfgname)
		return nil
	})
	exitfun = func() {
		if exitfunlog != nil {
			exitfunlog()
		}
	}
	engine.Use(gh)
	//engine.Use(limit.CheckIdentity())
	//engine.Use(limit.RateLimit())
	//engine.Use(statsCCU())
	return engine, exitfun
}

func MakeGinEngineNoLog() (engine *gin.Engine) {
	engine = gin.New()

	engine.Use(gin.Recovery())
	engine.Use(SentryGinLog())

	return engine
}
