package allmetrics

import (
	"fmt"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var payXRequestCounter gm.Counter

func PrefixPayxMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", PayxPrefix, gid, serverId)
}

func InitPayxMetrics() {
	NewPayCounter := func(name string) gm.Counter {
		return metrics.NewCustomCounter(name)
	}
	payXRequestCounter = NewPayCounter("req")
}

func AddPayxReqCount() {
	payXRequestCounter.Inc(1)
}
