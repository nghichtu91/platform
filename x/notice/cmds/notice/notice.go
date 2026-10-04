package notice

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/iputil"
	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
	"github.com/nghichtu91/platform/share/x/common/serDisMgr"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/fvbock/endless"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/ginhelper"
	"github.com/nghichtu91/platform/share/planx/limit"
	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/util"
	cconfig "github.com/nghichtu91/platform/share/x/common/config"
	"github.com/nghichtu91/platform/share/x/notice/cmds"
	noticeCfg "github.com/nghichtu91/platform/share/x/notice/config"
	"github.com/nghichtu91/platform/share/x/notice/noticeimp"
	"github.com/nghichtu91/platform/share/x/notice/rounters"
)

func init() {
	cmds.Register(&cli.Command{
		Name:   "notice",
		Usage:  "开启notice功能",
		Action: Start,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config, c",
				Value: "config.toml",
				Usage: "Onland Configuration toml config, in {CWD}/conf/ or {AppPath}/conf",
			},
		},
	})
}

var waitGroup util.WaitGroupWrapper

func Start(c *cli.Context) {
	defer etcd.CloseEtcd() // etcd要最后关闭，否则一些反注册会失败
	defer waitGroup.Wait()
	defer signalhandler.OnClose()
	defer tilogs.PanicCatcher("notice allinone Start panic")

	zaplog.InitZapLog("log.toml", map[string]string{
		"service": "notice",
	}, "notice")
	timeutil.CheckTimeLocation()
	tilogs.L().Infof("Start with local %s", time.Local.String())

	// config
	if !initConfig(c.String("config")) {
		tilogs.L().Errorf("initConfig failed")
		return
	}
	tilogs.L().Infof("load config %s %+v", noticeCfg.Cfg.String(), limit.LimitCfg)

	// internalip
	internalIp := iputil.GetPrivateIP(noticeCfg.Cfg.InternalIp)
	if internalIp == "" {
		tilogs.L().Errorf("config InternalIp error, %s", noticeCfg.Cfg.InternalIp)
		return
	}
	// pprof
	pprofEtcdKey := etcd.GetPprofAddrKey(
		noticeCfg.Cfg.EtcdServer, strconv.Itoa(int(noticeCfg.Cfg.Gid)),
		etcd.Server_Notice, noticeCfg.Cfg.ServerId)
	stop := net.PProfStart(pprofEtcdKey, internalIp)
	defer stop()

	// for ratelimit
	if !limit.Init() {
		return
	}

	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })

	// metrics
	signalhandler.SignalKillFunc(func() { metrics.Stop() })
	if !metrics.Start("metrics.toml", noticeCfg.Proj, allmetrics.PrefixNoticeMetrics(noticeCfg.Cfg.Gid, noticeCfg.Cfg.ServerId)) {
		return
	}
	allmetrics.InitNoticeMetrics()
	metrics.InitMetricUtil(noticeCfg.Cfg.Gid, noticeCfg.Cfg.ServerId)
	tilogs.L().Infof("metrics start finish")

	// notice
	signalhandler.SignalKillFunc(noticeimp.CloseNotice)
	if !noticeimp.InitNotice(&waitGroup) {
		tilogs.L().Errorf("InitNotice failed")
		return
	}

	// debug time
	if err := timeutil.Init(noticeCfg.GidCfg.Gid, fmt.Sprint(0), noticeCfg.Cfg.EtcdServer, noticeCfg.GidCfg.CheatEnable, &waitGroup); err != nil {
		tilogs.L().Errorf("timeutil.Init err %s", err.Error())
		return
	}
	signalhandler.SignalKillFunc(func() { timeutil.Close() })

	// gin set mode
	if planx.IsRunProd(noticeCfg.RunMode) || planx.IsRunPrefTest(noticeCfg.RunMode) {
		gin.SetMode(gin.ReleaseMode)
	}

	// gin
	r, exitfun := ginhelper.MakeGinEngine("accesslog.xml")
	signalhandler.SignalKillFunc(func() { exitfun() })
	rounters.RegPublic(r)

	chanPublicErr := make(chan error, 1)
	waitGroup.WrapRetErr(func(c chan error) {
		if noticeCfg.Cfg.EnableHttpTLS {
			tilogs.L().Infof("notice serve http tls port: %s", noticeCfg.Cfg.HttpsAddress)
			err := endless.ListenAndServeTLS(
				noticeCfg.Cfg.HttpsAddress,
				noticeCfg.Cfg.HttpsCertFile,
				noticeCfg.Cfg.HttpsKeyFile,
				r,
			)
			if err != nil {
				tilogs.L().Errorf("ListenAndServeTLS HttpsPort %s err %s", noticeCfg.Cfg.HttpsAddress, err.Error())
				c <- err
			}
		} else {
			tilogs.L().Infof("auth serve http port: %s", noticeCfg.Cfg.HttpAddress)
			err := endless.ListenAndServe(noticeCfg.Cfg.HttpAddress, r)
			if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				tilogs.L().Errorf("ListenAndServe port %s err %s", noticeCfg.Cfg.HttpAddress, err.Error())
				c <- err
			}
		}
	}, chanPublicErr)
	if !util.CheckGoroutineStartErr(chanPublicErr, "http.Serve public") {
		return
	}

	// 服务器发现注册：内网信息注册，可以被其他服务发现
	opt := &serDisMgr.SerDisOpt{
		EtcdRoot:     noticeCfg.Cfg.EtcdServer,
		Gid:          noticeCfg.Cfg.Gid,
		SrvType:      etcd.Ser_Notice,
		SerID:        noticeCfg.Cfg.ServerId,
		InternalAddr: internalIp + noticeCfg.Cfg.HttpAddress,
		PublicAddr:   "",
	}
	signalhandler.SignalKillFunc(func() { serDisMgr.EtcdStop() })
	if err := serDisMgr.EtcdReg(opt); err != nil {
		tilogs.L().Errorf("EtcdReg err %s", err.Error())
		return
	}

	waitGroup.Wait()
}

func initConfig(cfgName string) bool {
	var cfg struct{ Config noticeCfg.Config }
	cfgApp := config.NewConfigToml(cfgName, &cfg)
	if cfgApp == nil {
		tilogs.L().Errorf("Config Read Error\n")
		return false
	}
	noticeCfg.Cfg = cfg.Config
	tilogs.SetTiLogger(tilogs.L().With(tilogs.TagServerId, noticeCfg.Cfg.ServerId).With(tilogs.TagGid, noticeCfg.Cfg.Gid))

	var limitCfg struct{ LimitConfig limit.LimitConfig }
	cfgApp = config.NewConfigToml(cfgName, &limitCfg)
	if cfgApp == nil {
		tilogs.L().Errorf("LimitConfig Read Error\n")
		return false
	}
	limit.LimitCfg = limitCfg.LimitConfig
	// etcd
	if err := etcd.InitEtcd(noticeCfg.Cfg.EtcdEndPoint); err != nil {
		tilogs.L().Errorf("etcd.InitEtcd err %s", err.Error())
		return false
	}

	runmode, err := cconfig.GetRunMode(noticeCfg.Cfg.EtcdDevOps,
		noticeCfg.Cfg.Gid)
	if err != nil {
		tilogs.L().Errorf("get runmode err %s", err.Error())
		return false
	}
	noticeCfg.RunMode = runmode
	proj, err := cconfig.GetProj(noticeCfg.Cfg.EtcdDevOps,
		noticeCfg.Cfg.Gid)
	if err != nil {
		tilogs.L().Errorf("get proj err %s", err.Error())
		return false
	}
	noticeCfg.Proj = proj
	return true
}
