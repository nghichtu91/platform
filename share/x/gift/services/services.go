package services

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/fvbock/endless"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/gift/config"
	"github.com/nghichtu91/platform/share/x/gift/services/giftcode_service"

	"github.com/gin-gonic/gin"
)

// StartServices 开启服务。
//
// Param-g: 受理Https请求的gin服务。
func StartServices(waitGroup *util.WaitGroupWrapper) error {
	// 初始化gin。
	g := gin.Default()

	// 注册GiftCode模块的API受理项。
	if err := giftcode_service.Service(g); err != nil {
		return fmt.Errorf("StartServices giftcode_service.Service err: %v", err)
	}

	// 启动gin服务。
	ginStartErr := make(chan error, 1)
	waitGroup.WrapRetErr(func(c chan error) {
		tilogs.L().Infof("StartServices https port: %v", config.Cfg.GiftCfg.GinPort)

		// 加载CA认证证书。
		pool := x509.NewCertPool()
		caCrt, errRead := ioutil.ReadFile(config.Cfg.GiftCfg.CACertFile)
		if errRead != nil {
			c <- errRead
			return
		}
		pool.AppendCertsFromPEM(caCrt)

		// 设置TLS配置项。
		server := endless.NewServer(config.Cfg.GiftCfg.GinPort, g)
		server.TLSConfig = &tls.Config{
			ClientCAs:  pool,
			ClientAuth: tls.RequireAndVerifyClientCert,
		}

		// 启动HTTPS监听。
		err := server.ListenAndServeTLS(config.Cfg.GiftCfg.ServerCertFile, config.Cfg.GiftCfg.ServerKeyFile)
		if err != nil {
			tilogs.L().Errorf("ListenAndServeTLS HttpsPort %s err %v", config.Cfg.GiftCfg.GinPort, err)
			c <- err
			return
		}
	}, ginStartErr)
	if !util.CheckGoroutineStartErr(ginStartErr, "StartServices https") {
		return fmt.Errorf("gin.Engine ListenAndServeTLS fail")
	}

	return nil
}
