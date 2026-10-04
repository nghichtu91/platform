package metrics

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"unsafe"

	net2 "github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/BurntSushi/toml"
	gm "github.com/rcrowley/go-metrics"

	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/metrics/simplegraphite"
)

var (
	_proj                    string
	_prefix                  string // 前缀改为由程序提供，而不是在配置文件中由运维提供
	_cfg                     Config
	_stop_graphite           chan struct{}
	_goStatsPerSeond         gm.Registry
	_goCurstomStatsPerSecond gm.Registry
	_ip                      string
	_simpleGraphite          *simplegraphite.Graphite
)

func init() {
	_stop_graphite = make(chan struct{})
	_goStatsPerSeond = gm.NewRegistry()
	_goCurstomStatsPerSecond = gm.NewRegistry()
	ip, err := net2.GetExternalIP()
	if err != nil {
		panic(err)
	}
	_ip = strings.Replace(ip, ".", "_", -1)
	_ip = strings.Replace(_ip, ":", "_", -1)
	_p := simplegraphite.NewGraphiteNop("127.0.0.1", 2003)
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&(_simpleGraphite))),
		unsafe.Pointer(_p))

}

func GetIPToken() string {
	return _ip
}

//func GetRegister() gm.Registry {
//return _goStatsPerSeond
//}

func NewCounter(name string) gm.Counter {
	return gm.NewRegisteredCounter(name, _goStatsPerSeond)
}

func NewMeter(name string) gm.Meter {
	return gm.NewRegisteredMeter(name, _goStatsPerSeond)
}

func NewGauge(name string) gm.Gauge {
	return gm.NewRegisteredGauge(name, _goStatsPerSeond)
}

func NewHistogram(name string) gm.Histogram {
	return gm.NewRegisteredHistogram(name, _goStatsPerSeond, gm.NewUniformSample(100))
}

func NewCustomCounter(name string) gm.Counter {
	return gm.NewRegisteredCounter(name, _goCurstomStatsPerSecond)
}

func NewCustomMeter(name string) gm.Meter {
	return gm.NewRegisteredMeter(name, _goCurstomStatsPerSecond)
}

func NewCustomGauge(name string) gm.Gauge {
	return gm.NewRegisteredGauge(name, _goCurstomStatsPerSecond)
}

func NewCustomHistogram(name string) gm.Histogram {
	return gm.NewRegisteredHistogram(name, _goCurstomStatsPerSecond, gm.NewExpDecaySample(2048, 0.015))
}

func Start(cfgname, proj, prefix string) bool {
	_proj = proj
	signalhandler.SetServiceId(prefix)
	_prefix = fmt.Sprintf("%s.%s", proj, prefix)
	if res := config.NewConfig(cfgname, true, func(lcfgname string, cmd config.LoadCmd) error {
		switch cmd {
		case config.Load, config.Reload:
			var c Config
			if _, err := toml.DecodeFile(lcfgname, &c); err != nil {
				tilogs.L().Errorf("App config load failed. %s, %s\n", lcfgname, err.Error())
				return err
			} else {
				tilogs.L().Infof("Config loaded: %s\n", lcfgname)
				if cmd == config.Reload {
					Reload(c)
					return nil
				}
				_cfg = c
				tilogs.L().Infof("Metrics prefix %s  Config _cfg: %+v", _prefix, _cfg)
			}
		case config.Unload:
			close(_stop_graphite)
			_op := (*simplegraphite.Graphite)(
				atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&(_simpleGraphite)))))
			if _op != nil {
				_op.Disconnect()
			}
			_p := simplegraphite.NewGraphiteNop("127.0.0.1", 2003)
			atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&(_simpleGraphite))),
				unsafe.Pointer(_p))
		}
		return nil
	}); res == nil {
		tilogs.L().Errorf("NewConfig load %s failed", cfgname)
	}

	tilogs.L().Infof("metric started.")
	//统计状态数据应该可以通过信号Reload
	go startGolangStats(_stop_graphite)
	_op := (*simplegraphite.Graphite)(
		atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&(_simpleGraphite)))))
	if _op != nil {
		if err := _op.Disconnect(); err != nil {
			tilogs.L().Errorf("metric Start Disconnect err %s", err.Error())
			return false
		}
	}
	if sg, err := NewSimpleGraphie(_cfg); err == nil {
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&(_simpleGraphite))),
			unsafe.Pointer(sg))
	} else {
		tilogs.L().Errorf("NewSimpleGraphie err: %s", err.Error())
		return false
	}
	return true
}

func NewSimpleGraphie(cfg Config) (*simplegraphite.Graphite, error) {
	if !cfg.SimpleGraphite {
		return nil, nil
	}
	host, port, err := net.SplitHostPort(_cfg.GraphiteHost)
	if err != nil {
		return nil, err
	}
	if pport, err := strconv.Atoi(port); err != nil {
		return nil, err
	} else {
		if gg, err := simplegraphite.NewGraphiteUDP(host, pport); err != nil {
			return nil, err
		} else {
			return gg, nil
		}
	}
}

func Reload(cfg Config) {
	close(_stop_graphite)
	_cfg = cfg
	_stop_graphite = make(chan struct{})
	go startGolangStats(_stop_graphite)

	_op := (*simplegraphite.Graphite)(
		atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&(_simpleGraphite)))))
	if _op != nil {
		_op.Disconnect()
	}
	if sg, err := NewSimpleGraphie(_cfg); err == nil {
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&(_simpleGraphite))),
			unsafe.Pointer(sg))
	} else {
		tilogs.L().Errorf("NewSimpleGraphie err: %s", err.Error())
	}
}

func Stop() {
	close(_stop_graphite)
}

func SimpleSend(stat string, value string) error {
	if !_cfg.SimpleGraphite {
		return nil
	}
	_op := (*simplegraphite.Graphite)(
		atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&(_simpleGraphite)))))
	if _op != nil {
		return _op.SimpleSend(_proj+"."+stat, value)
	}
	return nil
}

func IsMetricsValid() bool {
	return _cfg.SimpleGraphite
}
