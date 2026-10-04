package allmetrics

import (
	"fmt"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	//尝试连接进行handshake的玩家
	_ccu gm.Counter
	//合法handshake通过的玩家
	_handShakeCCU gm.Counter
	//Request per second
	_rps gm.Meter
	//The number of Request单调增长
	_nRequests gm.Counter
	//The number of Request单调增长
	_nSends gm.Counter
)

func PrefixGatexMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", GatePrefix, gid, serverId)
}

func InitGatexMetrics() {
	_ccu = metrics.NewCustomCounter("ccu")
	_rps = metrics.NewCustomMeter("requests")
	_nRequests = metrics.NewCustomCounter("requesttotal")
	_nSends = metrics.NewCustomCounter("sendtotal")
}

func AddCCU() {
	if _ccu != nil {
		_ccu.Inc(1)
	}
}

func ReduceCCU() {
	if _ccu != nil {
		_ccu.Dec(1)
	}
}

func GetCCUCount() int64 {
	if _ccu != nil {
		return _ccu.Count()
	}
	return 0
}

func addHandShakeCCU() {
	if _handShakeCCU != nil {
		_handShakeCCU.Inc(1)
	}
}

func reduceHandShakeCCU() {
	if _handShakeCCU != nil {
		_handShakeCCU.Dec(1)
	}
}

func CountAddNewRequest(n int64) {
	if _rps != nil {
		_rps.Mark(n)
	}
	if _nRequests != nil {
		_nRequests.Inc(n)
	}
}

func CountAddNewSend() {
	if _nSends != nil {
		_nSends.Inc(1)
	}
}
