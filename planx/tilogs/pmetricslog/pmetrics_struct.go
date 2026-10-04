package pmetricslog

import (
	"encoding/json"
	"time"
)

type PMetricsConfig struct {
	FileTemplate    string `toml:"FileTemplate"`    // log文件模板，如：/opt/supervisor/log/logiclog_%y-%m-%d_%h.log
	RotateTimeLocal string `toml:"RotateTimeLocal"` // 时区，一般是"Asia/Shanghai"; 若为空字符串，则为utc时区
	RotateType      int    `toml:"RotateType"`      // log分割方式；目前支持两种，按天和按小时; 0:day,1:hour
	FileForTail     string `toml:"FileForTail"`     // 为运维tail日志导入kibana用，如：/opt/supervisor/log/logiclog.log
	SendOpen        bool   `toml:"SendOpen"`        // 是否开启PMetrics日志的服务。
}

type PMetricsLogInfo struct {
	Gid      uint   `json:"gid"`        // 大区的ID编号
	Sid      string `json:"sid"`        // 区服的ID编号
	LogTime  int64  `json:"log_time"`   // Log的生成时间戳
	Time     string `json:"@timestamp"` // 数据记录的当地时间
	TimeUTC8 string `json:"cn_time"`    // 数据记录的北京时间
	Type     string `json:"type"`       // PMetric的记录类型（Time[本次处理时间]、Count[本次处理数量]、Size[本次大小-字节]）
	Value    int64  `json:"value"`      // PMetric的记录数值
	Name     string `json:"name"`       // 触发日志的行为
}

// LogMarshal 将日志结构体Marshal为Json。
//
// Return-[]byte:Json对应的字节数组。Return-error：Marshal时的异常信息。
func (pm *PMetricsLogInfo) LogMarshal() ([]byte, error) {
	nt := time.Now()
	utc := nt.UTC().Add(8 * time.Hour)
	pm.LogTime = nt.UnixNano()
	pm.Time = nt.Format("2006-01-02T15:04:05.000Z07:00")
	pm.TimeUTC8 = utc.Format("2006-01-02 15:04:05")
	data, err := json.Marshal(*pm)
	if err != nil {
		return nil, err
	}
	data = append(data, "\n"...)
	return data, nil
}
