package giftcode

import (
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/gift/config"
	"github.com/nghichtu91/platform/share/x/gift/modules/common"
)

const (
	ModuleGiftCode       = "gift_code"            // GiftCode请求受理模块。
	CommonSubModule      = "commonSubModule"      // 通用受理子模块标识（日志输出）。
	IndependentSubModule = "independentSubModule" // 高耗时独立处理受理子模块。
	GenCodesSubModule    = "genCodesSubModule"    // 生成兑换码受理子模块。
)

type Module struct {
	commonSubModule      *common.CommonSubModule // 通用受理子模块。
	independentSubModule *common.CommonSubModule // 独立受理子模块。
	genCodesSubModule    *common.CommonSubModule // 生成受理子模块。
}

// createGiftCodeModule 创建GiftCode模块实例。
func createGiftCodeModule() *Module {
	return &Module{
		commonSubModule:      common.CreateCommonSubModule(config.DefaultReqHandlerNum, config.DefaultChanBufferNum, ModuleGiftCode, CommonSubModule),
		independentSubModule: common.CreateCommonSubModule(config.DefaultReqHandlerNum, config.DefaultChanBufferNum, ModuleGiftCode, IndependentSubModule),
		//genCodesSubModule:    common.CreateCommonSubModule(config.DefaultReqHandlerSingleNum, config.DefaultChanBufferSmallNum, ModuleGiftCode, GenCodesSubModule),
	}
}

// TODO 子模块也可以抽象出接口或者只有CommonSub的列表，依据标识获取，不用每个单独写函数。
// TODO 这里子模块较少，暂时不做抽象, 抽象时可以加入modules的common中。
// GetCommonSubModule 获取通用受理子模块。
func (mo *Module) GetCommonSubModule() *common.CommonSubModule {
	return mo.commonSubModule
}

func (mo *Module) GetIndependentSubModule() *common.CommonSubModule {
	return mo.independentSubModule
}

func (mo *Module) GetGenCodesSubModule() *common.CommonSubModule {
	return mo.genCodesSubModule
}

// Start 启动模块 == 启动所有子模块。
func (mo *Module) Start(w *util.WaitGroupWrapper) error {
	// 开启通用受理子模块。
	mo.commonSubModule.Start(w)
	// 开启独立处理受理子模块。
	mo.independentSubModule.Start(w)
	// 开启生成兑换码受理子模块。
	//mo.genCodesSubModule.Start(w)

	return nil
}

// AfterStart
func (mo *Module) AfterStart() {

}

// BeforeStop
func (mo *Module) BeforeStop() {

}

// Stop 终止模块运行 == 终止所有子模块运行。
func (mo *Module) Stop() {
	mo.commonSubModule.Stop()
	mo.independentSubModule.Stop()
	//mo.genCodesSubModule.Stop()
}
