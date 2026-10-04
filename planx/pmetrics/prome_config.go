package pmetrics

import (
	"github.com/BurntSushi/toml"
	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type PromeConfig struct {
	PushGateWayHost         string `toml:"push_gate_way_host"`         // 服务器发往PushGateWay的IP和端口。
	PushSpecialIntervalTime int64  `toml:"push_special_interval_time"` // 向PushGateWay推送服务器相关Collector的时间间隔。
	PushProcessIntervalTime int64  `toml:"push_process_interval_time"` // 向PushGateWay推送进程相关Collector的时间间隔。
	PushSpecialOpen         bool   `toml:"push_special_open"`          // 是否向PushGateWay推送服务器相关的Collector。
	PushCommonOpen          bool   `toml:"push_common_open"`           // 是否向PushGateWay推送协议相关（协议大小、数量、处理时间）的Collector。
	PushServerProcessOpen   bool   `toml:"push_server_process_open"`   // 是否向PushGateWay推送服务器进程（当前进程：goroutine、gc、mem相关）指标。
	PushServerOSOpen        bool   `toml:"push_server_os_open"`        // 是否向PushGateWay推送服务器操作系统（当前进程：cpu、mem、file相关）指标。【此项即使打开也仅在Linux类型或Windows操作系统环境下生效】
}

// readConfigHandler 读取配置文件时的处理函数。
//
// Param-cfgPath:配置文件名。Param-cmd:加载配置的命令。
func readConfigHandler(cfgPath string, cmd config.LoadCmd) error {
	switch cmd {
	case config.Load, config.Reload:
		var pConfig PromeConfig
		if _, err := toml.DecodeFile(cfgPath, &pConfig); err != nil {
			tilogs.L().Errorf("<Promethues> readConfigHandler DecodeFile failed. Path:%s, Err:%s\n", cfgPath, err.Error())
			return err
		} else {
			tilogs.L().Infof("<Promethues> readConfigHandler Config loaded: %s\n", cfgPath)

			/**
			分析一：Reload被加入ReloadHandler列表中之前，Load已经完成了所有的配置读取和服务配置，
			所以Load和Reload之间不会存在线程安全问题。
			分析二：Reload被加入ReloadHandler列表中之后，Reload不再允许多个线程同时操作，
			所以Reload和Reload之间不会存在线程安全问题。
			根据分析一和分析二，此处无需原子操作。
			*/
			if cmd == config.Reload {
				if err := reload(pConfig); err != nil {
					tilogs.L().Errorf("<Promethues> reload will use old config because reload failed: %v\n", err)
					return err
				}
				return nil
			}
			jobConfig = pConfig
			// 开始Promethues指标推送（服务器推送至PushGateWay，Promethues从PushGateWay刮取）
			if err := startPushMetrics(stopPromethuesChan); err != nil {
				tilogs.L().Errorf("<Promethues> load startPushMetrics failed: %v\n", err)
				return err
			}
			tilogs.L().Infof("<Promethues> readConfigHandler Metrics Prefix %s  Config : %+v", jobPrefix, jobConfig)
		}
	case config.Unload:
		// 关闭周期性监控推送。
		close(stopPromethuesChan)
	}
	return nil
}

// reload 配置文件通过信号Reload时的处理函数。
//
// Param-pConfig:读取到的配置信息。
func reload(pConfig PromeConfig) error {

	// 终止旧的服务
	close(stopPromethuesChan)

	// 配置变更设置
	tempSaveConfig := jobConfig
	jobConfig = pConfig
	stopPromethuesChan = make(chan struct{})

	// 重新开始服务（Promethues指标推送（服务器推送至PushGateWay，Promethues从PushGateWay刮取））
	if err := startPushMetrics(stopPromethuesChan); err != nil {
		close(stopPromethuesChan)
		jobConfig = tempSaveConfig
		stopPromethuesChan = make(chan struct{})
		if err := startPushMetrics(stopPromethuesChan); err != nil {
			tilogs.L().Errorf("<Promethues> reload will use old config, but also failed: %v\n", err)
		}
		return err
	}

	return nil
}
