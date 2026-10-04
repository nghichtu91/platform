package allinone

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli"

	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/nats_cli"
	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/version"
	"github.com/nghichtu91/platform/share/x/chat/web/cmds"
	gmConfig "github.com/nghichtu91/platform/share/x/chat/web/config"
	"github.com/nghichtu91/platform/share/x/chat/web/db"
	"github.com/nghichtu91/platform/share/x/chat/web/logic"
	"github.com/nghichtu91/platform/share/x/chat/web/model/oss"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"
	"github.com/nghichtu91/platform/share/x/chat/web/sql_string"
	"github.com/nghichtu91/platform/share/x/chat/web/util"
	commonConfig "github.com/nghichtu91/platform/share/x/common/config"
)

func init() {
	cmds.Register(&cli.Command{
		Name:   "allinone",
		Usage:  "开启所有功能",
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

var waitGroup = &util.WaitGroup

func Start(c *cli.Context) {
	defer etcd.CloseEtcd()     //etcd要最后关闭，否则一些反注册会失败  无用
	defer waitGroup.Wait()     // 等待协程完了后结束主线程
	defer nats_cli.CloseNats() // 消息中间件 无用
	defer oss.StopCloudDB()
	defer signalhandler.OnClose()
	defer tilogs.PanicCatcher("gmchatx Start panic") // 日志

	zaplog.InitZapLog("log.toml", map[string]string{
		"service": "chatweb",
		"subcmd":  "allinone",
	}, "chatweb")

	// 读取本地配置
	cfgName := c.String("config")
	var common_cfg struct{ CommonCfg gmConfig.LocalConfig }
	cfgApp := config.NewConfigToml(cfgName, &common_cfg) // 读取conf中的toml配置 common_cfg
	gmConfig.Cfg.LocalConfig = common_cfg.CommonCfg
	if cfgApp == nil {
		tilogs.L().Errorf("CommonConfig Read Error\n")
		return
	}

	//加载etcd配置
	if err := etcd.InitEtcd(gmConfig.Cfg.EtcdEndpoint); err != nil {
		tilogs.L().Errorf("etcd.InitEtcd err %s", err.Error())
		return
	}
	gidCfg := commonConfig.InitLoadGidConfig(gmConfig.Cfg.DevopsEtcdRoot, gmConfig.Cfg.LocalConfig.Gid)
	if gidCfg == nil {
		tilogs.L().Errorf("LoadGidConfig err")
		return
	}
	gmConfig.Cfg.GidConfig = *gidCfg
	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })
	reg_ser := etcd.NewRegServer(
		gmConfig.Cfg.ServerEtcdRoot,
		[]string{fmt.Sprintf("%d", gmConfig.Cfg.LocalConfig.Gid)},
		etcd.Ser_GmChat, "")
	signalhandler.SignalKillFunc(func() { reg_ser.UnRegServerVer() })
	if err := reg_ser.RegServerVer(version.GetVersion()); err != nil {
		tilogs.L().Errorf("RegServerVer err %s", err.Error())
		return
	}
	tilogs.L().Infof("load local config %v", gmConfig.Cfg)

	// metrics
	if !metrics.Start("metrics.toml", gmConfig.Cfg.Proj, allmetrics.PrefixGMChatMetrics(gmConfig.Cfg.LocalConfig.Gid)) {
		return
	}

	//建立database
	db.InitDB()
	sqlStr := sql_string.SqlString
	tilogs.L().Infof("create table sql %v", strings.ReplaceAll(sqlStr, "\n", " "))
	_, err := db.GetDB().Exec(strings.ReplaceAll(sqlStr, "\n", " ")) // 初始化数据表
	if err != nil {
		tilogs.L().Errorf("create table err %v", err)
		return
	}
	createAdminAccount()

	// 启动gin服务器
	r := gin.Default()
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	store := memstore.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("gm", store))
	logic.RegisterCommand(r)
	r.Static("/", "./webapp/build/dist")
	go r.Run(gmConfig.Cfg.LocalConfig.Port)

	// 启动oss
	if !oss.InitCloudDB() {
		tilogs.L().Errorf("battleconfig.InitCloudDb fail !!")
		return
	}
	signalhandler.SignalKillFunc(func() {
		oss.StopCloudDB()
	})

	waitGroup.Wait()

}

//func Start(c *cli.Context) {
//	defer etcd.CloseEtcd() //etcd要最后关闭，否则一些反注册会失败  无用
//	defer waitGroup.Wait() // 等待协程完了后结束主线程
//	defer oss.StopCloudDB()
//	defer signalhandler.SignalClose()                 // 信号
//	defer tilogs.PanicCatcher("gm_tools Start panic") // 日志
//
//	zaplog.InitZapLog("log.toml", map[string]string{
//		"service": "gm",
//		"subcmd":  "allinone",
//	})
//
//	// 读取本地配置
//	cfgName := c.String("config")
//	var common_cfg struct{ config.LocalConfig }
//	cfgApp := tomlUtil.NewConfigToml(cfgName, &common_cfg) // 读取conf中的toml配置 common_cfg
//	config.LocalCfg = common_cfg.LocalConfig
//	if cfgApp == nil {
//		tilogs.L().Errorf("CommonConfig Read Error\n")
//		return
//	}
//
//	//加载etcd配置
//	tilogs.L().Infof("呵呵呵 %v", config.LocalCfg)
//	if err := etcd.InitEtcd(config.LocalCfg.EtcdEndpoint); err != nil {
//		tilogs.L().Errorf("etcd.InitEtcd err %s", err.Error())
//		return
//	}
//	gidCfg := commonConfig.LoadGidConfig(config.LocalCfg.DevopsEtcdRoot, config.LocalCfg.Gid)
//	config.GidCfg = *gidCfg
//
//	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })
//	reg_ser := etcd.NewRegServer(
//		config.LocalCfg.ServerEtcdRoot,
//		[]string{fmt.Sprintf("%d", config.LocalCfg.Gid)},
//		etcd.Ser_GmChat, "")
//	signalhandler.SignalKillFunc(func() { reg_ser.UnRegServerVer() })
//	if err := reg_ser.RegServerVer(version.GetVersion()); err != nil {
//		tilogs.L().Errorf("RegServerVer err %s", err.Error())
//		return
//	}
//
//	//建立database
//	db.InitDB()
//	sqlStr := sql_string.SqlString
//	tilogs.L().Infof("create table sql %v", strings.ReplaceAll(sqlStr, "\n", " "))
//	_, err := db.GetDB().Exec(strings.ReplaceAll(sqlStr, "\n", " ")) // 初始化数据表
//	if err != nil {
//		tilogs.L().Errorf("create table err %v", err)
//		return
//	}
//	createAdminAccount()
//
//	// 启动gin服务器
//	r := gin.Default()
//	r.Use(gzip.Gzip(gzip.DefaultCompression))
//	store := memstore.NewStore([]byte("secret"))
//	r.Use(sessions.Sessions("gm", store))
//	logic.RegisterCommand(r)
//	r.Static("/", "./webapp/build/dist")
//	go r.Run(config.LocalCfg.Port)
//
//	// 启动oss
//	if !oss.InitCloudDB() {
//		tilogs.L().Errorf("battleconfig.InitCloudDb fail !!")
//		return
//	}
//	signalhandler.SignalKillFunc(func() {
//		oss.StopCloudDB()
//	})
//
//	waitGroup.Wait()
//
//}

func createAdminAccount() {
	row := db.GetDB().QueryRow("select email from users where email = ?", super_user.AdminRoot)
	var email string
	err := row.Scan(&email)
	if err == sql.ErrNoRows || email == "" {
		pwdWithSalt := []byte(super_user.AdminRoot + super_user.AdminPwd)
		pwdWithMd5 := fmt.Sprintf("%x", md5.Sum(pwdWithSalt))
		_, err := db.GetDB().Exec("insert into users (email, pwd, create_time, user_group) values(?, ?, ?,?)",
			super_user.AdminRoot,
			pwdWithMd5,
			time.Now().Unix(),
			super_user.AccountGroupAdmin)
		if err != nil {
			tilogs.L().Errorf("insert admin error %v", err)
		}
	} else if err != nil {
		tilogs.L().Errorf("create admin root user error %v", err)
	}
}
