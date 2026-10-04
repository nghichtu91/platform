package allmetrics

import (
	"fmt"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	//连接的玩家
	comet_ccu gm.Counter
	//Request per second
	comet_rps gm.Meter
	//The number of Request单调增长
	comet_nRequests gm.Counter
	comet_rsp       gm.Meter
	//The number of Request单调增长
	comet_nResps gm.Counter
)

func PrefixCometxMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", CometPrefix, gid, serverId)
}

func InitCometxMetrics() {
	comet_ccu = metrics.NewCustomCounter("ccu")
	comet_rps = metrics.NewCustomMeter("requests")
	comet_nRequests = metrics.NewCustomCounter("requesttotal")
	comet_rsp = metrics.NewCustomMeter("response")
	comet_nResps = metrics.NewCustomCounter("responsetotal")
}

func CometCountAddNewRequest(n int64) {
	if comet_rps != nil {
		comet_rps.Mark(n)
	}
	if comet_nRequests != nil {
		comet_nRequests.Inc(n)
	}
}

func CometCountAddNewResponse(n int64) {
	if comet_rsp != nil {
		comet_rsp.Mark(n)
	}
	if comet_nResps != nil {
		comet_nResps.Inc(n)
	}
}
func AddCometxCCU() {
	if comet_ccu != nil {
		comet_ccu.Inc(1)
	}
}

func ReduceCometxCCU() {
	if comet_ccu != nil {
		comet_ccu.Dec(1)
	}
}

func GetCometxCCUCount() int64 {
	if comet_ccu != nil {
		return comet_ccu.Count()
	}
	return 0
}
