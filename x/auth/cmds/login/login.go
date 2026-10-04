package login

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/nghichtu91/platform/share/planx/iputil"
	net2 "github.com/nghichtu91/platform/share/planx/net"

	"github.com/nghichtu91/platform/share/planx/ginhelper"

	"github.com/nghichtu91/platform/share/planx"

	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/version"

	"github.com/nghichtu91/platform/share/planx/etcd"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/auth/cmds"
	internalConfig "github.com/nghichtu91/platform/share/x/auth/config"
	loginConfig "github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/auth/models"
	"github.com/nghichtu91/platform/share/x/auth/routers"
)

func init() {
	cmds.Register(&cli.Command{
		Name:   "login",
		Usage:  "开启login功能",
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

func Start(c *cli.Context) {
	defer etcd.CloseEtcd() //etcd要最后关闭，否则一些反注册会失败
	defer signalhandler.OnClose()
	defer tilogs.PanicCatcher("login Start panic")

	zaplog.InitZapLog("log.toml", map[string]string{
		"service": "auth",
		"subcmd":  "login",
	}, "auth")
	tilogs.L().Infof("Start with local %s", time.Local.String())

	// config
	if !cmds.InitAuthConfig(c.String("config")) {
		tilogs.L().Errorf("InitAuthConfig failed")
		return
	}

	// 内网监听
	internalIp := iputil.GetPrivateIP(loginConfig.Cfg.CommonCfg.InternalIp)
	if internalIp == "" {
		tilogs.L().Errorf("config InternalIp error, %s", loginConfig.Cfg.CommonCfg.InternalIp)
		return
	}
	//pprof
	pprofEtcdKey := etcd.GetPprofAddrKey(
		loginConfig.Cfg.CommonCfg.EtcdServer, strconv.Itoa(int(loginConfig.Cfg.CommonCfg.Gid)),
		etcd.Server_Login, loginConfig.Cfg.CommonCfg.ServerId)
	stop := net2.PProfStart(pprofEtcdKey, internalIp)
	defer stop()

	var waitGroup util.WaitGroupWrapper
	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })

	if planx.IsRunProd(loginConfig.GidCfg.RunMode) || planx.IsRunPrefTest(loginConfig.GidCfg.RunMode) {
		gin.SetMode(gin.ReleaseMode)
	}

	tilogs.L().Debugf("Start login Server %v", loginConfig.Cfg.CommonCfg)

	if err := models.InitDb(&loginConfig.Cfg.CommonCfg, loginConfig.GidCfg); err != nil {
		tilogs.L().Errorf("init db  Error %v", err)
		return
	}
	models.InitLoginRedis(&loginConfig.Cfg.CommonCfg)

	// gonggao
	signalhandler.SignalKillFunc(func() { models.CloseWatchGongGao() })
	models.StartWatchGongGao(&waitGroup)

	// whiteListPwd
	signalhandler.SignalKillFunc(func() { models.CloseWatchWhiteListPwd() })
	models.StartWatchWhiteListPwd(&waitGroup)

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
	login_url := fmt.Sprintf("http://%s%s", internal_addr,
		internalConfig.Router_Login_Root)
	// 服务器发现注册：内网信息注册，可以被其他服务发现
	signalhandler.SignalKillFunc(func() { cmds.AuthEtcdStop() })
	if err := cmds.AuthEtcdReg(loginConfig.Cfg.CommonCfg.ServerId, login_url, internal_addr, ""); err != nil {
		tilogs.L().Errorf("AuthEtcdReg err %s", err.Error())
		return
	}

	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&loginConfig.LoginUrl)),
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

	reg_ser := etcd.NewRegServer(
		loginConfig.Cfg.CommonCfg.EtcdServer,
		[]string{fmt.Sprintf("%d", loginConfig.Cfg.CommonCfg.Gid)},
		etcd.Ser_Login, internal_addr)
	signalhandler.SignalKillFunc(func() { reg_ser.UnRegServerVer() })
	if err := reg_ser.RegServerVer(version.GetVersion()); err != nil {
		tilogs.L().Errorf("RegServerVer err %s", err.Error())
		return
	}

	waitGroup.Wait()
}
