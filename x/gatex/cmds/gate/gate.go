package gate

import (
	"fmt"
	"net"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/iputil"
	net2 "github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/planx/tilogs/grpclog"
	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/x/common/allmetrics"

	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"

	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/version"

	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/authdiscovery"
	"github.com/nghichtu91/platform/share/x/gatex/cmds"
	gateconfig "github.com/nghichtu91/platform/share/x/gatex/config"
	"github.com/nghichtu91/platform/share/x/gatex/gate"
)

func init() {
	cmds.Register(&cli.Command{
		Name:   "gate",
		Usage:  "启动gate模式",
		Action: Start,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config, c",
				Value: "gate.toml",
				Usage: "Gate Configuration toml config, in {CWD}/conf/ or {AppPath}/conf",
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
	defer tilogs.PanicCatcher("gatex Start panic")

	zaplog.InitZapLog("log.toml", map[string]string{
		"service": "gatex",
	}, "gatex")
	grpclog.SetGRPCLogger()
	timeutil.CheckTimeLocation()
	tilogs.L().Infof("Start with local %s", time.Local.String())

	cfgName := c.String("config")

	// handle kill signal
	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })

	// load
	if !gateconfig.LoadConfig(cfgName) {
		tilogs.L().Errorf("gate.LoadConfig fail !!")
		return
	}
	tilogs.L().Infof("gate config %+v", gateconfig.Cfg)

	gateCfg := gateconfig.Cfg.GateConfig

	// 内网ip
	internalIp := iputil.GetPrivateIP(gateCfg.InternalIp)
	if internalIp == "" {
		tilogs.L().Errorf("config InternalIp error, %s", gateCfg.InternalIp)
		return
	}
	// pprof
	pprofEtcdKey := etcd.GetPprofAddrKey(gateconfig.Cfg.GateConfig.EtcdServer, strconv.Itoa(int(gateconfig.Cfg.GateConfig.Gid)), etcd.Server_Gate, gateconfig.Cfg.GateConfig.ServerId)
	stop := net2.PProfStart(pprofEtcdKey, internalIp)
	defer stop()

	// debug time
	if err := timeutil.Init(gateCfg.Gid, fmt.Sprint(0), gateCfg.EtcdServer, gateconfig.CheatEnable, &waitGroup); err != nil {
		tilogs.L().Errorf("timeutil.Init err %s", err.Error())
		return
	}
	signalhandler.SignalKillFunc(func() { timeutil.Close() })

	// auth discovery
	authdiscovery.AuthDiscovery(
		gateCfg.EtcdServer,
		gateCfg.Gid)
	signalhandler.SignalKillFunc(func() { authdiscovery.AuthDiscoveryStop() })
	err := authdiscovery.AuthDiscoveryStart()
	if err != nil {
		tilogs.L().Errorf("authdiscovery.AuthDiscovery err %s", err.Error())
		return
	}

	lis_Internal := net2.TryListen()
	if lis_Internal == nil {
		tilogs.L().Errorf("TryListen fail nil...")
		return
	}

	// metrics
	signalhandler.SignalKillFunc(func() { metrics.Stop() })
	if !metrics.Start("metrics.toml", gateconfig.Proj, allmetrics.PrefixGatexMetrics(gateCfg.Gid, gateCfg.ServerId)) {
		return
	}
	allmetrics.InitGatexMetrics()
	allmetrics.InitServerCheckMetrics()
	metrics.InitMetricUtil(gateCfg.Gid, gateCfg.ServerId)
	tilogs.L().Infof("metrics start finish")

	// internal ip
	addr := lis_Internal.Addr()
	port := fmt.Sprintf("%d", addr.(*net.TCPAddr).Port)
	internalIpPort := net.JoinHostPort(internalIp, port)
	net2.DebugForTryAddr(internalIpPort)

	// public ip
	pip := gateconfig.Cfg.GateConfig.PublicIP
	if gateconfig.Cfg.GateConfig.CloudGAAddress != "" {
		pip = net.JoinHostPort(gateconfig.Cfg.GateConfig.CloudGAAddress, strings.TrimLeft(gateconfig.Cfg.GateConfig.Listen, ":"))
		tilogs.L().Infof("used CloudGAAddress %s", pip)
	}
	gateconfig.Cfg.GateConfig.PublicIP = gate.GetPublicIP(pip, gateconfig.Cfg.GateConfig.Listen)
	tilogs.L().Infof("[GateServer] with public ip %s binded. public host %s",
		gateconfig.Cfg.GateConfig.PublicIP,
		gateconfig.Cfg.GateConfig.GetPublicHost(),
	)

	reg_ser := etcd.NewRegServer(
		gateCfg.EtcdServer,
		[]string{fmt.Sprintf("%d", gateCfg.Gid)},
		etcd.Ser_Gate, gateconfig.Cfg.GateConfig.ServerId)
	signalhandler.SignalKillFunc(func() { reg_ser.UnRegServerVer() })
	if err := reg_ser.RegServerVer(version.GetVersion()); err != nil {
		tilogs.L().Errorf("RegServerVer err %s", err.Error())
		return
	}

	GateSer := gate.NewGateServer(lis_Internal)
	signalhandler.SignalKillHandler(GateSer)

	if !GateSer.NewChatRedis() {
		tilogs.L().Errorf("gate new chat redis failed")
	}

	gameMgr := gate.NewGameMgr(GateSer)
	signalhandler.SignalKillHandler(gameMgr)
	if err := gameMgr.StartGameMgr(internalIpPort); err != nil {
		tilogs.L().Errorf("gate.StartGameMgr err %s", err.Error())
		return
	}
	gameserver := gate.NewProtoGameServerManager(gameMgr)

	if err := GateSer.Listen(); err != nil {
		return
	}
	waitGroup.Wrap(func() { GateSer.Start(gameserver, internalIpPort) })

	// 注册服务发现
	signalhandler.SignalKillFunc(cmds.StopServiceReg)
	if err := cmds.StartServiceReg(gateconfig.Cfg.GateConfig.ServerId,
		&gateconfig.Cfg.GateConfig, internalIpPort); err != nil {
		tilogs.L().Errorf("gate.StartGameMgr StartServiceReg err %s", err.Error())
		return
	}

	signalhandler.SignalDownlinerFunc(cmds.DownLiner)

	waitGroup.Wait()
}
