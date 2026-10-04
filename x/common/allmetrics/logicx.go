package allmetrics

import (
	"fmt"
	"sync"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	// Request per second
	logic_rps gm.Meter
	// The number of Request单调增长
	logic_nRequests gm.Counter
	// Request per second
	translate_rps gm.Meter
	// The number of Request单调增长
	translate_nRequests gm.Counter

	_chatTypeRoomCCU     map[string]gm.Counter
	_chatTypeRoomCCULock sync.RWMutex
)

func PrefixLogicMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", LogicPrefix, gid, serverId)
}

func InitLogicMetrics() {
	logic_rps = metrics.NewCustomMeter("requests")
	logic_nRequests = metrics.NewCustomCounter("requesttotal")

	translate_rps = metrics.NewCustomMeter("translate")
	translate_nRequests = metrics.NewCustomCounter("translatetotal")

	_chatTypeRoomCCU = make(map[string]gm.Counter, 4)
}

func chatTypeRoomNumPrefix(chatType string) string {
	return fmt.Sprintf("chattyperoomnum.%s", chatType)
}

func LogicCountAddNewRequest(channel string) {
	k := chatTypeRoomNumPrefix(channel)
	_chatTypeRoomCCULock.RLock()
	v, ok := _chatTypeRoomCCU[k]
	_chatTypeRoomCCULock.RUnlock()
	if !ok {
		_chatTypeRoomCCULock.Lock()
		v, ok = _chatTypeRoomCCU[k]
		if !ok {
			v = metrics.NewCounter(k)
			_chatTypeRoomCCU[k] = v
		}
		_chatTypeRoomCCULock.Unlock()
	}
	v.Inc(1)

	if logic_rps != nil {
		logic_rps.Mark(1)
	}
	if logic_nRequests != nil {
		logic_nRequests.Inc(1)
	}
}

func TranslateCountAddNewRequest(n int64) {
	if translate_rps != nil {
		translate_rps.Mark(n)
	}
	if translate_nRequests != nil {
		translate_nRequests.Inc(n)
	}
}
