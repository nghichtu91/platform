package giftcode

import "github.com/nghichtu91/platform/share/planx/servers"

var instance *Module // GiftCode处理模块实例。

// GenModule 生成模块实例。
func GenModule() servers.Module {
	instance = createGiftCodeModule()
	return instance
}

// GetModule 获取模块实例。
func GetModule() *Module {
	return instance
}
