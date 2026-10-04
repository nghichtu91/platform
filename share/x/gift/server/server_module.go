package server

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/servers"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

var (
	moduleMng          *ModuleMng
	server_gen_modules map[string]GenServerModuleFunc
)

// 模块管理器。
type ModuleMng struct {
	modules map[string]servers.Module // 模块管理器所管理的模块（无顺序启动、关闭）。
}

// 模块生成器。
type GenServerModuleFunc func() servers.Module

// StartModuleMng 开启模块管理器。
func StartModuleMng(w *util.WaitGroupWrapper) error {
	// 初始化模块管理器。
	moduleMng = &ModuleMng{
		modules: make(map[string]servers.Module, 32),
	}

	// 初始化并启动模块（无顺序启动）。
	for name, moduleGen := range server_gen_modules {
		module := moduleGen()
		if module == nil {
			return fmt.Errorf("StartModuleMng module gen is nil: %s", name)
		}

		if err := module.Start(w); err != nil {
			return fmt.Errorf("StartModuleMng module start err, module: %v, err: %v", name, err)
		}

		module.AfterStart()

		moduleMng.modules[name] = module
	}

	return nil
}

// StopModuleMng 关闭模块管理器。
func StopModuleMng() {
	for name, module := range moduleMng.modules {
		func() {
			// 防止一个模块stop panic导致其余模块无法正常停止。
			defer tilogs.PanicCatcher("StopModule module stop panic: %v", name)
			module.BeforeStop()
			module.Stop()
			tilogs.L().Infof("StopModule module stop success: %v", name)
		}()
	}
}

// regModuleGen 注册模块生成器。
//
// Note: 如果重复name注册，则给出警告，后一次注册生效。
//
// Param-name: 模块名字; Param-moduleGen: 模块生成器。
func regModuleGen(name string, moduleGen GenServerModuleFunc) {
	if server_gen_modules == nil {
		server_gen_modules = make(map[string]GenServerModuleFunc, 32)
	}

	if _, ok := server_gen_modules[name]; ok {
		tilogs.L().Warnf("regModuleGen module repeat（Effect of post registration）: %s", name)
	}

	server_gen_modules[name] = moduleGen
}
