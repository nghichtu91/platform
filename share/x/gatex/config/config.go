package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/nghichtu91/platform/share/planx/etcd"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/BurntSushi/toml"

	"github.com/nghichtu91/platform/share/planx/config"
	cconfig "github.com/nghichtu91/platform/share/x/common/config"
)

type BasicCfg struct {
	EtcdEndpoint []string `toml:"etcd_endpoint"`
	EtcdServer   string   `toml:"etcd_server"`
	EtcdDevops   string   `toml:"etcd_devops"`

	Gid      uint   `toml:"gid"`
	ServerId string `toml:"serverid"`

	ElbAddr         []string `toml:"elbaddr"`
	LSBDeadLine     int      `toml:"LSBDeadLine"`
	PublicIP        string   `toml:"publicip"`
	InternalIp      string   `toml:"internalip"`
	Listen          string   `toml:"listen"`
	MaxConn         uint     `toml:"maxconn"`
	CloudGAAddress  string   `toml:"CloudGAAddress"`
	GameConnNum     uint     `toml:"game_conn_num"`
	GameServers     []string `toml:"gameservers"`
	ConnServer      string   `toml:"connetion_server"`
	NAcceptor       uint     `toml:"acceptors"`
	NWaitingConn    uint     `toml:"waiting_queue"`
	ChatxRedisAddr  string   `toml:"ChatxRedisAddr"`
	ChatxRedisDb    string   `toml:"ChatxRedisDb"`
	ChatxRedisDbPwd string   `toml:"ChatxRedisDbPwd"`

	SslCfg     SSLCertCfg     `toml:"SslCfg"`     // 自签名证书
	HostSslCfg HostSSLCertCfg `toml:"HostSslCfg"` // 带有Host的证书
}

// GetPublicHost 获取公网地址. 如果配置了Host, 则返回Host, 否则返回PublicIP
func (b *BasicCfg) GetPublicHost() string {
	host := b.HostSslCfg.Host
	if host == "" {
		return b.PublicIP
	}
	addr, err := net.ResolveTCPAddr("tcp", b.PublicIP)
	if err != nil {
		panic(fmt.Sprintf("GetPublicHost %s has problem. %s", host, err.Error()))
	}
	return host + ":" + strconv.Itoa(addr.Port)
}

// GetPublicIP 获取公网IP
func (b *BasicCfg) GetPublicIP() string {
	return b.PublicIP
}

type SSLCertCfg struct {
	Cert string `toml:"cert"` // 证书
	Key  string `toml:"key"`  // 私钥
	CA   string `toml:"ca"`   // CA证书. 自签名证书需要
}

type HostSSLCertCfg struct {
	SSLCertCfg
	Host string `toml:"host"` // 主机名
}

func (s SSLCertCfg) IsSet() bool {
	if s.Cert != "" && s.Key != "" {
		return true
	}
	return false
}

// HasCA 是否有CA证书
func (s SSLCertCfg) HasCA() bool {
	if s.CA != "" {
		return true
	}
	return false
}

type LoginStatus struct {
	LoginToken string
	AccountID  string
	LogInOff   bool
}

type CcumetricsConfig struct {
	LoginConnectorTickTime int64 `toml:"LoginConnector_tick_time"`
}

type Config struct {
	GateConfig       BasicCfg
	CCUMetricsConfig CcumetricsConfig
}

var (
	Cfg         Config
	EtcdConf    cconfig.GidConfig
	RunMode     string
	CheatEnable string
	Proj        string
)

const (
	RunModeLocal = "local"
	//RunModeDev 开发模式
	RunModeDev = "dev"
	// RunModeTest 测试模式
	RunModeTest = "test"
	//RunModeProd 生产模式
	RunModeProd = "prod"
)

func LoadConfig(cfgname string) bool {
	if nil == config.NewConfig(cfgname, true, func(lcfgname string, cmd config.LoadCmd) error {
		switch cmd {
		case config.Load: //, config.Reload:
			var c Config
			if _, err := toml.DecodeFile(lcfgname, &c); err != nil {
				tilogs.L().Errorf("App config load failed. %s, %s\n", lcfgname, err.Error())
				return err
			} else {
				tilogs.SetTiLogger(tilogs.L().With(tilogs.TagServerId, c.GateConfig.ServerId).With(tilogs.TagGid, c.GateConfig.Gid))
				tilogs.L().Infof("Config loaded: %s\n", lcfgname)
				Cfg = c
			}
		}
		return nil
	}) {
		return false
	}

	if err := FixConfigInLocalEnvironment(); nil != err {
		tilogs.L().Errorf("FixConfigInLocalEnvironment error, %v", err)
		return false
	}

	if err := etcd.InitEtcd(Cfg.GateConfig.EtcdEndpoint); err != nil {
		tilogs.L().Errorf("etcd.InitEtcd err %s", err.Error())
		return false
	}

	runmode, err := cconfig.GetRunMode(Cfg.GateConfig.EtcdDevops,
		Cfg.GateConfig.Gid)
	if err != nil {
		tilogs.L().Errorf("get runmode err %s", err.Error())
		return false
	}
	cheatEnable, err := cconfig.GetCheatEnable(Cfg.GateConfig.EtcdDevops,
		Cfg.GateConfig.Gid)
	if err != nil {
		tilogs.L().Errorf("get cheatEnable err %s", err.Error())
		return false
	}

	proj, err := cconfig.GetProj(Cfg.GateConfig.EtcdDevops,
		Cfg.GateConfig.Gid)
	if err != nil {
		tilogs.L().Errorf("get proj err %s", err.Error())
		return false
	}

	// 加载etcd配置
	if !LoadConfigFromEtcd() {
		tilogs.L().Errorf("load etcd config failed")
		return false
	}

	RunMode = runmode
	CheatEnable = cheatEnable
	Proj = proj
	return true
}

func LoadConfigFromEtcd() bool {
	pConfig := cconfig.InitLoadGidConfig(Cfg.GateConfig.EtcdDevops, Cfg.GateConfig.Gid)
	if pConfig == nil {
		return false
	}
	EtcdConf = *pConfig
	return true
}

// FixConfigInLocalEnvironment fix some config item with local environment parameters. (for developer execute in local environment)
func FixConfigInLocalEnvironment() error {
	if ipStr, exist := os.LookupEnv("DEV_Local_Public_IP"); exist {
		arr := strings.Split(Cfg.GateConfig.PublicIP, ":")
		if 2 > len(arr) {
			return fmt.Errorf("Invalid PublicIP config [%v]", Cfg.GateConfig.PublicIP)
		}
		Cfg.GateConfig.PublicIP = ipStr + ":" + arr[1]
	}
	return nil
}
