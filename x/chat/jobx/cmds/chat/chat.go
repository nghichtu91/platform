package chat

import (
	"strconv"
	"time"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/iputil"
	"github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/planx/tilogs/logiclog"
	"github.com/nghichtu91/platform/share/planx/tracing"

	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/limit"
	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/nats_cli"
	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/chat/jobx/cmds"
	conf "github.com/nghichtu91/platform/share/x/chat/jobx/config"
	"github.com/nghichtu91/platform/share/x/chat/jobx/job"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

func init() {
	cmds.Register(&cli.Command{
		Name:   "start",
		Usage:  "启动job",
		Action: Start,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "config, c",
				Value: "config.toml",
				Usage: "Onland Configuration toml config, in {CWD}/conf/ or {AppPath}/conf",
			},
			cli.StringFlag{
				Name:  "port, p",
				Value: ":10003",
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

// Start 启动job
func Start(c *cli.Context) {
	defer etcd.CloseEtcd() // etcd要最后关闭，否则一些反注册会失败
	defer waitGroup.Wait()
	defer nats_cli.CloseNats()
	defer signalhandler.OnClose()
	defer tilogs.PanicCatcher("job Start panic")

	zaplog.InitZapLog("log.toml", map[string]string{
		"service": "job",
	}, "job")

	tilogs.L().Infof("Start with local %s", time.Local.String())

	if !conf.LoadConfig() {
		tilogs.L().Errorf("load loigc config failed")
		return
	}

	if !limit.Init() {
		return
	}

	// internalip
	internalIp := iputil.GetPrivateIP(conf.Cfg.InternalIp)
	if internalIp == "" {
		tilogs.L().Errorf("config InternalIp error, %s", conf.Cfg.InternalIp)
		return
	}
	// pprof
	pprofEtcdKey := etcd.GetPprofAddrKey(
		conf.Cfg.EtcdServer, strconv.Itoa(int(conf.Cfg.Gid)),
		"jobx", conf.Cfg.ServerId)
	stop := net.PProfStart(pprofEtcdKey, internalIp)
	defer stop()

	// metrics
	signalhandler.SignalKillFunc(func() { metrics.Stop() })
	if !metrics.Start("metrics.toml", conf.EtcdConf.Proj, allmetrics.PrefixJobMetrics(
		conf.Cfg.Gid, conf.Cfg.ServerId)) {
		return
	}
	allmetrics.InitJobMetrics()
	metrics.InitMetricUtil(conf.Cfg.Gid, conf.Cfg.ServerId)
	tilogs.L().Infof("metrics start finish")

	// nats
	if err := nats_cli.InitNats(strconv.Itoa(int(conf.Cfg.Gid)), conf.Cfg.NatsUrl,
		planx.IsRunProd(conf.EtcdConf.RunMode) && !planx.IsCheatEnable(conf.EtcdConf.CheatEnable)); err != nil {
		tilogs.L().Errorf("InitNats err %s, natsUrl %s", err.Error(), conf.Cfg.NatsUrl)
		return
	}
	tilogs.L().Infof("InitNats success")

	j := job.New()
	j.InitConsume()

	// 发现服务
	cmds.CometDiscovery(conf.Cfg.EtcdServer,
		conf.Cfg.Gid, j)
	signalhandler.SignalKillFunc(func() { cmds.CometDiscoveryStop() })
	err := cmds.CometDiscoveryStart()
	if err != nil {
		tilogs.L().Errorf("ChatDiscovery err %s", err.Error())
		return
	}

	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })

	// logiclog
	if !logiclog.LoadGameLogic(c.String("logiclog")) {
		tilogs.L().Errorf("LoadGameLogic failed")
		return
	}

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

	tilogs.L().Infof("job Server started")

	signalhandler.SignalKillFunc(func() { nats_cli.CloseNats() })
	waitGroup.Wait()

	tilogs.L().Infof(".........job Server exit........")
}
