package pmetricslog

import (
	"fmt"
	"sync"

	"github.com/nghichtu91/platform/share/planx/tilogs/bilog"

	"github.com/BurntSushi/toml"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/config"
)

var (
	cfg PMetricsConfig

	Gid      uint
	ServerID string
	JobName  string

	PMetricsLog *bilog.BiLog
	logLock     sync.RWMutex
)

// LoadPMetricsLog 加载prometheus的metrics相关日志服务。
//
// Param-cfgPath：配置文件名。Param-gid：服务器所在大区。Param-sid:服务器所在区服（不存在则为空）
//
// Return-bool: 加载日志服务是否成功。
func LoadPMetricsLog(cfgPath string, gid uint, serverID string, jobName string) bool {
	Gid = gid
	ServerID = serverID
	JobName = jobName

	// 读取配置文件
	if res := config.NewConfig(cfgPath, true, readConfigHandler); res == nil {
		return false
	}
	return true
}

// readConfigHandler 读取配置文件时的处理函数。
//
// Param-cfgPath：配置文件名。Param-cmd:读取配置的命令。
func readConfigHandler(cfgPath string, cmd config.LoadCmd) error {
	// 获取配置文件内容
	var tempCfg PMetricsConfig
	if _, err := toml.DecodeFile(cfgPath, &tempCfg); err != nil {
		tilogs.L().Errorf("<pmetricslog> LoadPMetricsLog readConfigHandler DecodeFile failed. %s, %s\n", cfgPath, err.Error())
		return err
	} else {
		tilogs.L().Infof("<pmetricslog> LoadPMetricsLog readConfigHandler config loaded: %s\n", cfgPath)
	}

	switch cmd {
	case config.Load, config.Reload:
		if !readConfigLoadOrReload(tempCfg) {
			return fmt.Errorf("readConfigHandler.readConfigLoadOrReload fail")
		}
	}
	return nil
}

// readConfigLoadOrReload 加载或重新加载BILOG。
//
// Param-tempCfg:配置信息。
func readConfigLoadOrReload(tempCfg PMetricsConfig) bool {
	logLock.Lock()
	defer logLock.Unlock()

	if tempCfg.FileTemplate == "" || tempCfg.RotateTimeLocal == "" {
		tilogs.L().Errorf("<pmetricslog> readConfigLoadOrReload config defect: %s\n", tempCfg)
		return false
	}

	if !tempCfg.SendOpen {
		cfg = tempCfg
		PMetricsLog = nil
	} else {
		tempPMetricsLog := bilog.CreateBiLog(tempCfg.FileTemplate, tempCfg.RotateTimeLocal, bilog.RotateType(tempCfg.RotateType), tempCfg.FileForTail)
		if tempPMetricsLog == nil {
			tilogs.L().Errorf("<pmetricslog> bilog.CreateBiLog nil")
			return false
		}

		cfg = tempCfg
		PMetricsLog = tempPMetricsLog
	}
	return true
}

// LogPMetrics 记录prometheus的metrics相关日志
//
// Param-info:由上层信息构建的日志结构体。
func LogPMetrics(info *PMetricsLogInfo) {
	contentBytes, err := info.LogMarshal()
	if err != nil {
		tilogs.L().Errorf("<pmetricslog> LogPMetrics %v err %v", *info, err.Error())
		return
	}

	logLock.RLock()
	defer logLock.RUnlock()

	if PMetricsLog == nil {
		return
	}

	if _, err := PMetricsLog.Write(contentBytes); err != nil {
		tilogs.L().Errorf("<pmetricslog> PMetricsLog.Write err %v", err.Error())
		return
	}

}
