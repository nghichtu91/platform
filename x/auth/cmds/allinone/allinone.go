package allinone

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/nghichtu91/platform/share/planx/virtual_gid"
	"github.com/nghichtu91/platform/share/x/auth/config_data"

	"github.com/nghichtu91/platform/share/x/auth/controllers"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/mosn/holmes"
	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx/tracing"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/ginhelper"
	"github.com/nghichtu91/platform/share/planx/iputil"
	"github.com/nghichtu91/platform/share/planx/limit"
	"github.com/nghichtu91/platform/share/planx/metrics"
	net2 "github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/logiclog"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/auth/cmds"
	authConfig "github.com/nghichtu91/platform/share/x/auth/config"
	internalConfig "github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/auth/hero_log/common_log"
	"github.com/nghichtu91/platform/share/x/auth/models"
	"github.com/nghichtu91/platform/share/x/auth/routers"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

func init() {
	cmds.Register(&cli.Command{
		Name:   "allinone",
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

var waitGroup util.WaitGroupWrapper

func Start(c *cli.Context) {
	defer etcd.CloseEtcd() // etcd要最后关闭，否则一些反注册会失败
	defer waitGroup.Wait()
	defer signalhandler.OnClose()
	defer timeutil.Close()
	defer tilogs.PanicCatcher("auth allinone Start panic")

	zaplog.InitZapLog("log.toml", map[string]string{
		"service": "auth",
	}, "auth")
	timeutil.CheckTimeLocation()

	tilogs.L().Infof("Start with local %s IsAsiaShanghaiTZ() %v", time.Local.String(), timeutil.IsAsiaShanghaiTZ())

	// config
	if !cmds.InitAuthConfig(c.String("config")) {
		tilogs.L().Errorf("InitAuthConfig failed")
		return
	}
	limit.LimitCfg = authConfig.Cfg.LimitCfg

	// logiclog
	if !logiclog.LoadGameLogic(c.String("logiclog")) {
		tilogs.L().Errorf("LoadGameLogic failed")
		return
	}

	if !limit.Init() {
		return
	}

	if planx.IsRunProd(authConfig.GidCfg.RunMode) || planx.IsRunPrefTest(authConfig.GidCfg.RunMode) {
		gin.SetMode(gin.ReleaseMode)
	}
	// 内网端口
	internalIp := iputil.GetPrivateIP(authConfig.Cfg.CommonCfg.InternalIp)
	if internalIp == "" {
		tilogs.L().Errorf("config InternalIp error, %s", authConfig.Cfg.CommonCfg.InternalIp)
		return
	}
	// pprof
	pprofEtcdKey := etcd.GetPprofAddrKey(
		authConfig.Cfg.CommonCfg.EtcdServer, strconv.Itoa(int(authConfig.Cfg.CommonCfg.Gid)),
		etcd.Server_Auth, authConfig.Cfg.CommonCfg.ServerId)
	stop := net2.PProfStart(pprofEtcdKey, internalIp)
	defer stop()

	// 加载运营返利配置文件。
	if err := config_data.LoadConfigData(c.String("data")); err != nil {
		tilogs.L().Errorf(err.Error())
		return
	}

	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })

	// holmes
	if planx.IsRunPrefTest(authConfig.GidCfg.RunMode) {
		h, err := holmes.New(
			holmes.WithCollectInterval("5s"),
			holmes.WithCoolDown("1m"),
			holmes.WithDumpPath("./", fmt.Sprintf("auth%d_holmes.log", authConfig.GidCfg.Gid)),
			holmes.WithTextDump(),
			holmes.WithLoggerLevel(holmes.LogLevelInfo),

			// holmes.WithCPUDump(10, 25, 50),
			// holmes.WithMemDump(30, 25, 80),
			holmes.WithGoroutineDump(500, 25, 20000),
		)
		if err != nil {
			tilogs.L().Errorf("new holmes failed, %v", err)
			return
		}

		h.EnableMemDump().
			EnableCPUDump().
			EnableGoroutineDump()

		h.Start()
		defer h.Stop()
	}

	// metrics
	signalhandler.SignalKillFunc(func() { metrics.Stop() })
	if !metrics.Start("metrics.toml", authConfig.GidCfg.Proj, allmetrics.PrefixAuthMetrics(
		authConfig.GidCfg.Gid, authConfig.Cfg.CommonCfg.ServerId)) {
		return
	}
	tilogs.L().Infof("metrics start finish")
	allmetrics.InitAuthMetrics()
	metrics.InitMetricUtil(authConfig.GidCfg.Gid, fmt.Sprint(authConfig.Cfg.CommonCfg.ServerId))

	if err := models.InitDb(&authConfig.Cfg.CommonCfg, authConfig.GidCfg); err != nil {
		tilogs.L().Errorf("init db %s Error: %v", authConfig.GidCfg.DBDriver, err)
		return
	}
	models.InitLoginRedis(&authConfig.Cfg.CommonCfg)
	err := models.InitNumberIDInRedis()
	if err != nil {
		tilogs.L().Errorf("init numberID in redis err by %v", err.Error())
		return
	}

	sSerId := authConfig.Cfg.CommonCfg.ServerId
	reg := regexp.MustCompile(`[0-9]`)
	nonn := reg.Find([]byte(sSerId))
	iSerId, err := strconv.Atoi(sSerId[strings.Index(sSerId, string(nonn)):])
	if err != nil {
		tilogs.L().Errorf("serverid no number %s, err %v", sSerId, err)
		return
	}
	if !controllers.InitReqIdGenerator(uint(iSerId)) {
		tilogs.L().Errorf("InitReqIdGenerator fail, iSerId %d", iSerId)
		return
	}

	signalhandler.SignalKillFunc(func() { models.StopShardInfo() })

	// 虚拟大区初始化，需要在读取服务器列表之前
	err = virtual_gid.Init(authConfig.Cfg.CommonCfg.EtcdRoot, authConfig.Cfg.CommonCfg.Gid)
	if err != nil {
		tilogs.L().Errorf("virtual_gid.Init err: %s", err.Error())
		return
	}

	err = models.InitShardInfo()
	if err != nil {
		tilogs.L().Errorf("models.InitShardInfo err: %s", err.Error())
		return
	}

	g_internal := ginhelper.MakeGinEngineNoLog()
	routers.RegUserMgr(g_internal)
	lis_Internal := net2.TryListen()
	if lis_Internal == nil {
		tilogs.L().Errorf("TryListen fail nil...")
		return
	}
	addr := lis_Internal.Addr()
	port := fmt.Sprintf("%d", addr.(*net.TCPAddr).Port)
	internal_addr := net.JoinHostPort(internalIp, port)
	net2.DebugForTryAddr(internal_addr)
	login_url := fmt.Sprintf("http://%s%s", internal_addr,
		internalConfig.Router_Login_Root)

	// 外网
	_port := authConfig.Cfg.CommonCfg.Httpport
	if c.String("port") != "" {
		_port = c.String("port")
	}
	if authConfig.Cfg.CommonCfg.EnableHttpTLS {
		_port = authConfig.Cfg.CommonCfg.HttpsPort
	}
	pip := authConfig.Cfg.CommonCfg.PublicIP
	publicIp := iputil.GetPublicIP(pip, _port)
	// 如果auth配置了elb地址，用elb地址替换公网IP
	if authConfig.Cfg.CommonCfg.ElbAuthAddr != "" {
		publicIp = authConfig.Cfg.CommonCfg.ElbAuthAddr + _port
	}
	tilogs.L().Infof("%s with public ip %s bind, private ip %s", authConfig.Cfg.CommonCfg.ServerId, publicIp, internal_addr)
	// TODO to etcd  etcd_server/gid/auth/publicIp

	// 服务器发现注册：内网信息注册，可以被其他服务发现
	signalhandler.SignalKillFunc(func() { cmds.AuthEtcdStop() })
	if err := cmds.AuthEtcdReg(authConfig.Cfg.CommonCfg.ServerId, login_url, internal_addr, publicIp); err != nil {
		tilogs.L().Errorf("AuthEtcdReg err %s", err.Error())
		return
	}

	authUrl := fmt.Sprintf("%s/%d/%s", authConfig.Cfg.CommonCfg.EtcdServer, authConfig.Cfg.CommonCfg.Gid, consts.AUTH_URL_SUFFIX)
	if _, err := etcd.GetEtcd().Put(context.Background(), authUrl, publicIp); err != nil {
		return
	}

	reg_ser := etcd.NewRegServer(
		authConfig.Cfg.CommonCfg.EtcdServer,
		[]string{strconv.Itoa(int(authConfig.Cfg.CommonCfg.Gid))},
		etcd.Ser_Auth, authConfig.Cfg.CommonCfg.ServerId)
	// register ver
	signalhandler.SignalKillFunc(func() { reg_ser.UnRegServerVer() })
	if err := reg_ser.RegServerVer(version.GetVersion()); err != nil {
		tilogs.L().Errorf("RegServerVer err %s", err.Error())
		return
	}

	// debug time
	if err := timeutil.Init(authConfig.GidCfg.Gid, fmt.Sprint(0), authConfig.Cfg.CommonCfg.EtcdServer, authConfig.GidCfg.CheatEnable, &waitGroup); err != nil {
		tilogs.L().Errorf("timeutil.Init err %s", err.Error())
		return
	}
	signalhandler.SignalKillFunc(func() { timeutil.Close() })

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

	// 内网监听
	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&authConfig.LoginUrl)),
		(unsafe.Pointer)(&login_url))
	chanInternalErr := make(chan error, 1)
	go func(c chan error) {
		if err := http.Serve(lis_Internal, g_internal); err != nil {
			tilogs.L().Errorf("http.Serve %s err %s", lis_Internal.Addr().String(), err.Error())
			c <- err
		}
	}(chanInternalErr)
	if !util.CheckGoroutineStartErr(chanInternalErr, "http.Serve %s", lis_Internal.Addr().String()) {
		return
	}

	r, exitfun := ginhelper.MakeGinEngine("accesslog.xml")
	r.Use(ginhelper.MetricHandler())
	// 外网监听
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
	// BDC上报日志
	signalhandler.SignalKillFunc(func() { common_log.Stop() })
	common_log.Start(authConfig.Cfg.CommonCfg.ServerId, planx.IsRunProd(authConfig.GidCfg.RunMode))

	// tracing
	err = tracing.Start("tracing.toml")
	if err != nil {
		tilogs.L().Errorf("tracing start err by %v", err.Error())
		return
	}
	closer, err := tracing.InitGlobalTracer(tilogs.L())
	if err != nil {
		tilogs.L().Errorf("tracing start err by %v", err.Error())
		return
	}
	signalhandler.SignalKillFunc(func() {
		if err := closer.Close(); err != nil {
			tilogs.L().Errorf("tracing close err by %v", err.Error())
		}
	})

	waitGroup.Wait()
}
