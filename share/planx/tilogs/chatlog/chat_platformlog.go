package chatlog

import (
	"strings"
	"sync"

	"github.com/nghichtu91/platform/share/planx/tilogs/bilog"
	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/BurntSushi/toml"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/config"
)

var (
	cfg ChatPlatFormConfig

	Gid uint

	ChatLog *bilog.BiLog
	logLock sync.RWMutex
)

// LoadChatPlatformLog 加载chat platform的metrics相关日志服务。
//
// Param-cfgPath：配置文件名。Param-gid：服务器所在大区。
//
// Return-bool: 加载日志服务是否成功。
func LoadChatPlatformLog(cfgPath string, gid uint) bool {
	Gid = gid

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
	var tempCfg ChatPlatFormConfig
	if _, err := toml.DecodeFile(cfgPath, &tempCfg); err != nil {
		tilogs.L().Errorf("<chatlog> LoadChatPlatformLog readConfigHandler DecodeFile failed. %s, %s\n", cfgPath, err.Error())
		return err
	} else {
		tilogs.L().Infof("<chatlog> LoadChatPlatformLog readConfigHandler config loaded: %s\n", cfgPath)
	}

	switch cmd {
	case config.Load, config.Reload:
		readConfigLoadOrReload(tempCfg)
	}
	return nil
}

// readConfigLoadOrReload 加载或重新加载BILOG。
//
// Param-tempCfg:配置信息。
func readConfigLoadOrReload(tempCfg ChatPlatFormConfig) {
	logLock.Lock()
	defer logLock.Unlock()

	if tempCfg.FileTemplate == "" || tempCfg.RotateTimeLocal == "" {
		tilogs.L().Errorf("<chatlog> readConfigLoadOrReload config defect: %s\n", tempCfg)
		return
	}

	tempPMetricsLog := bilog.CreateBiLog(tempCfg.FileTemplate, tempCfg.RotateTimeLocal, bilog.RotateType(tempCfg.RotateType), tempCfg.FileForTail)
	if tempPMetricsLog == nil {
		tilogs.L().Errorf("<pmetricslog> bilog.CreateBiLog nil")
		return
	}

	cfg = tempCfg
	ChatLog = tempPMetricsLog
}

func LogChatPlatform(com *ChatPlatFormInfo) {
	// 过滤机器人
	if strings.HasPrefix(com.Operator, "robot:") || com.Sender == consts.RobotChannel {
		return
	}

	l := &ChatPlatFormInfo{}
	l.Gid = com.Gid
	l.LogicId = com.LogicId
	l.Type = com.Type
	l.Operator = com.Operator

	// info
	l.IsRoomMsg = com.IsRoomMsg
	l.Sender = com.Sender
	l.Receiver = com.Receiver
	l.SenderDevcie = com.SenderDevcie
	l.SendTime = com.SendTime
	l.Content = com.Content
	l.GameSerId = com.GameSerId
	l.SenderName = com.SenderName
	l.IsTrumpet = com.IsTrumpet

	logInfo(l)
}

// LogChatPlatform 记录聊天平台相关信息
//
// Param-info:由上层信息构建的日志结构体。
func logInfo(info *ChatPlatFormInfo) {
	contentBytes, err := info.LogMarshal()
	if err != nil {
		tilogs.L().Errorf("<chatlog> LogChatPlatform%v err %v", *info, err.Error())
		return
	}

	logLock.RLock()
	defer logLock.RUnlock()

	if ChatLog == nil {
		return
	}

	if _, err := ChatLog.Write(contentBytes); err != nil {
		tilogs.L().Errorf("<chatlog> LogChatPlatform.Write err %v", err.Error())
		return
	}
}

// LogTranslate 翻译埋点
func LogTranslate(com *TranslateInfo) {
	contentBytes, err := com.LogMarshal()
	if err != nil {
		tilogs.L().Errorf("<chatlog> LogTranslate%v err %v", *com, err.Error())
		return
	}

	logLock.RLock()
	defer logLock.RUnlock()

	if ChatLog == nil {
		return
	}

	if _, err := ChatLog.Write(contentBytes); err != nil {
		tilogs.L().Errorf("<chatlog> LogTranslate.Write err %v", err.Error())
		return
	}
}
