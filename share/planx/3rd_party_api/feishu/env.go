package feishu

import (
	"strconv"
	"time"
)

var (
	secretKey string

	sender *Sender
)

// InitFeishu 初始化飞书环境变量
func InitFeishu(key string) {
	secretKey = key

	data = []byte((strconv.Itoa(int(time.Now().Unix())) + "\n" + secretKey))

	sender = &Sender{
		recChan: make(chan Req, 2048),
		quit:    make(chan struct{}, 1),
	}

	go sender.start()
}

func Stop() {
	if sender != nil {
		sender.stop()
	}
}
