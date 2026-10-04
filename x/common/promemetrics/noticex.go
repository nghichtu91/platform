package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	NoticexRequestName = "req"
)

var (
	noticexRequestCounter prometheus.Counter // 用以记录公告服收到请求数目的Counter。
)

// 用于向PushGateWay推送时的JobName。
var NoticeJobName string

// GetNoticeJobName 获取Notice服在PushGateWay和Prome展示的Job昵称.
//
// Param-gid:服务器的大区ID。 Param-serverid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetNoticeJobName(gid uint, serverId string) string {
	NoticeJobName = fmt.Sprintf("%s_%d_%s", NoticePrefix, gid, serverId)
	return NoticeJobName
}

// InitNoticeMetrics 初始化Noticex公告服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件名。 Param-gid:大区号。 Param-sid：区服号。
//
// Return-error:初始化时的报错信息。
func InitNoticeMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetNoticeJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitNoticeMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, sid, NoticeJobName) {
		return fmt.Errorf("<InitNoticeMetrics> LoadPMetricsLog err")
	}

	requstCounter, requstErr := pmetrics.NewGauge(pmetrics.GetCollectorName(NoticeJobName, NoticexRequestName))
	if requstErr != nil {
		return requstErr
	}
	noticexRequestCounter = requstCounter

	return nil
}

// IncNoticexRequstCounter 公告服收到请求数目+1。
func IncNoticexRequstCounter() {
	if noticexRequestCounter == nil {
		return
	}
	noticexRequestCounter.Inc()
}
