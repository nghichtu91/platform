package allmetrics

import (
	"fmt"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	authPrefix     string
	authGid        uint
	authRegister_C gm.Counter
	authValidReq_C gm.Counter
	authAllReq_C   gm.Counter
)

func PrefixAuthMetrics(gid uint, serverId string) string {
	authPrefix = fmt.Sprintf("%s.%d.%s", AuthPrefix, gid, serverId)
	authGid = gid
	return authPrefix
}

func InitAuthMetrics() {
	NewAuthCounter := func(name string) gm.Counter {
		return metrics.NewCustomCounter(name)
	}
	authRegister_C = NewAuthCounter("register")
	authValidReq_C = NewAuthCounter("validreq")
	authAllReq_C = NewAuthCounter("allreq")
}

func GetAuthDBStatPrefix(key, oper string) string {
	return fmt.Sprintf("%s.%d.%s.%s", AuthPrefix, authGid, key, oper)
}

func AddAuthRegisterCount() {
	authRegister_C.Inc(1)
}

func AddAuthValidReqCount() {
	authValidReq_C.Inc(1)
}

func AddAuthAllReqCount() {
	authAllReq_C.Inc(1)
}
