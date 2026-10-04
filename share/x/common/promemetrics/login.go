package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"
)

var LoginJobName string

// GetLoginJobName 获取Login服在PushGateWay和Prome展示的Job昵称
//
// Param-gid:服务器的大区ID。 Param-serverid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetLoginJobName(gid uint, serverId string) string {
	LoginJobName = fmt.Sprintf("%s_%d_%s", LoginPrefix, gid, serverId)
	return LoginJobName
}

// InitLoginMetrics 初始化Login服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件名。 Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-error:初始化时的报错信息。
func InitLoginMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetLoginJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitLoginMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, sid, LoginJobName) {
		return fmt.Errorf("<InitLoginMetrics> LoadPMetricsLog err")
	}

	return nil
}
