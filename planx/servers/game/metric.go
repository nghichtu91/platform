package game

import (
	"fmt"
	"strings"

	"github.com/nghichtu91/platform/share/planx/metrics"
)

func SendMetrics(msgCode, clientMsgName string, size int, gid, shardId uint) {
	name := graphiteResponseName(msgCode, gid, shardId)
	metrics.ClientCountStatistics(name, clientMsgName, int64(size))
	metrics.SizeStatistics(name, size)
}

func SendNMetrics(msgCode, clientMsgName string, size int, n int32, gid, shardId uint) {
	name := graphiteResponseName(msgCode, gid, shardId)
	metrics.ClientNCountStatistics(name, clientMsgName, n, int64(size))
	metrics.SizeStatistics(name, size)
}

func graphiteResponseName(code string, gid, shardId uint) string {
	lr := strings.Replace(code, "/", ".", -1)
	llr := strings.ToLower(lr) + "resp"
	return fmt.Sprintf("requests.%d.%d.%s", gid, shardId, llr)
}
