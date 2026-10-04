package controllers

import (
	"encoding/base64"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
)

var (
	logInitOnce sync.Once
)

func InitLog() {
	logInitOnce.Do(func() {
		zaplog.InitZapLog("", nil, "")
	})
}

func TestSdkLoginInfo_Decode(t *testing.T) {
	InitLog()

	li := &SdkLoginInfo{
		Token:         "ooxx",
		ChannelId:     "1122",
		AppChannelId:  "123",
		Device:        "fakeDevice",
		Uid:           "0003332",
		PlatformType:  "ios",
		DeviceId:      "fakeDeviceID",
		IP:            "127.0.0.1",
		AddressFlag:   "231",
		SdkChannelUid: "332232",
	}

	// 这里生成SdkLoginInfo对应的base64编码
	di := &SdkDebugLoginInfo{
		Enc: new(SdkLoginInfo),
		Dec: li,
	}

	di.Encode()
	t.Logf("enc params: %s", di.Enc.GenParams(nil))

	assert.True(t, di.Enc.Check())
	assert.Nil(t, di.Enc.Decode())
}

func TestEncode(t *testing.T) {
	t.Logf("%v", base64.URLEncoding.EncodeToString([]byte("jy00001")))
	t.Logf("%v", secure.DefaultEncode.Encode64ForNet([]byte("1")))
}
