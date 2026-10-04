package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	CometxCCUName    = "ccu"
	CometxRequstName = "request_total"
)

var (
	cometxCCUGauge      prometheus.Gauge   // 用以记录Cometx或Shards(整合Modulex)服务器当前CCU的Gauge。
	cometxRequstCounter prometheus.Counter // 用以记录收到的请求数目。
)

// 用于向PushCometWay推送时的JobName。
var CometJobName string

// GetCometxJobName 获取Cometx服在PushCometWay和Prome展示的Job昵称.
//
// Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-string:向PushCometWay推送时的JobName。
func GetCometxJobName(gid uint, sid string) string {
	CometJobName = fmt.Sprintf("%s_%d_%s", CometPrefix, gid, sid)
	return CometJobName
}

// TODO GetCometxDBStatPrefix

// InitCometxMetrics 初始化Cometx\Shards逻辑服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件。 Param-gid:大区号。 Param-sid:区服号。
//
// Return-error:初始化报错信息。
func InitCometxMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetCometxJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitCometxMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, fmt.Sprintf("%v", sid), CometJobName) {
		return fmt.Errorf("<InitCometxMetrics> LoadPMetricsLog err")
	}

	ccuGauge, ccuErr := pmetrics.NewGauge(pmetrics.GetCollectorName(CometJobName, CometxCCUName))
	if ccuErr != nil {
		return ccuErr
	}
	cometxCCUGauge = ccuGauge

	requstCounter, requstErr := pmetrics.NewCounter(pmetrics.GetCollectorName(CometJobName, CometxRequstName))
	if requstErr != nil {
		return requstErr
	}
	cometxRequstCounter = requstCounter

	return nil
}

// IncCometxCCU Cometx或Shards（Cometx与Modulex整合）的CCU递增。
func IncCometxCCU() {
	if cometxCCUGauge == nil {
		return
	}
	cometxCCUGauge.Inc()
}

// DecCometxCCU Cometx或Shards（Cometx与Modulex整合）的CCU递减。
func DecCometxCCU() {
	if cometxCCUGauge == nil {
		return
	}
	cometxCCUGauge.Dec()
}

// IncNewRequest Cometx服务器收到的请求数目+1。
func IncCometxNewRequest() {
	if cometxRequstCounter == nil {
		return
	}
	cometxRequstCounter.Inc()
}
