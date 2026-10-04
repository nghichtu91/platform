package allmetrics

import (
	"fmt"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	//机器人发送的请求数量
	_sendReq gm.Counter
	//机器人发送请求失败的数量
	_sendFail gm.Counter
	//机器人创建缓慢的数量
	_createSlow gm.Counter
	//机器人发送的ping包数量
	_sendPing gm.Counter
	//机器人收到的响应数量
	_receiveRes gm.Counter
)

func PrefixStressxMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", StressxPrefix, gid, serverId)
}

func InitStressxMetrics() {
	_sendReq = metrics.NewCustomCounter("sendreq")
	_sendFail = metrics.NewCustomCounter("sendfail")
	_createSlow = metrics.NewCustomCounter("createslow")
	_sendPing = metrics.NewCustomCounter("sendping")
	_receiveRes = metrics.NewCustomCounter("receiveres")
}

func IncStressxSendReq() {
	if _sendReq != nil {
		_sendReq.Inc(1)
	}
}

func IncStressxSendFail() {
	if _sendFail != nil {
		_sendFail.Inc(1)
	}
}

func IncStressxCreateSlow() {
	if _createSlow != nil {
		_createSlow.Inc(1)
	}
}

func IncStressxSendPing() {
	if _sendPing != nil {
		_sendPing.Inc(1)
	}
}

func IncStressxReceiveRes() {
	if _receiveRes != nil {
		_receiveRes.Inc(1)
	}
}
