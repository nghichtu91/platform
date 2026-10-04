package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	StressxSendReqName    = "send_req"
	StressxSendFailName   = "send_fail"
	StressxCreateSlowName = "create_slow"
	StressxSendPingName   = "send_ping"
	StressxReceiveResName = "receive_res"
)

var (
	stressxSendReqCounter    prometheus.Counter //机器人发送的请求数量
	stressxSendFailCounter   prometheus.Counter //机器人发送请求失败的数量
	stressxCreateSlowCounter prometheus.Counter //机器人创建缓慢的数量
	stressxSendPingCounter   prometheus.Counter //机器人发送的ping包数量
	stressxReceiveResCounter prometheus.Counter //机器人收到的响应数量
)

// 用于向PushGateWay推送时的JobName。
var StressxJobName string

// GetStressxJobName 获取Stressx服在PushGateWay和Prome展示的Job昵称.
//
// Param-gid:服务器的大区ID。 Param-stressId：服务器ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetStressxJobName(gid uint, stressId string) string {
	StressxJobName = fmt.Sprintf("%s_%d_%s", StressxPrefix, gid, stressId)
	return StressxJobName
}

// InitStressxMetrics 初始化Stressx逻辑服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件。 Param-gid:大区号。 Param-stressId:服务器ID。
//
// Return-error:初始化报错信息。
func InitStressxMetrics(cfgPath string, gid uint, stressId string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetStressxJobName(gid, stressId)); err != nil {
		return fmt.Errorf("<InitStressxMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, fmt.Sprintf("%v", stressId), CometJobName) {
		return fmt.Errorf("<InitStressxMetrics> LoadPMetricsLog err")
	}

	reqCounter, err := pmetrics.NewCounter(pmetrics.GetCollectorName(StressxJobName, StressxSendReqName))
	if err != nil {
		return err
	}
	stressxSendReqCounter = reqCounter

	failCounter, err := pmetrics.NewCounter(pmetrics.GetCollectorName(StressxJobName, StressxSendFailName))
	if err != nil {
		return err
	}
	stressxSendFailCounter = failCounter

	slowCounter, err := pmetrics.NewCounter(pmetrics.GetCollectorName(StressxJobName, StressxCreateSlowName))
	if err != nil {
		return err
	}
	stressxCreateSlowCounter = slowCounter

	pingCounter, err := pmetrics.NewCounter(pmetrics.GetCollectorName(StressxJobName, StressxSendPingName))
	if err != nil {
		return err
	}
	stressxSendPingCounter = pingCounter

	resCounter, err := pmetrics.NewCounter(pmetrics.GetCollectorName(StressxJobName, StressxReceiveResName))
	if err != nil {
		return err
	}
	stressxReceiveResCounter = resCounter

	return nil
}

// IncSendReq stress服务器发送的请求数目+1。
func IncStressxSendReq() {
	if stressxSendReqCounter == nil {
		return
	}
	stressxSendReqCounter.Inc()
}

// IncSendFail stress服务器发送失败的请求数目+1。
func IncStressxSendFail() {
	if stressxSendFailCounter == nil {
		return
	}
	stressxSendFailCounter.Inc()
}

// IncCreateSlow stress服务器机器人创建缓慢的数量+1。
func IncStressxCreateSlow() {
	if stressxCreateSlowCounter == nil {
		return
	}
	stressxCreateSlowCounter.Inc()
}

// IncSendPing stress服务器发送的ping数目+1。
func IncStressxSendPing() {
	if stressxSendPingCounter == nil {
		return
	}
	stressxSendPingCounter.Inc()
}

// IncReceiveRes stress服务器收到的响应数目+1。
func IncStressxReceiveRes() {
	if stressxReceiveResCounter == nil {
		return
	}
	stressxReceiveResCounter.Inc()
}
