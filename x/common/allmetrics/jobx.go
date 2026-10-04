package allmetrics

import (
	"fmt"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	//Request per second
	job_rps gm.Meter
	//The number of Request单调增长
	job_nRequests gm.Counter
)

func PrefixJobMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", JobPrefix, gid, serverId)
}

func InitJobMetrics() {
	job_rps = metrics.NewCustomMeter("requests")
	job_nRequests = metrics.NewCustomCounter("requesttotal")
}

func JobCountAddNewRequest(n int64) {
	if job_rps != nil {
		job_rps.Mark(n)
	}
	if job_nRequests != nil {
		job_nRequests.Inc(n)
	}
}
