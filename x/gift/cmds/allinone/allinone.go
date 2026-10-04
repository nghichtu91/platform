package allinone

import (
	"time"

	net2 "github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/x/gift/db/oss"

	"github.com/nghichtu91/platform/share/x/gift/db/mysql"

	"github.com/nghichtu91/platform/share/x/gift/server"

	"github.com/nghichtu91/platform/share/x/gift/services"

	"github.com/nghichtu91/platform/share/planx/signalhandler"

	"github.com/nghichtu91/platform/share/x/gift/config"

	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/gift/cmds"
)

func init() {
	// 注册Gift服务器运行命令。
	cmds.Register(&cli.Command{
		Name:   "allinone", // 运行时使用 ./[编译名] [Name] 执行。
		Usage:  "开启Gift功能",
		Action: Start,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  config.ConfigFlag,  // 设置基础配置文件所在位置的参数调用和简写。
				Value: "config.toml",      // 默认采用的基础配置文件。
				Usage: "基础配置Toml文件的所在位置。", // 说明。
			},
			cli.StringFlag{
				Name:  config.LogicLogFlag, // 设置埋点配置文件所在位置的参数调用和简写。
				Value: "logiclog.toml",     // 默认采用的埋点配置文件。
				Usage: "埋点配置Toml文件的所在位置。",  // 说明。
			},
			cli.StringFlag{
				Name:  config.PortFlag, // 设置服务所在的端口的参数调用和简写。
				Value: ":8082",         // 默认提供服务的端口。
				Usage: "服务所在的端口",       // 说明。
			},
		},
	})
}

var waitGroup util.WaitGroupWrapper // 用于确保所有加入的线程执行完毕。

func Start(c *cli.Context) {
	defer waitGroup.Wait()
	defer signalhandler.OnClose()
	defer tilogs.PanicCatcher("Gift Start panic")

	// 日志模块初始化。
	zaplog.InitZapLog(config.LogFileName, map[string]string{
		config.LogService: config.Gift,
		config.LogSubCmd:  config.AllInOne,
	}, config.Gift)

	// 检查服务器所在时区。（必须设置环境变量TZ,否则直接Panic。）
	timeutil.CheckTimeLocation()
	tilogs.L().Infof("Server start with %s local time。", time.Local.String())

	// 获取服务的基础配置。
	if errConf := config.LoadGiftConfigAndInit(c); errConf != nil {
		tilogs.L().Errorf("Gift Server %v", errConf)
		return
	}
	// PProf
	net2.PProfStart("", "")

	// 初始化DB连接池。
	defer mysql.CloseDB()
	if errDB := mysql.InitDB(); errDB != nil {
		tilogs.L().Errorf("Gift Server %v", errDB)
		return
	}

	// 初始化OSS数据库。
	defer oss.CloseDB()
	giftCfg := config.Cfg.GiftCfg
	if errDB := oss.InitDB(giftCfg.OSSEndPoint, giftCfg.OSSDataBucket, giftCfg.OSSCloudRoot, giftCfg.OSSAccessKey, giftCfg.OSSSecretKey); errDB != nil {
		tilogs.L().Errorf("Gift Server %v", errDB)
		return
	}

	// 当收到终止信号时，会终止注册过Stop的模块。
	// 注意！当前此步骤处理有问题，若在起服过程中触发终止信号，会导致尚未注册的服务继续启动，导致终止失败。
	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })

	// 开启模块服务。
	if err := server.StartServer(&waitGroup); err != nil {
		tilogs.L().Errorf("%v", err)
		return
	}
	signalhandler.SignalKillFunc(func() { server.StopModuleMng() })

	// 注册GiftServer受理的API请求。
	if err := services.StartServices(&waitGroup); err != nil {
		tilogs.L().Errorf("%v", err)
		return
	}

	waitGroup.Wait()
}
