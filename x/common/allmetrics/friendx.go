package allmetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/metrics"

	gm "github.com/rcrowley/go-metrics"
)

var reqCounter gm.Counter

func PrefixFriendXMetrics(gID uint, serverID string) string {
	return fmt.Sprintf("%s.%d.%s", FriendXPrefix, gID, serverID)
}

func InitFriendXMetrics() {
	newReqCounter := func(name string) gm.Counter {
		return metrics.NewCustomCounter(name)
	}
	reqCounter = newReqCounter("req")
}

func ADDFriendXReqCount() {
	reqCounter.Inc(1)
}
