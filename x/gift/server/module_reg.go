package server

import (
	"github.com/nghichtu91/platform/share/x/gift/modules/giftcode"
)

func init() {
	regModuleGen(giftcode.ModuleGiftCode, giftcode.GenModule) // 注册GiftCode模块生成器。
}
