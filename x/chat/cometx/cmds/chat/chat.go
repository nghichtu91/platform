package chat

import (
	"fmt"
	"net"
	"runtime"
	"strconv"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs/logiclog"
	"github.com/nghichtu91/platform/share/planx/tracing"

	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/iputil"
	"github.com/nghichtu91/platform/share/planx/limit"
	"github.com/nghichtu91/platform/share/planx/metrics"
	net2 "github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/chat/cometx/cmds"
	"github.com/nghichtu91/platform/share/x/chat/cometx/comet"
	"github.com/nghichtu91/platform/share/x/chat/cometx/comet/grpc"
	cometConfig "github.com/nghichtu91/platform/share/x/chat/cometx/config"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

func init() {
	cmds.Register(&cli.Command{
		Name:   "start",
		Usage:  "启动comet",
		Action: Start,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config, c",
				Value: "config.toml",
				Usage: "Onland Configuration toml config, in {CWD}/conf/ or {AppPath}/conf",
			},
			cli.StringFlag{
				Name:  "port, p",
				Value: ":10001",
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

// Start 启动comet
func Start(c *cli.Context) {
	defer etcd.CloseEtcd() // etcd要最后关闭，否则一些反注册会失败
	defer waitGroup.Wait()
	defer signalhandler.OnClose()
	defer tilogs.PanicCatcher("comet Start panic")

	zaplog.InitZapLog("log.toml", map[string]string{
		"service": "comet",
	}, "comet")

	tilogs.L().Infof("Start with local %s", time.Local.String())

	cfgName := c.String("config")
	if !cometConfig.LoadConfig(cfgName) {
		tilogs.L().Errorf("load loigc config failed")
		return
	}

	if !limit.Init() {
		return
	}

	// metrics
	signalhandler.SignalKillFunc(func() { metrics.Stop() })
	if !metrics.Start("metrics.toml", cometConfig.EtcdConf.Proj, allmetrics.PrefixCometxMetrics(
		cometConfig.Cfg.Gid, cometConfig.Cfg.ServerId)) {
		return
	}
	allmetrics.InitCometxMetrics()
	metrics.InitMetricUtil(cometConfig.Cfg.Gid, cometConfig.Cfg.ServerId)
	tilogs.L().Infof("metrics start finish")

	reg_ser := etcd.NewRegServer(
		cometConfig.Cfg.EtcdServer,
		[]string{fmt.Sprintf("%d", cometConfig.Cfg.Gid)},
		etcd.Ser_ChatComet, fmt.Sprintf("%s", cometConfig.Cfg.ServerId))
	signalhandler.SignalKillFunc(func() { reg_ser.UnRegServerVer() })
	if err := reg_ser.RegServerVer(version.GetVersion()); err != nil {
		tilogs.L().Errorf("RegServerVer err %s", err.Error())
		return
	}

	internalIp := iputil.GetPrivateIP(cometConfig.Cfg.InternalIp)
	if internalIp == "" {
		tilogs.L().Errorf("config InternalIp error, %s", cometConfig.Cfg.InternalIp)
		return
	}
	// pprof
	pprofEtcdKey := etcd.GetPprofAddrKey(
		cometConfig.Cfg.EtcdServer, strconv.Itoa(int(cometConfig.Cfg.Gid)),
		etcd.Server_ChatComet, cometConfig.Cfg.ServerId)
	stop := net2.PProfStart(pprofEtcdKey, internalIp)
	defer stop()

	// ///////////////// grpc internal ip ////////////////////
	grpcListener := net2.TryListen()
	if grpcListener == nil {
		tilogs.L().Errorf("TryListen grpc fail nil...")
		return
	}

	addr := grpcListener.Addr()
	port := fmt.Sprintf("%d", addr.(*net.TCPAddr).Port)
	grpc_internal_addr := net.JoinHostPort(internalIp, port)
	net2.DebugForTryAddr(grpc_internal_addr)

	pip := cometConfig.Cfg.PublicIP
	cometConfig.Cfg.PublicIP = iputil.GetPublicIP(pip, cometConfig.Cfg.Listen)
	tilogs.L().Infof("[ChatServer] with public ip %s binded.", cometConfig.Cfg.PublicIP)

	// comet 服务启动
	cometSer := comet.NewServer()
	runtime.GOMAXPROCS(runtime.NumCPU())
	// accepts的数据按照cpu数来
	cometSer.InitTcp(runtime.NumCPU(), cometConfig.Cfg.Listen)

	// comet gprc服务端 --> client is job
	grpcSer := grpc.New(&cometConfig.Cfg, cometSer, grpcListener)
	info := grpcSer.GetServiceInfo()
	tilogs.L().Infof("grpc service info (%v)", info)

	// 注册服务
	signalhandler.SignalKillFunc(func() { cmds.CometEtcdStop() })
	publicIP := cometConfig.Cfg.PublicIP
	// 如果cometx配置了elb地址，用elb地址替换公网IP
	if cometConfig.Cfg.ElbCometxAddr != "" {
		publicIP = cometConfig.Cfg.ElbCometxAddr + cometConfig.Cfg.Listen
	}
	if err := cmds.CometEtcdReg(cometConfig.Cfg.ServerId, internalIp+cometConfig.Cfg.Listen, publicIP, grpc_internal_addr, cometSer); err != nil {
		tilogs.L().Errorf("CometEtcdReg err %s", err.Error())
		return
	}

	// 发现服务
	cmds.LogicDiscovery(cometConfig.Cfg.EtcdServer,
		cometConfig.Cfg.Gid, cometSer)
	signalhandler.SignalKillFunc(func() { cmds.LogicDiscoveryStop() })
	err := cmds.LogicDiscoveryStart()
	if err != nil {
		tilogs.L().Errorf("Discovery logic err %s", err.Error())
		return
	}

	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })

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

	// logiclog
	if !logiclog.LoadGameLogic(c.String("logiclog")) {
		tilogs.L().Errorf("LoadGameLogic failed")
		return
	}

	tilogs.L().Infof("Comet Server started")
	waitGroup.Wait()

	cometSer.Close()
	grpcSer.GracefulStop()
	tilogs.L().Infof(".........Comet Server exit........")
}
