package promemetrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/nghichtu91/platform/share/planx/pmetrics"
	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"
)

const (
	PayXRequestName = "request_total"
)

var (
	payXRequestCounter prometheus.Counter // 用以记录收到的请求数目。
)

// 用于向PushGateWay推送时的JobName。
var PayXJobName string

// GetPayxJobName 获取Payx服在PushGateWay和Prome展示的Job昵称.
//
// Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetPayxJobName(gid uint, sid string) string {
	PayXJobName = fmt.Sprintf("%s_%d_%s", PayxPrefix, gid, sid)
	return PayXJobName
}

// TODO GetPayxDBStatPrefix

// InitPayxMetrics 初始化Payx支付服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件。 Param-gid:大区号。 Param-sid:区服号。
//
// Return-error:初始化报错信息。
func InitPayxMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetPayxJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitPayxMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, fmt.Sprintf("%v", sid), PayXJobName) {
		return fmt.Errorf("<InitPayxMetrics> LoadPMetricsLog err")
	}

	requstCounter, requstErr := pmetrics.NewCounter(pmetrics.GetCollectorName(PayXJobName, PayXRequestName))
	if requstErr != nil {
		return requstErr
	}
	payXRequestCounter = requstCounter
	return nil
}

// IncPayxRequest Payx服务器收到的请求数目+1。
func IncPayxRequest() {
	if payXRequestCounter == nil {
		return
	}
	payXRequestCounter.Inc()
}
