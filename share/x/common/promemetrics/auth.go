package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	AuthRegisterCounterName = "register" // 用以记录玩家UUID创建数量的Counter名称。
	AuthCCUGaugeName        = "ccu"      // 用以记录Auth服CCU的Counter名称。
)

// 用于向PushGateWay推送时的JobName。
var AuthJobName string

var (
	authRegisterCounter prometheus.Counter // 用以记录玩家UUID创建数量的Counter。
	authCCUGauge        prometheus.Gauge   // 用以记录Auth服CCU（Concurren User同时在线人数）的Gauge。
)

// GetAuthJobName 获取Auth服在PushGateWay和Prome展示的Job昵称
//
// Param-gid:服务器的大区ID。 Param-serverid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetAuthJobName(gid uint, serverId string) string {
	AuthJobName = fmt.Sprintf("%s_%d_%s", AuthPrefix, gid, serverId)
	return AuthJobName
}

// GetAuthDBStatPrefix 获取Auth服DB相关的前缀
func GetAuthDBStatPrefix(key, oper string, authGid uint) string {
	return fmt.Sprintf("%s_%d_%s_%s", AuthPrefix, authGid, key, oper)
}

// InitAuthMetrics 初始化Auth登录服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件名。 Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-error:初始化时的报错信息。
func InitAuthMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetAuthJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitAuthMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, sid, AuthJobName) {
		return fmt.Errorf("<InitAuthMetrics> LoadPMetricsLog err")
	}

	// 用以记录玩家UUID创建数量的Counter。
	registerCounter, registerErr := pmetrics.NewCounter(pmetrics.GetCollectorName(AuthJobName, AuthRegisterCounterName))
	if registerErr != nil {
		return registerErr
	}
	authRegisterCounter = registerCounter

	// 用以记录Auth服CCU的Counter。
	ccuGauge, ccuErr := pmetrics.NewGauge(pmetrics.GetCollectorName(AuthJobName, AuthCCUGaugeName))
	if ccuErr != nil {
		return ccuErr
	}
	authCCUGauge = ccuGauge

	return nil
}

// IncAuthRegisterCounter 记录玩家UUID创建数量的Counter+1。
func IncAuthRegisterCounter() {
	if authRegisterCounter == nil {
		return
	}
	authRegisterCounter.Inc()
}

// IncAuthCCUGauge 记录Auth服CCU的Counter+1（当前记录的为累计访问Auth服的数量）。
func IncAuthCCUGauge() {
	if authCCUGauge == nil {
		return
	}
	authCCUGauge.Inc()
}
