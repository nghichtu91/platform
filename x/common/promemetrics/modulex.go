package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"
)

var ModulexJobName string

// GetModulexJobName 获取Modulex服在PushGateWay和Prome展示的Job昵称
//
// Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetModulexJobName(gid, sid uint) string {
	ModulexJobName = fmt.Sprintf("%s_%d_%d", ModulexPrefix, gid, sid)
	return ModulexJobName
}

// InitModulexMetrics 初始化Modulex服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件名。 Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-error:初始化时的报错信息。
func InitModulexMetrics(cfgPath string, gid uint, sid string) error {
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, sid, ModulexJobName) {
		return fmt.Errorf("<InitModulexMetrics> LoadPMetricsLog err")
	}

	return nil
}
