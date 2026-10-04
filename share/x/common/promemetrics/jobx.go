package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	JobxRequstName = "request_total"
)

var (
	jobxRequstCounter prometheus.Counter // 用以记录收到的请求数目。
)

// 用于向PushGateWay推送时的JobName。
var JobxJobName string

// GetJobxJobName 获取Jobx服在PushGateWay和Prome展示的Job昵称.
//
// Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetJobxJobName(gid uint, sid string) string {
	JobxJobName = fmt.Sprintf("%s_%d_%s", JobPrefix, gid, sid)
	return JobxJobName
}

// TODO GetJobxDBStatPrefix

// InitJobxMetrics 初始化Jobx\Shards逻辑服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件。 Param-gid:大区号。 Param-sid:区服号。
//
// Return-error:初始化报错信息。
func InitJobxMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetJobxJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitJobxMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, fmt.Sprintf("%v", sid), JobxJobName) {
		return fmt.Errorf("<InitJobxMetrics> LoadPMetricsLog err")
	}

	requstCounter, requstErr := pmetrics.NewCounter(pmetrics.GetCollectorName(JobxJobName, JobxRequstName))
	if requstErr != nil {
		return requstErr
	}
	jobxRequstCounter = requstCounter
	return nil
}

// IncNewRequest Jobx服务器收到的请求数目+1。
func IncJobxNewRequest() {
	if jobxRequstCounter == nil {
		return
	}
	jobxRequstCounter.Inc()
}
