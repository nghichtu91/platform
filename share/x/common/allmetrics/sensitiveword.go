package allmetrics

import (
	"fmt"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	// Request per second
	sensitiveWordRps gm.Meter
	// The number of Request单调增长
	sensitiveWordRequests gm.Counter
)

func PrefixSensitiveWordMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", SensitiveWordPrefix, gid, serverId)
}

func InitSensitiveWordMetrics() {
	sensitiveWordRps = metrics.NewCustomMeter("requests")
	sensitiveWordRequests = metrics.NewCustomCounter("requesttotal")
}

func SensitiveWordCountAddNewRequest(n int64) {
	if sensitiveWordRps != nil {
		sensitiveWordRps.Mark(n)
	}
	if sensitiveWordRequests != nil {
		sensitiveWordRequests.Inc(n)
	}
}
