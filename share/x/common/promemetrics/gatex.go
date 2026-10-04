package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	GatexCCUName          = "ccu"
	GatexHandShakeCCUName = "hand_shake_ccu"
	GatexRequstName       = "request_total"
	GateSessionName       = "session_size"
	GatexSendName         = "send_total"
)

var (
	gatexCCUGauge          prometheus.Gauge   // 用以记录尝试进行handshake的玩家，此值写入ETCD，用于Gatex服务器的负载均衡。
	gatexHandShakeCCUGauge prometheus.Gauge   // 用以记录handshake通过的玩家。
	gatexSessionGuage      prometheus.Gauge   // 用以记录Gatex最近一次收到的SessionMap的大小len(Map[accountId][sessionInfo])。
	gatexRequstCounter     prometheus.Counter // 用以记录收到的请求数目。
	gatexSendCounter       prometheus.Counter // 用以记录发送的响应数目。
)

// 用于向PushGateWay推送时的JobName。
var GateJobName string

// GetGatexJobName 获取Gatex服在PushGateWay和Prome展示的Job昵称
//
// Param-gid:服务器的大区ID。 Param-serverid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetGatexJobName(gid uint, serverId string) string {
	GateJobName = fmt.Sprintf("%s_%d_%s", GatePrefix, gid, serverId)
	return GateJobName
}

// InitGatexMetrics 初始化Gatex网关服务器关于Metrics的Collector与Statistics。
func InitGatexMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetGatexJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitGatexMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, sid, GateJobName) {
		return fmt.Errorf("<InitGatexMetrics> LoadPMetricsLog err")
	}

	ccuGauge, ccuErr := pmetrics.NewGauge(pmetrics.GetCollectorName(GateJobName, GatexCCUName))
	if ccuErr != nil {
		return ccuErr
	}
	gatexCCUGauge = ccuGauge

	handShakeCCUGauge, handShakeCCUErr := pmetrics.NewGauge(pmetrics.GetCollectorName(GateJobName, GatexHandShakeCCUName))
	if handShakeCCUErr != nil {
		return handShakeCCUErr
	}
	gatexHandShakeCCUGauge = handShakeCCUGauge

	SessionGuage, sessionGuageErr := pmetrics.NewGauge(pmetrics.GetCollectorName(GateJobName, GateSessionName))
	if sessionGuageErr != nil {
		return sessionGuageErr
	}
	gatexSessionGuage = SessionGuage

	requstCounter, requstErr := pmetrics.NewCounter(pmetrics.GetCollectorName(GateJobName, GatexRequstName))
	if requstErr != nil {
		return requstErr
	}
	gatexRequstCounter = requstCounter

	sendCounter, sendErr := pmetrics.NewCounter(pmetrics.GetCollectorName(GateJobName, GatexSendName))
	if sendErr != nil {
		return sendErr
	}
	gatexSendCounter = sendCounter

	return nil
}

// IncGatexCCUGauge Gatex服务器当前CCU值+1。
func IncGatexCCUGauge() {
	if gatexCCUGauge == nil {
		return
	}
	gatexCCUGauge.Inc()
}

// DecGatexCCUGauge Gatex服务器当前CCU值-1。
func DecGatexCCUGauge() {
	if gatexCCUGauge == nil {
		return
	}
	gatexCCUGauge.Dec()
}

// GetGatexCCUGauge 获取Gatex服务器当前CCU值。
//
// Return-int64：此值写入ETCD，用于Gatex服务器的负载均衡。
func GetGatexCCUGauge() int64 {
	if gatexCCUGauge == nil {
		return pmetrics.CannotGetValue
	}
	return pmetrics.GetMetricValueByType(gatexCCUGauge, pmetrics.Gauge)
}

// IncGatexHandShakeCCUGauge 暂不使用
func IncGatexHandShakeCCUGauge() {
	if gatexHandShakeCCUGauge == nil {
		return
	}
	gatexHandShakeCCUGauge.Inc()
}

// DecGatexHandShakeCCUGauge 暂不使用
func DecGatexHandShakeCCUGauge() {
	if gatexHandShakeCCUGauge == nil {
		return
	}
	gatexHandShakeCCUGauge.Dec()
}

// IncNewRequest Gatex服务器收到的请求数目+1。
func IncNewRequest() {
	if gatexRequstCounter == nil {
		return
	}
	gatexRequstCounter.Inc()
}

// IncNewSend Gatex服务器发送的响应数目+1。
func IncNewSend() {
	if gatexSendCounter == nil {
		return
	}
	gatexSendCounter.Inc()
}

// UpdateGatexSessionGuage 记录Gatex最近一次收到的SessionMap的大小len(Map[accountId][sessionInfo])。
//
// Param-value:len(Map[accountId][sessionInfo])。
func UpdateGatexSessionGuage(value int64) {
	if gatexSessionGuage == nil {
		return
	}
	gatexSessionGuage.Set(float64(value))
}
