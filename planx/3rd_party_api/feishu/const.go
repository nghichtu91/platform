package feishu

import (
	"errors"
	"time"
)

const (
	timeout = 500 * time.Millisecond
)

var (
	ErrFeishuNotInit         = errors.New("feishu not init")          // 飞书未初始化
	ErrFeishuSendChannelFull = errors.New("feishu send channel full") // 发送通道满
)
