package logiclog

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/nghichtu91/platform/share/planx/tilogs/bilog"
)

/*
	本logiclog库，用来满足游戏服记录logiclog的需求：
		1：若指定了指定Level的配置，则会把指定Level的log输出到Level对应的log文件中；其他Level的配置输出到AllLevel对应的log文件中
			此功能应用场景，如：一个进程要输出两种日志，则可以用日志等级区分
		2：需要其他log格式的场景，可以用callback功能；如：英雄的bilog需求
*/

type TAData interface {
	GetProperties() map[string]interface{}
	NeedTA() bool
}

type LogicCallback interface {
	Init()
	Close()
	OnMsg(lvl LogicLogLevel, msg string)
}

var (
	gameLogicCfg LogicConfig
	lvBiLogs     [LevelCount]*bilog.BiLog
	allLvBiLog   *bilog.BiLog
	logCallback  LogicCallback
)

func LoadGameLogic(configFileName string) bool {
	if !loadLogicCfg(configFileName, &gameLogicCfg) {
		return false
	}

	ok, all := initAllLvBiLog(gameLogicCfg.AllLevelLogic)
	if !ok {
		return false
	}

	if !initLvBiLogs(gameLogicCfg.LevelLogic, &lvBiLogs) {
		return false
	}

	if gameLogicCfg.AllLevelLogic.Valid {
		initTALogicLog(&gameLogicCfg.TALogic)
	}

	// 初始化
	allLvBiLog = all

	return true
}

func CloseGameLogicLog() {
	lock.Lock()
	defer lock.Unlock()
	// all level
	if allLvBiLog != nil {
		if err := allLvBiLog.Close(); err != nil {
			tilogs.L().Errorf("CloseGameLogicLog allLevelLog err %s", err.Error())
		}
		allLvBiLog = nil
	}
	// level
	for i := range lvBiLogs {
		_LevelLog := lvBiLogs[i]
		if _LevelLog != nil {
			if err := _LevelLog.Close(); err != nil {
				tilogs.L().Errorf("CloseGameLogicLog %s err %s", LogicLogLevel(i).String(), err.Error())
			}
		}
		lvBiLogs[i] = nil
	}
	// callback
	if logCallback != nil {
		logCallback.Close()
		logCallback = nil
	}

	// taLogic
	if ta != nil {
		once.Do(func() {
			ta.Flush()
			ta.Close()
		})
	}
}

func LogGameLogicTrace(typ string, com *Common, info interface{}, last string) {
	_LogGameLogic(traceLevel, typ, com, info, last, "")
}

func LogGameLogicDebug(typ string, com *Common, info interface{}, last string) {
	_LogGameLogic(debugLevel, typ, com, info, last, "")
}

func LogGameLogicInfo(typ string, com *Common, info interface{}, last string) {
	_LogGameLogic(infoLevel, typ, com, info, last, "")
}

func LogGameLogicWarn(typ string, com *Common, info interface{}, last string) {
	_LogGameLogic(warnLevel, typ, com, info, last, "")
}

func LogGameLogicError(typ string, com *Common, info interface{}, last string) {
	_LogGameLogic(errorLevel, typ, com, info, last, "")
}

func LogClientBILogLogicError(typ string, com *Common, info interface{}, last string, designatedTime string) {
	_LogGameLogic(errorLevel, typ, com, info, last, designatedTime)
}

func LogClientBILogLogicInfo(typ string, com *Common, info interface{}, last string, designatedTime string) {
	_LogGameLogic(infoLevel, typ, com, info, last, designatedTime)
}

func _LogGameLogic(lvl LogicLogLevel, typ string, com *Common, info interface{}, last string, designatedTime string) {
	// 机器人不记录BI。
	if com.Channel == consts.RobotChannel {
		return
	}

	// 获取UTC+8东八区的UTC时间。
	nt := time.Now()
	utc := nt.UTC().Add(8 * time.Hour)
	logTime := nt.UnixNano()                              // 日志记录时间戳（纳秒级）。
	timeUTC := nt.Format("2006-01-02T15:04:05.000Z07:00") // 日志记录UTC时间。
	timeUTC8 := utc.Format("2006-01-02 15:04:05.000")     // 日志记录UTC+8时间。
	taTime := timeUTC8
	if designatedTime != "" {
		taTime = designatedTime
	}

	// 获取Marshal后的信息和TaMap。
	l := gameLoggerInfo{}
	bb, err := l.marshal(lvl.String(), typ, com, info, last, logTime, timeUTC, timeUTC8, taTime)
	if err != nil {
		tilogs.L().Errorf("logic marshal %s err %s", lvl.String(), err.Error())
		return
	}

	// lock.RLock()
	// defer lock.RUnlock()
	if allLvBiLog == nil {
		return
	}
	// level
	if lvBiLogs[lvl] != nil {
		if _, err := lvBiLogs[lvl].Write(bb); err != nil {
			tilogs.L().Errorf("logic Write %s err %s", lvl.String(), err.Error())
			return
		}
	} else {
		// all level
		if _, err := allLvBiLog.Write(bb); err != nil {
			tilogs.L().Errorf("logic Write alllevel err %s", err.Error())
			return
		}
	}
	// callback
	if logCallback != nil {
		logCallback.OnMsg(lvl, string(bb))
	}

	// ta
	_LogTaLogic(lvl, typ, com, info, last, designatedTime, false)
}

type gameLoggerInfo struct {
	Level                    string      `json:"level"`
	Type                     string      `json:"type"`
	Gid                      uint        `json:"gid"`
	Sid                      uint        `json:"sid"`
	Channel                  string      `json:"channel"`
	CreateChannel            string      `json:"createchannel"`
	Uid                      string      `json:"uid"`
	DeviceId                 string      `json:"deviceid"` //
	AccountID                string      `json:"accountid"`
	PlayerId                 int64       `json:"playerid"`
	PlayerName               string      `json:"playername"`
	PlayerLevel              uint32      `json:"playerlevel"`
	VIP                      int32       `json:"vip"`        //
	CreateTime               string      `json:"createtime"` // 玩家注册时间的北京时间
	MoneySum                 int32       `json:"moneysum"`   // 玩家累计充值
	ClientTime               string      `json:"clienttime"`
	ClientBuildTimeAndHotVer string      `json:"clientbuildtimeandhotver"` // 客户端编包时间和热更号，几号包_编包时间_热更号
	Language                 string      `json:"language"`                 // 客户端用的语言
	Info                     interface{} `json:"info"`
	Last                     string      `json:"last,omitempty"`
	LogTime                  int64       `json:"logtime"`  // 产生log的时间戳
	Time                     string      `json:"time"`     // 产生log的当地时间
	TimeUTC8                 string      `json:"timeutc8"` // 产生log的北京时间
	TATime                   string      `json:"tatime"`   // 用于进ta的时间
}

func (l *gameLoggerInfo) marshal(level, typ string, c *Common, info interface{}, last string,
	timeStamp int64, utc, utc8, taTime string) ([]byte, error) {
	// 填充GameLoggerInfo。
	l.Level = level
	l.Type = typ
	l.Gid = c.Gid
	l.Sid = c.Sid
	l.Channel = c.Channel
	l.CreateChannel = c.CreateChannel
	l.Uid = c.Uid
	l.DeviceId = c.DeviceId
	l.AccountID = c.AccountID
	l.PlayerId = c.PlayerId
	l.PlayerName = c.PlayerName
	l.PlayerLevel = c.PlayerLevel
	l.VIP = c.VIP
	l.CreateTime = c.CreateTime
	l.MoneySum = c.MoneySum
	l.ClientTime = c.ClientTime
	l.ClientBuildTimeAndHotVer = c.ClientBuildTimeAndHotVer
	l.Language = c.Language
	l.Info = info
	l.Last = last
	l.LogTime = timeStamp
	l.Time = utc
	l.TimeUTC8 = utc8
	l.TATime = taTime

	// 获得GameLoggerInfo填充后的Json字符串。
	data, err := json.Marshal(*l)
	if err != nil {
		return nil, fmt.Errorf("gameLoggerInfo marshal Error for %s: %s %+v", typ, err.Error(), l)
	}
	data = append(data, "\n"...)

	return data, nil
}
