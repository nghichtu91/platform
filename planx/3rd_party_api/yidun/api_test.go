package yidun

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

// 由于tc会自动跑单元测试，用完需要把这里注释掉，避免易盾后台产生脏数据
func TestCheck(t *testing.T) {
	zaplog.InitZapLog("", nil, "")

	InitYiDun("224af423e0a84d37ee459cc1bfdfac89",
		"ef6a1dd6d17379d48d2908e6f093260b",
		"81f815ec14c789f1571b0a4f3ca5f5e5",
		"v5.1", "http://as.dun.163.com/v5/text/check")

	t.Run("Check", func(t *testing.T) {
		result, labels, err := Check(&pb.TextValidateReq{
			Text:     proto.String("没问题的普通聊天"),
			Acid:     proto.String("16:1600001:3344-556677-8899999999-0000"),
			Nickname: proto.String("测试玩家"),
			Receiver: proto.String("16:1600001:2343-346677-88941299999-2300"),
			DeviceOS: proto.String("超级无敌至尊PC"),
			DeviceID: proto.String("deviceIDWithLowerCase"),
			Ip:       proto.String("202.112.202.112"),
			NtID:     proto.String("11223344556677889900"),
		}, nil)

		assert.Nil(t, err)
		assert.Equal(t, RetPass, result)
		if len(labels) > 0 {
			for _, l := range labels {
				t.Logf("success labels %v", l)
			}
		}

		result, labels, err = Check(&pb.TextValidateReq{
			Text:     proto.String("648元VIP15，加微信23432112"),
			Acid:     proto.String("16:1600001:3344-556677-8899999999-0000"),
			Nickname: proto.String("测试玩家"),
			Receiver: proto.String("16:1600001:2343-346677-88941299999-2300"),
			DeviceOS: proto.String("超级无敌至尊PC2号"),
			DeviceID: proto.String("deviceID2WithLowerCase"),
			Ip:       proto.String("222.202.222.112"),
			NtID:     proto.String("68923042342309"),
		}, nil)

		assert.Nil(t, err)
		assert.Equal(t, RetFail, result)
		if len(labels) > 0 {
			for _, l := range labels {
				t.Logf("fail labels %v", l)
			}
		}
	})

	t.Run("BlackList", func(t *testing.T) {
		code, err := BlackList(0, "16:1600001:3344-556677-8899999999-0000", "12:1200001:3344-556677-8899999999-0000")
		assert.Nil(t, err)
		assert.Equal(t, 200, code)

		code, err = BlackList(1659024000, "15:1500001:3344-556677-8899999999-0000", "18:1800001:3344-556677-8899999999-0000")
		assert.Nil(t, err)
		assert.Equal(t, 200, code)
	})
}
