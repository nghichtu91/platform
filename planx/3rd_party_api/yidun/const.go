package yidun

import (
	"errors"
	"time"
)

const (
	RetPass    = iota // 通过
	RetSuspect        // 嫌疑
	RetFail           // 不通过
)

const (
	headerTypeKey   = "Content-Type"
	headerTypeValue = "application/x-www-form-urlencoded"

	signKey = "signature"

	// HttpTimeOut
	// 1秒的超时时间来源于易盾文本：
	// 文本检测接口的响应时间依赖文本的长度，根据不同长度建议适当调整接口超时时间
	// 200字以内建议200ms，200字以上建议1s
	HttpTimeOut = 3 * time.Second

	Retries = 1 // 重试1次
)

var (
	ErrYiDunNotInit       = errors.New("yiDun environment not init")
	ErrNoAcidForBlackList = errors.New("no acid for black list")
)

var (
	Results = map[int64]string{
		1: "嫌疑",
		2: "不通过",
	}
)

var (
	Labels = map[int64]string{
		100:  "色情",
		200:  "广告",
		260:  "广告法",
		300:  "暴恐",
		400:  "违禁",
		500:  "涉政",
		600:  "谩骂",
		700:  "灌水",
		900:  "其他",
		1100: "涉价值观",
	}
)
