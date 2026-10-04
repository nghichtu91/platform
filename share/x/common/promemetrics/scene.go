package promemetrics

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var (
	sceneLineGuageMap map[string]prometheus.Gauge
	sceneLock         sync.RWMutex
)

// 用于向PushGateWay推送时的JobName。
var SceneJobName string

// GetSceneJobName 获取Scene场景服在PushGateWay和Prome展示的Job昵称
//
// Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetSceneJobName(gid uint, sid string) string {
	SceneJobName = fmt.Sprintf("%s_%d_%v", ScenePrefix, gid, sid)
	return SceneJobName
}

// InitSceneMetricsMap 初始化Scene场景服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件名。 Param-gid:大区号。
//
// Param-sid：区服号。
func InitSceneMetricsMap(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetSceneJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitSceneMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, sid, SceneJobName) {
		return fmt.Errorf("<InitSceneMetrics> LoadPMetricsLog err")
	}

	sceneLineGuageMap = make(map[string]prometheus.Gauge, 4)

	return nil
}

// InitOneSceneMetric 初始化一个新场景线程的指标。
//
// Param-key：新场景线程的ID编号。
func InitOneSceneMetric(key string) {
	contents := strings.Split(key, ":")

	sceneLock.Lock()
	defer sceneLock.Unlock()

	_, ok := sceneLineGuageMap[key]
	if ok {
		return
	}

	gauge, err := pmetrics.NewGauge("name_" + strings.Join(contents, "_"))
	if err != nil {
		tilogs.L().Errorf("<Promethues> scene new gauge err : %v", err)
		return
	}

	sceneLineGuageMap[key] = gauge
}

// UpdateSceneMetrics 更新一个场景线程的当前角色数目。
//
// Param-key：场景线程ID。Param-count:场景线程数目。
func UpdateSceneMetrics(key string, count int64) {
	sceneLock.Lock()
	defer sceneLock.Unlock()

	metric, ok := sceneLineGuageMap[key]
	if !ok {
		return
	}

	metric.Set(float64(count))
}
