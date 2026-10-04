package allmetrics

import (
	"fmt"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var noticeReq_C gm.Counter

func PrefixNoticeMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", NoticePrefix, gid, serverId)
}

func InitNoticeMetrics() {
	NewAuthCounter := func(name string) gm.Counter {
		return metrics.NewCustomCounter(name)
	}
	noticeReq_C = NewAuthCounter("req")
}

func AddNoticeReqCount() {
	noticeReq_C.Inc(1)
}
