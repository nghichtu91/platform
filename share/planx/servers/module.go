package servers

import (
	"sync/atomic"

	"github.com/nghichtu91/platform/share/planx/util"
)

type Module interface {
	Start(w *util.WaitGroupWrapper) error
	AfterStart()
	BeforeStop()
	Stop()
}

// HotModuleComponent 此组件实现HotModule除Module之外的额外接口
// 目前主要用于cross member动态开关module
// gamex或其他服务暂时不需要此组件
type HotModuleComponent struct {
	isEnd int32

	// 模块结束时需要执行的函数
	// 任意函数出现错误时，立刻返回，不执行后续的函数
	endFns []func() error
}

// AddEndFn 添加函数并在End时执行
// 任意函数出现错误时，立刻返回，不执行后续的函数
func (hc *HotModuleComponent) AddEndFn(fn ...func() error) {
	hc.endFns = append(hc.endFns, fn...)
}

// End 模块结束时执行的特殊操作
func (hc *HotModuleComponent) End() error {
	atomic.StoreInt32(&hc.isEnd, 1)

	if len(hc.endFns) > 0 {
		for _, fn := range hc.endFns {
			if err := fn(); err != nil {
				return err
			}
		}
	}

	return nil
}

// IsEnd 判断模块是否已处于End阶段
func (hc *HotModuleComponent) IsEnd() bool {
	return atomic.LoadInt32(&hc.isEnd) == 1
}

// HotModule 支持热加热减的Module
type HotModule interface {
	Module

	// AddEndFn 添加函数并在End时执行
	AddEndFn(...func() error)

	// IsEnd 判断模块是否已处于End阶段
	IsEnd() bool

	// End 模块结束时执行的特殊操作
	End() error
}
