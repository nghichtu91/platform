package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"
)

var CrossJobName string

// GetCrossxJobName 获取Crossx服在PushGateWay和Prome展示的Job昵称
//
// Param-gid:服务器的大区ID。 Param-serverid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetCrossxJobName(gid uint, serverId string) string {
	CrossJobName = fmt.Sprintf("%s_%d_%s", CrossxPrefix, gid, serverId)
	return CrossJobName
}

// InitAuthMetrics 初始化Auth登录服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件名。 Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-error:初始化时的报错信息。
func InitCrossxMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetCrossxJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitCrossxMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, sid, CrossJobName) {
		return fmt.Errorf("<InitCrossxMetrics> LoadPMetricsLog err")
	}

	return nil
}
