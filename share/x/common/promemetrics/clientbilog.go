package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	ClientBIRequestsNumName = "request_total"
)

var (
	clientBIRequestsNumCounter prometheus.Counter // 用以记录BI服务器累计收到的客户端BI请求数目。
)

// 用于向PushGateWay推送时的JobName。
var ClientBILog string

// GetClientBIJobName 获取BI服在PushGateWay和Prome展示的Job昵称
//
// Param-gid:服务器的大区ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetClientBIJobName(gid uint) string {
	ClientBILog = fmt.Sprintf("%s_%d", ClientBIPrefix, gid)
	return ClientBILog
}

// InitClientBIMetrics 初始化BI埋点服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件名。Param-gid：服务器大区号。
//
// Return-error:初始化报错信息。
func InitClientBIMetrics(cfgPath string, gid uint, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetClientBIJobName(gid)); err != nil {
		return fmt.Errorf("<InitClientBIMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, "", ClientBILog) {
		return fmt.Errorf("<InitClientBIMetrics> LoadPMetricsLog err")
	}

	RequestsNumCounter, clientBIErr := pmetrics.NewCounter(pmetrics.GetCollectorName(ClientBILog, ClientBIRequestsNumName))
	if clientBIErr != nil {
		return clientBIErr
	}
	clientBIRequestsNumCounter = RequestsNumCounter

	return nil
}

// IncClientBIRequestsNumCounter 服务器收到的客户端BI请求数目+1。
func IncClientBIRequestsNumCounter() {
	if clientBIRequestsNumCounter == nil {
		return
	}
	clientBIRequestsNumCounter.Inc()
}

// GetClientBIRequestsNumCounter 获取服务器收到的客户端BI请求数目。
func GetClientBIRequestsNumCounter() int64 {
	if clientBIRequestsNumCounter == nil {
		return pmetrics.CannotGetValue
	}
	return pmetrics.GetMetricValueByType(clientBIRequestsNumCounter, pmetrics.Counter)
}
