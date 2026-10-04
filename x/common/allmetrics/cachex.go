package allmetrics

import (
	"fmt"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	// Request per second
	cacheRps gm.Meter
	// The number of Request单调增长
	cacheRequests gm.Counter
)

func PrefixCachexMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", CachexPrefix, gid, serverId)
}

func InitCachexMetrics() {
	cacheRps = metrics.NewCustomMeter("requests")
	cacheRequests = metrics.NewCustomCounter("requesttotal")
}

func CachexCountAddNewRequest(n int64) {
	if cacheRps != nil {
		cacheRps.Mark(n)
	}
	if cacheRequests != nil {
		cacheRequests.Inc(n)
	}
}
