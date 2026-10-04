package chatlog

import (
	"encoding/json"
	"time"
)

type ChatPlatFormConfig struct {
	FileTemplate    string `toml:"FileTemplate"`    // log文件模板，如：/opt/supervisor/log/logiclog_%y-%m-%d_%h.log
	RotateTimeLocal string `toml:"RotateTimeLocal"` // 时区，一般是"Asia/Shanghai"; 若为空字符串，则为utc时区
	RotateType      int    `toml:"RotateType"`      // log分割方式；目前支持两种，按天和按小时; 0:day,1:hour
	FileForTail     string `toml:"FileForTail"`     // 为运维tail日志导入kibana用，如：/opt/supervisor/log/logiclog.log
}

type ChatPlatFormInfo struct {
	Gid          uint   `json:"gid"`           // 大区的ID编号
	LogicId      string `json:"logicic"`       // logic server id
	LogTime      int64  `json:"log_time"`      // Log的生成时间戳
	Time         string `json:"@timestamp"`    // 数据记录的当地时间
	TimeUTC8     string `json:"cn_time"`       // 数据记录的北京时间
	Type         string `json:"type"`          // 记录类型
	GameSerId    string `json:"gameserver_id"` // 游戏服务器id
	Operator     string `json:"operator"`      // 谁触发的log
	IsRoomMsg    bool   `json:"isroom"`        // 是否是房间消息
	Sender       string `json:"sender"`        // 发送消息者
	Receiver     string `json:"receiver"`      // 接收消息者 如果是房间消息则为房间id
	SenderDevcie string `json:"sender_device"` // 发送者的设备id
	SendTime     int64  `json:"sendtime"`      // 发送时间
	SenderName   string `json:"sender_name"`   // 发送者名字
	Content      string `json:"content"`       // 发送内容
	IsTrumpet    bool   `json:"istrumpet"`     // 是否喇叭消息
}

// LogMarshal 将日志结构体Marshal为Json。
//
// Return-[]byte:Json对应的字节数组。Return-error：Marshal时的异常信息。
func (pm *ChatPlatFormInfo) LogMarshal() ([]byte, error) {
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

type TranslateInfo struct {
	Gid              uint   `json:"gid"`               // 大区的ID编号
	LogicId          string `json:"logicic"`           // logic server id
	LogTime          int64  `json:"log_time"`          // Log的生成时间戳
	Time             string `json:"@timestamp"`        // 数据记录的当地时间
	TimeUTC8         string `json:"cn_time"`           // 数据记录的北京时间
	Type             string `json:"type"`              // 记录类型
	Sender           string `json:"sender"`            // 发送消息者
	TargetLanguage   string `json:"target_language"`   // 目标翻译语言
	Content          string `json:"content"`           // 发送内容
	TranslateContent string `json:"translate_content"` // 翻译内容
	TranslateNum     int64  `json:"translate_num"`     // 该语言API翻译的第几条(0没走谷歌)
}

func (pm *TranslateInfo) LogMarshal() ([]byte, error) {
	nt := time.Now()
	utc := nt.UTC().Add(8 * time.Hour)
	pm.LogTime = nt.UnixNano()
	pm.Time = nt.Format("2006-01-02T15:04:05.000Z07:00")
	pm.TimeUTC8 = utc.Format("2006-01-02 15:04:05")
	pm.Type = Translate
	data, err := json.Marshal(*pm)
	if err != nil {
		return nil, err
	}
	data = append(data, "\n"...)
	return data, nil
}
