package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	GamexCCUName = "ccu"
)

var (
	gamexCCUGauge prometheus.Gauge // 用以记录Gamex或Shards(整合Modulex)服务器当前CCU的Gauge。
)

// 用于向PushGateWay推送时的JobName。
var GamexJobName string

// GetGamexJobName 获取Gamex服在PushGateWay和Prome展示的Job昵称.
//
// Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetGamexJobName(gid, sid uint) string {
	GamexJobName = fmt.Sprintf("%s_%d_%d", GamePrefix, gid, sid)
	return GamexJobName
}

// TODO GetGamexDBStatPrefix

// InitGamexMetrics 初始化Gamex\Shards逻辑服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件。 Param-gid:大区号。 Param-sid:区服号。
//
// Return-error:初始化报错信息。
func InitGamexMetrics(cfgPath string, gid uint, sid uint, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetGamexJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitGamexMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, fmt.Sprintf("%v", sid), GamexJobName) {
		return fmt.Errorf("<InitGamexMetrics> LoadPMetricsLog err")
	}

	ccuGauge, ccuErr := pmetrics.NewGauge(pmetrics.GetCollectorName(GamexJobName, GamexCCUName))
	if ccuErr != nil {
		return ccuErr
	}
	gamexCCUGauge = ccuGauge
	return nil
}

// IncGamexCCU Gamex或Shards（Gamex与Modulex整合）的CCU递增。
func IncGamexCCU() {
	if gamexCCUGauge == nil {
		return
	}
	gamexCCUGauge.Inc()
}

// DecGamexCCU Gamex或Shards（Gamex与Modulex整合）的CCU递减。
func DecGamexCCU() {
	if gamexCCUGauge == nil {
		return
	}
	gamexCCUGauge.Dec()
}
