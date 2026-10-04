package allinone

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/iputil"
	"github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/x/auth/config_data"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/ginhelper"
	"github.com/nghichtu91/platform/share/planx/limit"
	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/logiclog"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/auth/cmds"
	authConfig "github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/auth/models"
	"github.com/nghichtu91/platform/share/x/auth/routers"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

func init() {
	cmds.Register(&cli.Command{
		Name:   "auth",
		Usage:  "开启auth功能",
		Action: Start,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config, c",
				Value: "config.toml",
				Usage: "Onland Configuration toml config, in {CWD}/conf/ or {AppPath}/conf",
			},
			cli.StringFlag{
				Name:  "logiclog, ll",
				Value: "logiclog.toml",
				Usage: "log player logic logs, in {CWD}/conf/ or {AppPath}/conf",
			},
			cli.StringFlag{
				Name:  "port, p",
				Usage: "httpport",
			},
			cli.StringFlag{
				Name:  "data, d",
				Value: "/conf",
				Usage: "data dictionary",
			},
		},
	})
}

func Start(c *cli.Context) {
	defer etcd.CloseEtcd() // etcd要最后关闭，否则一些反注册会失败
	defer signalhandler.OnClose()
	defer tilogs.PanicCatcher("auth Start panic")

	zaplog.InitZapLog("log.toml", map[string]string{
		"service": "auth",
		"subcmd":  "auth",
	}, "auth")
	tilogs.L().Infof("Start with local %s", time.Local.String())

	// config
	if !cmds.InitAuthConfig(c.String("config")) {
		tilogs.L().Errorf("InitAuthConfig failed")
		return
	}

	// logiclog
	if !logiclog.LoadGameLogic(c.String("logiclog")) {
		tilogs.L().Errorf("LoadGameLogic failed")
		return
	}

	// 加在运营配置文件
	config_data.LoadConfigData(c.String("data"))

	if !limit.Init() {
		return
	}

	if planx.IsRunProd(authConfig.GidCfg.RunMode) || planx.IsRunPrefTest(authConfig.GidCfg.RunMode) {
		gin.SetMode(gin.ReleaseMode)
	}

	// internalip
	internalIp := iputil.GetPrivateIP(authConfig.Cfg.CommonCfg.InternalIp)
	if internalIp == "" {
		tilogs.L().Errorf("config InternalIp error, %s", authConfig.Cfg.CommonCfg.InternalIp)
		return
	}

	// pprof
	pprofEtcdKey := etcd.GetPprofAddrKey(
		authConfig.Cfg.CommonCfg.EtcdServer, strconv.Itoa(int(authConfig.Cfg.CommonCfg.Gid)),
		etcd.Server_Auth, authConfig.Cfg.CommonCfg.ServerId)
	stop := net.PProfStart(pprofEtcdKey, internalIp)
	defer stop()

	var waitGroup util.WaitGroupWrapper
	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })

	// metrics
	signalhandler.SignalKillFunc(func() { metrics.Stop() })
	if !metrics.Start("metrics.toml", authConfig.GidCfg.Proj, allmetrics.PrefixAuthMetrics(
		authConfig.GidCfg.Gid, authConfig.Cfg.CommonCfg.ServerId)) {
		return
	}
	tilogs.L().Infof("metrics start finish")
	allmetrics.InitAuthMetrics()
	metrics.InitMetricUtil(authConfig.GidCfg.Gid, authConfig.Cfg.CommonCfg.ServerId)

	if err := models.InitDb(&authConfig.Cfg.CommonCfg, authConfig.GidCfg); err != nil {
		tilogs.L().Errorf("init db  Error %v", err)
		return
	}
	models.InitLoginRedis(&authConfig.Cfg.CommonCfg)
	err := models.InitNumberIDInRedis()
	if err != nil {
		tilogs.L().Errorf("init numberID in redis err by %v", err.Error())
		return
	}

	signalhandler.SignalKillFunc(func() { models.StopShardInfo() })
	err = models.InitShardInfo()
	if err != nil {
		tilogs.L().Errorf("models.InitShardInfo err: %s", err.Error())
		return
	}

	_port := authConfig.Cfg.CommonCfg.Httpport
	if c.String("port") != "" {
		_port = c.String("port")
	}
	if authConfig.Cfg.CommonCfg.EnableHttpTLS {
		_port = authConfig.Cfg.CommonCfg.HttpsPort
	}
	// ver
	reg_ser := etcd.NewRegServer(
		authConfig.Cfg.CommonCfg.EtcdServer,
		[]string{fmt.Sprintf("%d", authConfig.Cfg.CommonCfg.Gid)},
		etcd.Ser_Auth, _port)
	signalhandler.SignalKillFunc(func() { reg_ser.UnRegServerVer() })
	if err := reg_ser.RegServerVer(version.GetVersion()); err != nil {
		tilogs.L().Errorf("RegServerVer err %s", err.Error())
		return
	}

	// gate mgr
	models.GateMgr.Init()
	signalhandler.SignalKillFunc(func() { models.GateMgr.Stop() })
	err = models.GateMgr.Start()
	if err != nil {
		tilogs.L().Errorf("GateMgr start err by %v", err.Error())
		return
	}

	// gonggao
	signalhandler.SignalKillFunc(func() { models.CloseWatchGongGao() })
	models.StartWatchGongGao(&waitGroup)

	// whiteListPwd
	signalhandler.SignalKillFunc(func() { models.CloseWatchWhiteListPwd() })
	models.StartWatchWhiteListPwd(&waitGroup)

	// 外网监听
	r, exitfun := ginhelper.MakeGinEngine("accesslog.xml")
	r.Use(ginhelper.MetricHandler())
	signalhandler.SignalKillFunc(func() { exitfun() })
	routers.RegAuth(r)
	routers.RegLogin(r)

	chanPublicErr := make(chan error, 1)
	waitGroup.WrapRetErr(func(c chan error) {
		if authConfig.Cfg.CommonCfg.EnableHttpTLS {
			tilogs.L().Infof("auth serve http tls port: %s", authConfig.Cfg.CommonCfg.HttpsPort)
			if err := endless.ListenAndServeTLS(authConfig.Cfg.CommonCfg.HttpsPort,
				authConfig.Cfg.CommonCfg.HttpsCertFile,
				authConfig.Cfg.CommonCfg.HttpsKeyFile,
				r); err != nil {
				tilogs.L().Errorf("ListenAndServeTLS HttpsPort %s err %s", authConfig.Cfg.CommonCfg.HttpsPort, err.Error())
				c <- err
			}
		} else {
			tilogs.L().Infof("auth serve http port: %s", _port)
			err := endless.ListenAndServe(_port, r)
			if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				tilogs.L().Errorf("ListenAndServe port %s err %s", _port, err.Error())
				c <- err
			}
		}
	}, chanPublicErr)
	if !util.CheckGoroutineStartErr(chanPublicErr, "http.Serve public") {
		return
	}

	waitGroup.Wait()
}
