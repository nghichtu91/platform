package chat

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/nghichtu91/platform/share/planx"

	"github.com/nghichtu91/platform/share/x/chat/logicx/logic/silence_sys"

	"github.com/nghichtu91/platform/share/x/chat/logicx/logic/translate"

	"github.com/nghichtu91/platform/share/planx/3rd_party_api/yidun"
	"github.com/nghichtu91/platform/share/planx/tracing"

	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/ginhelper"
	"github.com/nghichtu91/platform/share/planx/iputil"
	"github.com/nghichtu91/platform/share/planx/limit"
	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/nats_cli"
	net2 "github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/chatlog"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/chat/logicx/cmds"
	conf "github.com/nghichtu91/platform/share/x/chat/logicx/config"
	"github.com/nghichtu91/platform/share/x/chat/logicx/logic"
	"github.com/nghichtu91/platform/share/x/chat/logicx/logic/grpc"
	"github.com/nghichtu91/platform/share/x/chat/logicx/logic/hot_update"
	"github.com/nghichtu91/platform/share/x/chat/logicx/logic/http"

	//"github.com/nghichtu91/platform/share/x/chat/logicx/logic/illegalwords"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

func init() {
	cmds.Register(&cli.Command{
		Name:   "start",
		Usage:  "启动logic",
		Action: Start,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config, c",
				Value: "config.toml",
				Usage: "Onland Configuration toml config, in {CWD}/conf/ or {AppPath}/conf",
			},
			cli.StringFlag{
				Name:  "port, p",
				Value: ":10002",
				Usage: "port",
			},
			cli.StringFlag{
				Name:  "logiclog, logic",
				Value: "logiclog.toml",
				Usage: "Logic logiclog Configuration toml config, in {CWD}/conf/ or {AppPath}/conf",
			},
		},
	})
}

var waitGroup util.WaitGroupWrapper

// Start 启动logic
func Start(c *cli.Context) {
	defer etcd.CloseEtcd() // etcd要最后关闭，否则一些反注册会失败
	defer waitGroup.Wait()
	defer nats_cli.CloseNats()
	defer signalhandler.OnClose()
	defer tilogs.PanicCatcher("logic Start panic")

	zaplog.InitZapLog("log.toml", map[string]string{
		"service": "logic",
	}, "logic")

	cfgName := c.String("config")
	if !conf.LoadConfig(cfgName) {
		tilogs.L().Errorf("load loigc config failed")
		return
	}

	timeutil.CheckTimeLocation()
	tilogs.L().Infof("Start with local %s", time.Local.String())
	timeutil.InitTimeConst()

	// debug time
	if err := timeutil.Init(conf.Cfg.Gid, conf.Cfg.ServerId, conf.Cfg.EtcdServer, conf.EtcdConf.CheatEnable, &waitGroup); err != nil {
		tilogs.L().Errorf("timeutil.Init err %s", err.Error())
		return
	}
	signalhandler.SignalKillFunc(func() { timeutil.Close() })

	if !limit.Init() {
		return
	}

	// 加载平台log
	if !chatlog.LoadChatPlatformLog("logiclog.toml", conf.Cfg.Gid) {
		tilogs.L().Errorf("load chat platform config failed")
		return
	}

	// metrics
	signalhandler.SignalKillFunc(func() { metrics.Stop() })
	if !metrics.Start("metrics.toml", conf.EtcdConf.Proj, allmetrics.PrefixLogicMetrics(
		conf.Cfg.Gid, conf.Cfg.ServerId)) {
		return
	}
	allmetrics.InitLogicMetrics()
	metrics.InitMetricUtil(conf.Cfg.Gid, conf.Cfg.ServerId)
	tilogs.L().Infof("metrics start finish")

	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })
	reg_ser := etcd.NewRegServer(
		conf.Cfg.EtcdServer,
		[]string{fmt.Sprintf("%d", conf.Cfg.Gid)},
		etcd.Ser_ChatLogic, fmt.Sprintf("%s", conf.Cfg.ServerId))
	signalhandler.SignalKillFunc(func() { reg_ser.UnRegServerVer() })
	if err := reg_ser.RegServerVer(version.GetVersion()); err != nil {
		tilogs.L().Errorf("RegServerVer err %s", err.Error())
		return
	}

	/*
		gin.SetMode(gin.ReleaseMode)
		r, exitfun := ginhelper.MakeGinEngine("accesslog.xml")
		signalhandler.SignalKillFunc(func() { exitfun() })
		chanPublicErr := make(chan error, 1)
		waitGroup.WrapRetErr(func(c chan error) {
			tilogs.L().Infof("router serve http port: %s", conf.Cfg.HttpPort)
			err := endless.ListenAndServe(conf.Cfg.HttpPort, r)
			if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				tilogs.L().Errorf("ListenAndServe port %s err %s", conf.Cfg.HttpPort, err.Error())
				c <- err
			}
		}, chanPublicErr)
	*/

	// 内网端口
	internalIp := iputil.GetPrivateIP(conf.Cfg.InternalIp)
	if internalIp == "" {
		tilogs.L().Errorf("config InternalIp error, %s", conf.Cfg.InternalIp)
		return
	}
	// pprof
	pprofEtcdKey := etcd.GetPprofAddrKey(
		conf.Cfg.EtcdServer, strconv.Itoa(int(conf.Cfg.Gid)),
		etcd.Server_ChatLogic, conf.Cfg.ServerId)
	stop := net2.PProfStart(pprofEtcdKey, internalIp)
	defer stop()

	g_internal := ginhelper.MakeGinEngineNoLog()

	// //////////////////////// grpc listener /////////////////////
	grpcListener := net2.TryListen()
	if grpcListener == nil {
		tilogs.L().Errorf("TryListen grpc fail nil...")
		return
	}

	addr := grpcListener.Addr()
	port := fmt.Sprintf("%d", addr.(*net.TCPAddr).Port)
	grpcAddr := net.JoinHostPort(internalIp, port)
	net2.DebugForTryAddr(grpcAddr)

	// //////////////////////// http listener /////////////////////
	httpListener := net2.TryListen()
	if httpListener == nil {
		tilogs.L().Errorf("TryListen http fail nil...")
		return
	}

	addr = httpListener.Addr()
	port = fmt.Sprintf("%d", addr.(*net.TCPAddr).Port)
	httpAddr := net.JoinHostPort(internalIp, port)
	net2.DebugForTryAddr(httpAddr)

	/*
		login_url := fmt.Sprintf("http://%s%s", internal_addr,
			internalConfig.Router_Login_Root)
	*/

	// nats
	if err := nats_cli.InitNats(strconv.Itoa(int(conf.Cfg.Gid)), conf.Cfg.NatsUrl,
		planx.IsRunProd(conf.EtcdConf.RunMode) && !planx.IsCheatEnable(conf.EtcdConf.CheatEnable)); err != nil {
		tilogs.L().Errorf("InitNats err %s, natsUrl %s", err.Error(), conf.Cfg.NatsUrl)
		return
	}
	tilogs.L().Infof("InitNats success")

	logic := logic.New()
	// 服务器发现注册：内网信息注册，可以被其他服务发现
	signalhandler.SignalKillFunc(func() { cmds.LogicEtcdStop() })
	if err := cmds.LogicEtcdReg(conf.Cfg.ServerId, grpcAddr, httpAddr, logic); err != nil {
		tilogs.L().Errorf("LogicEtcdReg err %s", err.Error())
		return
	}

	signalhandler.SignalDownlinerHandler(logic)
	httpSrv := http.New(logic, httpListener, g_internal)
	if httpSrv == nil {
		tilogs.L().Errorf("gin init failed")
		return
	}

	grpcSer := grpc.New(&conf.Cfg, grpcListener, logic)
	info := grpcSer.GetServiceInfo()
	tilogs.L().Infof("grpc service info (%v)", info)

	// 易盾初始化
	if conf.Cfg.YiDun.Enable {
		yidun.InitYiDun(
			conf.Cfg.YiDun.BusinessId,
			conf.Cfg.YiDun.SecretId,
			conf.Cfg.YiDun.SecretKey,
			conf.Cfg.YiDun.Version,
			conf.Cfg.YiDun.Url,
		)

		// yidunSever.InitYiDun(
		// 	conf.Cfg.YiDun.BusinessId,
		// 	conf.Cfg.YiDun.SecretId,
		// 	conf.Cfg.YiDun.SecretKey,
		// 	conf.Cfg.YiDun.Version,
		// 	conf.Cfg.YiDun.Url,
		// )
	}

	// 翻译
	err := translate.InitCache(
		conf.Cfg.Translate.TranslateRedisAddr,
		conf.Cfg.Translate.TranslateRedisLanguage,
		conf.Cfg.Translate.TranslateRedisDb,
		conf.Cfg.Translate.TranslateRedisPwd,
		conf.Cfg.Translate.LRUMaxMemory,
	)
	if err != nil {
		tilogs.L().Errorf("translate init cache err %s", err.Error())
		return
	}

	// 加载脏字
	//if err := illegalwords.LoadIllegalWords(&waitGroup); err != nil {
	//	tilogs.L().Errorf("load sensitive words err %s", err.Error())
	//	return
	//}
	// 如果不关闭 导致ctrl-c结束不了进程
	//signalhandler.SignalKillFunc(func() { illegalwords.StopWatchIllegalWordsHash() })

	// logic db config load
	if err := hot_update.LoadDbConfig(&waitGroup); err != nil {
		tilogs.L().Errorf("load hot update err %s", err.Error())
		return
	}
	// 如果不关闭 导致ctrl-c结束不了进程
	signalhandler.SignalKillFunc(func() { hot_update.StopConfigWatch() })

	silenceModule := silence_sys.NewSilenceSysModule(uint32(conf.Cfg.Gid), 0)
	if err := silenceModule.Start(&waitGroup); err != nil {
		tilogs.L().Errorf("start silenceModule err %s", err.Error())
		return
	}
	signalhandler.SignalKillFunc(func() { silence_sys.StopModule() })

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

	tilogs.L().Infof("Logic Server started")

	signalhandler.SignalKillFunc(func() { nats_cli.CloseNats() })
	waitGroup.Wait()

	logic.Close()
	httpSrv.Close()
	grpcSer.GracefulStop()
	tilogs.L().Infof(".........Logic Server exit........")
}
