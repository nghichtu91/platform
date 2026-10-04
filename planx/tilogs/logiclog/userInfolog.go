package logiclog

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/bilog"
)

const (
	LogType = "BaseUserInfo" // log类型，NT要求的固定名称
)

var (
	userInfoCfg    LogicConfig
	userLvBiLogs   [LevelCount]*bilog.BiLog
	userAllLvBiLog *bilog.BiLog
)

func LoadUserInfoLogic(configFileName string) bool {
	if !loadLogicCfg(configFileName, &userInfoCfg) {
		return false
	}

	ok, all := initAllLvBiLog(userInfoCfg.AllLevelLogic)
	if !ok {
		return false
	}

	if !initLvBiLogs(userInfoCfg.LevelLogic, &userLvBiLogs) {
		return false
	}

	if userInfoCfg.TALogic.TALogicValid {
		initTALogicLog(&userInfoCfg.TALogic)
	}

	userAllLvBiLog = all

	return true
}

func CloseUserInfoLogicLog() {
	lock.Lock()
	defer lock.Unlock()
	// all level
	if userAllLvBiLog != nil {
		if err := userAllLvBiLog.Close(); err != nil {
			tilogs.L().Errorf("CloseUserInfoLogicLog allLevelLog err %s", err.Error())
		}
		userAllLvBiLog = nil
	}
	// level
	for i := range userLvBiLogs {
		_LevelLog := userLvBiLogs[i]
		if _LevelLog != nil {
			if err := _LevelLog.Close(); err != nil {
				tilogs.L().Errorf("CloseUserInfoLogicLog %s err %s", LogicLogLevel(i).String(), err.Error())
			}
		}
		userLvBiLogs[i] = nil
	}

	// taLogic
	if ta != nil {
		once.Do(func() {
			ta.Flush()
			ta.Close()
		})
	}
}

func LogTaUserInfo(com *Common, info interface{}, last string) {
	if userInfoCfg.TALogic.TALogicValid {
		_LogTaLogic(errorLevel, LogType, com, info, last, "", true)
	}
}

func _LogUserInfo(lvl LogicLogLevel, typ string, com *Common, info interface{}, last, designatedTime string) {
	// 机器人不记录BI。
	if com.Channel == "robot" {
		return
	}

	if userAllLvBiLog == nil {
		return
	}

	// 获取UTC+8东八区的UTC时间。
	nt := time.Now()
	utc := nt.UTC().Add(8 * time.Hour)
	logTime := nt.UnixNano()                              // 日志记录时间戳。
	timeUTC := nt.Format("2006-01-02T15:04:05.000Z07:00") // 日志记录UTC时间。
	timeUTC8 := utc.Format("2006-01-02 15:04:05.000")     // 日志记录UTC+8时间（纳秒级）。

	// 获取Marshal后的信息和TaMap。
	l := logUserInfo{}
	bb, err := l.marshal(lvl.String(), typ, com, info, last, logTime, timeUTC, timeUTC8)
	if err != nil {
		tilogs.L().Errorf("logic marshal %s err %s", lvl.String(), err.Error())
		return
	}

	// lock.RLock()
	// defer lock.RUnlock()

	// all level
	if _, err := userAllLvBiLog.Write(bb); err != nil {
		tilogs.L().Errorf("logic Write alllevel err %s", err.Error())
		return
	}
	// level
	if userLvBiLogs[lvl] != nil {
		if _, err := userLvBiLogs[lvl].Write(bb); err != nil {
			tilogs.L().Errorf("logic Write %s err %s", lvl.String(), err.Error())
			return
		}
	}

	// ta
	if userInfoCfg.TALogic.TALogicValid {
		_LogTaLogic(lvl, typ, com, info, last, designatedTime, true)
	}
}

func LogLogicUserTrace(com *Common, info interface{}, last string) {
	_LogUserInfo(traceLevel, LogType, com, info, last, "")
}

func LogLogicUserDebug(com *Common, info interface{}, last string) {
	_LogUserInfo(debugLevel, LogType, com, info, last, "")
}

func LogLogicUserInfo(com *Common, info interface{}, last string) {
	_LogUserInfo(infoLevel, LogType, com, info, last, "")
}

func LogLogicUserWarn(com *Common, info interface{}, last string) {
	_LogUserInfo(warnLevel, LogType, com, info, last, "")
}

func LogLogicUserError(com *Common, info interface{}, last string) {
	_LogUserInfo(errorLevel, LogType, com, info, last, "")
}

type logUserInfo struct {
	Level       string      `json:"level"`
	Type        string      `json:"type"`
	Gid         uint        `json:"gid"`
	Sid         uint        `json:"sid"`
	Channel     string      `json:"channel,omitempty"`
	Uid         string      `json:"uid,omitempty"`
	DeviceId    string      `json:"deviceid,omitempty"` //
	AccountID   string      `json:"accountid,omitempty"`
	PlayerId    int64       `json:"playerid,omitempty"`
	PlayerName  string      `json:"playername,omitempty"`
	PlayerLevel uint32      `json:"playerlevel,omitempty"`
	VIP         int32       `json:"vip,omitempty"`        //
	CreateTime  string      `json:"createtime,omitempty"` // 玩家注册时间的北京时间
	MoneySum    int32       `json:"moneysum,omitempty"`   // 玩家累计充值
	ClientTime  string      `json:"clienttime,omitempty"`
	Info        interface{} `json:"info"`
	Last        string      `json:"last,omitempty"`
	LogTime     int64       `json:"logtime"`  // 产生log的时间戳
	Time        string      `json:"time"`     // 数据记录产生的当地时间
	TimeUTC8    string      `json:"timeutc8"` // 北京时间
}

func (l *logUserInfo) marshal(level string, typ string, c *Common, info interface{}, last string, timeStamp int64, utc string, utc8 string) ([]byte, error) {
	// 填充logUserInfo。
	l.Level = level
	l.Type = typ
	l.Gid = c.Gid
	l.Sid = c.Sid
	l.Channel = c.Channel
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
	l.Info = info
	l.Last = last
	l.LogTime = timeStamp
	l.Time = utc
	l.TimeUTC8 = utc8

	// 获得logUserInfo填充后的Json字符串。
	data, err := json.Marshal(*l)
	if err != nil {
		tilogs.L().Errorf("logUserInfo marshal Error for %s: %s %v", typ, err.Error(), l)
		return nil, fmt.Errorf("logUserInfo marshal Error for %s: %s %v", typ, err.Error(), l)
	}
	data = append(data, "\n"...)

	return data, nil
}
