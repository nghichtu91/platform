package feishu

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/nghichtu91/platform/share/planx/timeutil"
)

func TestSendFeishu(t *testing.T) {
	secret := "5XmM1vTD7YNUI3wg7Tdouc"
	// 飞书测试
	url := "https://open.feishu.cn/open-apis/bot/v2/hook/9003ff1d-58e4-4bf2-b4fb-a5db6a633794"
	ticker := timeutil.TimerSec.After(time.Second)

	InitFeishu(secret)

	assert.Nil(t, SendFeishu(url, "QA环境调试飞书测试", false))

	<-ticker
	ticker = timeutil.TimerSec.After(time.Second)

	Stop()

	<-ticker
	ticker = timeutil.TimerSec.After(time.Second)

	// 虽然关了，但是没错误
	assert.Nil(t, SendFeishu(url, "QA环境调试飞书测试", false))

	<-ticker
}
