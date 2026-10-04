package promemetrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/nghichtu91/platform/share/planx/pmetrics"
	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"
)

const (
	GiftRequestName = "request_total"
)

var (
	giftRequestCounter prometheus.Counter // 用以记录收到的请求数目。
)

// 用于向PushGateWay推送时的JobName。
var GiftJobName string

// GetGiftJobName 获取Gift服在PushGateWay和Prome展示的Job昵称.
//
// Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetGiftJobName(gid uint, sid string) string {
	GiftJobName = fmt.Sprintf("%s_%d_%s", GiftPrefix, gid, sid)
	return GiftJobName
}

// TODO GetGiftDBStatPrefix

// InitGiftMetrics 初始化Gift礼包码服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件。 Param-gid:大区号。 Param-sid:区服号。
//
// Return-error:初始化报错信息。
func InitGiftMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetGiftJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitGiftMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, fmt.Sprintf("%v", sid), GiftJobName) {
		return fmt.Errorf("<InitGiftMetrics> LoadPMetricsLog err")
	}

	requestCounter, requestErr := pmetrics.NewCounter(pmetrics.GetCollectorName(GiftJobName, GiftRequestName))
	if requestErr != nil {
		return requestErr
	}
	giftRequestCounter = requestCounter
	return nil
}

// IncGiftRequest Gift服务器收到的请求数目+1。
func IncGiftRequest() {
	if giftRequestCounter == nil {
		return
	}
	giftRequestCounter.Inc()
}
