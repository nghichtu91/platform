package allmetrics

import (
	"fmt"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	_nRequests_client_bi gm.Counter
)

func PrefixClientBIMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", ClientBIPrefix, gid, serverId)
}

func InitClientBiLogMetrics() {
	_nRequests_client_bi = metrics.NewCustomCounter("requesttotal")
}

func ClientBICountAddNewRequest(n int64) {
	if _nRequests_client_bi != nil {
		_nRequests_client_bi.Inc(n)
	}
}

func ClientBIGetRequestCount() int64 {
	if _nRequests_client_bi != nil {
		return _nRequests_client_bi.Count()
	}
	return 0
}
