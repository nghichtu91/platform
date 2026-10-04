package promemetrics

import (
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"
)

const (
	StatisticsTime  = "time"
	StatisticsCount = "count"
	StatisticsSize  = "size"
)

// 处理时间统计（同graphite实现的TimeStatistics）
//
// Des:记录一个请求从收到到处理完成的处理时间。
//
// Param-name:消息名或与其它信息拼接后的名字。Param-beforeTimeNano:收到请求的时间。
func TimeStatistics(name string, beforeTimeNano int64) {
	if pmetricslog.PMetricsLog == nil {
		return
	}

	delayTime := time.Now().UnixNano() - beforeTimeNano
	pmetricslog.LogPMetrics(&pmetricslog.PMetricsLogInfo{
		Gid:   pmetricslog.Gid,
		Sid:   pmetricslog.ServerID,
		Type:  StatisticsTime,
		Value: delayTime,
		Name:  name,
	})
}

// 处理数量统计（同graphite实现的CountStatistics）
//
// Param-name:消息名或与其它信息拼接后的名字。
func CountStatistics(name string) {
	if pmetricslog.PMetricsLog == nil {
		return
	}

	pmetricslog.LogPMetrics(&pmetricslog.PMetricsLogInfo{
		Gid:   pmetricslog.Gid,
		Sid:   pmetricslog.ServerID,
		Type:  StatisticsCount,
		Value: int64(1),
		Name:  name,
	})
}

// 处理大小统计（同graphite实现的SizeStatistics）
//
// Param-name:消息名或与其它信息拼接后的名字。Param-size:本次统计的字节大小。
//
// Example-name：requests.[Gid].[Sid].[MessageID（"/"被替换为"."）]
func SizeStatistics(name string, size int64) {
	if pmetricslog.PMetricsLog == nil {
		return
	}

	pmetricslog.LogPMetrics(&pmetricslog.PMetricsLogInfo{
		Gid:   pmetricslog.Gid,
		Sid:   pmetricslog.ServerID,
		Type:  StatisticsSize,
		Value: size,
		Name:  name,
	})
}
