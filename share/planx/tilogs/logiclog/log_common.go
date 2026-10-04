package logiclog

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ThinkingDataAnalytics/go-sdk/thinkingdata"
	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/bilog"
)

/*
	该文件中定义log系统通用的数据结构和方法
*/

// Common 隐藏BI埋点公共字段
// Note: 即使Tag中存在InWord问题，也请不要修改，因为TA和Kibana要统一
type Common struct {
	Gid                      uint   `json:"gid"`
	Sid                      uint   `json:"sid"`
	Channel                  string `json:"channel"`
	CreateChannel            string `json:"createchannel"`
	Uid                      string `json:"uid"`
	DeviceId                 string `json:"deviceid"`
	AccountID                string `json:"accountid"`
	PlayerId                 int64  `json:"playerid"`
	PlayerName               string `json:"playername"`
	PlayerLevel              uint32 `json:"playerlevel"`
	VIP                      int32  `json:"vip"`
	CreateTime               string `json:"createtime"` // 玩家注册时间的北京时间
	MoneySum                 int32  `json:"moneysum"`   // 玩家累计充值
	ClientTime               string `json:"clienttime"`
	ClientBuildTimeAndHotVer string `json:"clientbuildtimeandhotver"` // 客户端编包时间和热更号，时间_热更号
	Language                 string `json:"language"`                 // 客户端用的语言
	TAPushReceipt            string `json:"-"`                        // 数数推送回执，非必须项
	PackageType              int32  `json:"packageType"`              // 客户端包类型
	DistinctID               string `json:"distinctid"`               // 海外为了打通Appfly，使用中台生成的唯一ID替换deviceID
}

type LogicLogLevel int

const (
	traceLevel LogicLogLevel = iota
	debugLevel
	infoLevel
	warnLevel
	errorLevel
	LevelCount
)

var (
	lock sync.RWMutex
	ta   *thinkingdata.TDAnalytics
	once sync.Once
)

const ModAccount = "mod" // 为好统计，所有非玩家为主体的埋点中的AccountID都用这个常量

func (logLv LogicLogLevel) String() string {
	switch logLv {
	case traceLevel:
		return "trace"
	case debugLevel:
		return "debug"
	case infoLevel:
		return "info"
	case warnLevel:
		return "warn"
	case errorLevel:
		return "error"
	default:
		return "wrongLogicLogLevel"
	}
}

func (logLv *LogicLogLevel) unmarshalText(text []byte) bool {
	switch string(text) {
	case "trace", "TRACE":
		*logLv = traceLevel
	case "debug", "DEBUG":
		*logLv = debugLevel
	case "info", "INFO", "": // make the zero value useful
		*logLv = infoLevel
	case "warn", "WARN":
		*logLv = warnLevel
	case "error", "ERROR":
		*logLv = errorLevel
	default:
		return false
	}
	return true
}

type AllLogConfig struct {
	Valid           bool   // 功能是否开启，若LogicConfig未开启，则LogicLevelConfig也不会开启
	FileTemplate    string // log文件模板，如：/opt/supervisor/log/logiclog_%y-%m-%d_%h.log
	RotateTimeLocal string // 时区，一般是"Asia/Shanghai"; 若为空字符串，则为utc时区
	RotateType      int    // log分割方式；目前支持两种，按天和按小时; 0:day,1:hour
	FileForTail     string // 为运维tail日志导入kibana用，如：/opt/supervisor/log/logiclog.log
}

type LevelLogConfig struct {
	Level           string // log的level
	FileTemplate    string // log文件模板，如：/opt/supervisor/log/logiclog_%y-%m-%d_%h.log
	RotateTimeLocal string // 时区，一般是"Asia/Shanghai"; 若为空字符串，则为utc时区
	RotateType      int    // log分割方式；目前支持两种，按天和按小时; 0:day,1:hour
}

type TALogConfig struct {
	ConfigPath     string // TA的日志生成目标文件夹。
	ConfigSplit    uint   // 按天（0）/小时（1）划分文件。
	ConfigFileSize int    // 文件上限大小。
	TALogicValid   bool   // 是否开启TALogicLog。
}

type LogicConfig struct {
	AllLevelLogic AllLogConfig
	LevelLogic    []LevelLogConfig
	TALogic       TALogConfig
}

// 加载LogicConfig
func loadLogicCfg(configFileName string, logicCfg *LogicConfig) bool {
	res := config.NewConfig(configFileName, true, func(lcfgname string, cmd config.LoadCmd) error {
		if _, err := toml.DecodeFile(lcfgname, logicCfg); err != nil {
			tilogs.L().Errorf("App config load failed. %s, %s\n", lcfgname, err.Error())
			return err
		} else {
			tilogs.L().Infof("Config loaded: %s\n", lcfgname)
		}

		switch cmd {
		case config.Load, config.Reload:
			if !func(c *LogicConfig) bool {
				lock.Lock()
				defer lock.Unlock()
				if !c.AllLevelLogic.Valid {
					return true
				}
				if c.AllLevelLogic.FileTemplate == "" || c.AllLevelLogic.RotateTimeLocal == "" {
					tilogs.L().Errorf("loadLogicCfg AllLevelLogic cfg wrong %v", c)
					return false
				}
				return true
			}(logicCfg) {
				return fmt.Errorf("loadLogicCfg fail")
			}
		}
		return nil
	})
	return res != nil
}

// 初始化allLvBiLog
func initAllLvBiLog(allLog AllLogConfig) (bool, *bilog.BiLog) {
	// all level
	biLog := bilog.CreateBiLog(allLog.FileTemplate, allLog.RotateTimeLocal, bilog.RotateType(allLog.RotateType), allLog.FileForTail)
	if biLog == nil {
		tilogs.L().Errorf("initAllLvBiLog fail bilog.CreateBiLog allLevelLog nil")
		return false, nil
	}
	return true, biLog
}

// 初始化lvBiLogs
func initLvBiLogs(logs []LevelLogConfig, lvBiLogs *[LevelCount]*bilog.BiLog) bool {
	if len(logs) <= 0 {
		return true
	}
	for _, log := range logs {
		var logLv LogicLogLevel
		if !logLv.unmarshalText([]byte(log.Level)) {
			tilogs.L().Errorf("initLevelLogic Level %s not recognize", log.Level)
			return false
		}

		_Log := bilog.CreateBiLog(log.FileTemplate, log.RotateTimeLocal, bilog.RotateType(log.RotateType), "")
		if _Log == nil {
			tilogs.L().Errorf("initLevelLogic bilog.CreateBiLog %s nil", logLv.String())
			return false
		}
		lvBiLogs[logLv] = _Log
	}
	return true
}

// 初始化taLog
func initTALogicLog(taLog *TALogConfig) bool {
	if taLog.ConfigPath == "" {
		return true
	}
	// 确保目标文件夹存在。
	_, errStat := os.Stat(taLog.ConfigPath)
	if errStat != nil {
		if os.IsNotExist(errStat) {
			if errMk := os.Mkdir(taLog.ConfigPath, os.ModePerm); errMk != nil {
				tilogs.L().Errorf("initTALogicLog os.Mkdir %v err %v", taLog.ConfigPath, errMk)
				return false
			}
		} else {
			tilogs.L().Errorf("initTALogicLog os.Stat %v err %v", taLog.ConfigPath, errStat)
			return false
		}
	}
	// 创建本地文件写入的Consumer。
	consumer, errConsumer := thinkingdata.NewLogConsumerWithFileSize(taLog.ConfigPath, thinkingdata.RotateMode(taLog.ConfigSplit), taLog.ConfigFileSize)
	if errConsumer != nil {
		tilogs.L().Errorf("initTALogicLog NewLogConsumerWithFileSize err: %v", errConsumer)
		return false
	}

	// 创建TA实例。
	handler := thinkingdata.New(consumer)
	ta = &handler
	return true
}

// 发送ta日志
func _LogTaLogic(lvl LogicLogLevel, typ string, com *Common, info interface{}, last, designatedTime string, isUser bool) {
	// 机器人不记录BI。
	if com.Channel == "robot" || ta == nil {
		return
	}

	// lock.RLock()
	// defer lock.RUnlock()

	// 获取TA事件属性Map。
	taHandler, success := info.(TAData)
	if !success {
		tilogs.L().Errorf("_LogTaLogic marshal  info.(TAData) %v err， type: %v", info, typ)
		return
	}

	// 检查是否需要TA记录。
	if !taHandler.NeedTA() {
		return
	}

	// 获取UTC+8东八区的UTC时间。
	nt := time.Now()
	_, offset := nt.Zone()
	utc := nt.UTC().Add(8 * time.Hour)                    // 日志记录时间戳。
	timeUTC := nt.Format("2006-01-02T15:04:05.000Z07:00") // 日志记录UTC时间。
	timeUTC8 := utc.Format("2006-01-02 15:04:05")         // 日志记录UTC+8时间。

	taPro := taHandler.GetProperties()
	if designatedTime != "" {
		taPro["#time"] = designatedTime
	}
	taPro["#zone_offset"] = int64(offset / 3600)
	addCommonTa(taPro, com, lvl.String(), typ, last, timeUTC, timeUTC8)

	// 数数推送回执
	if com.TAPushReceipt != "" {
		taPro["#ops_receipt_properties"] = com.TAPushReceipt
	}

	// JWS2-45976 为了打通Appfly，使用中台生成的唯一ID替换deviceID
	// 为了兼容旧存档不存在distinctID的情况，做保底判断
	distinctID := com.DistinctID
	if distinctID == "" {
		distinctID = com.DeviceId
	}

	if isUser {
		if err := ta.UserSet(com.AccountID, distinctID, taPro); err != nil {
			tilogs.L().Errorf("_LogTaLogic ta.Track err: %v, typ %v  %v", err, typ, taPro)
			return
		}
	} else {
		if err := ta.Track(com.AccountID, distinctID, typ, taPro); err != nil {
			tilogs.L().Errorf("_LogGameLogic ta.Track err: %v, typ %v  %v", err, typ, taPro)
			return
		}
	}
}

// 给log消息添加公共的消息头
func addCommonTa(taPro map[string]interface{}, com *Common, lvl, typ, last, timeUTC, timeUTC8 string) {
	// taPro := taHandler.GetProperties()
	taPro["Gid"] = com.Gid
	taPro["Sid"] = com.Sid
	taPro["Channel"] = com.Channel
	taPro["Uid"] = com.Uid
	taPro["DeviceId"] = com.DeviceId
	taPro["AccountID"] = com.AccountID
	taPro["PlayerId"] = com.PlayerId
	taPro["PlayerName"] = com.PlayerName
	taPro["PlayerLevel"] = com.PlayerLevel
	taPro["VIP"] = com.VIP
	if com.CreateTime != "" {
		taPro["CreateTime"] = com.CreateTime
	}
	taPro["CreateChannel"] = com.CreateChannel
	taPro["MoneySum"] = com.MoneySum
	taPro["ClientTime"] = com.ClientTime
	taPro["Level"] = lvl
	taPro["Type"] = typ
	taPro["Last"] = last
	taPro["Time"] = timeUTC
	taPro["TimeUTC8"] = timeUTC8
	taPro["ClientBuildTimeAndHotVer"] = com.ClientBuildTimeAndHotVer
	taPro["Language"] = com.Language
}
