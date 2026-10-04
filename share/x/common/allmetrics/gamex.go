package allmetrics

import (
	"fmt"
	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	gamexGid   uint
	_shard_ccu gm.Counter

	// Request per second
	yidunRps gm.Meter
	// The number of Request单调增长
	yidunRequests         gm.Counter
	_gameLocalRecordNum   gm.Counter
	_gameLocalRecordBytes gm.Counter
	cacheWriteNum         gm.Counter //写缓存
	cacheReadNum          gm.Counter //读缓存
)

func PrefixGamexMetrics(gid, sid uint) string {
	gamexGid = gid
	return fmt.Sprintf("%s.%d.%d", GamePrefix, gid, sid)
}

func InitGamexMetrics() {
	_shard_ccu = metrics.NewCustomCounter("ccu")
	_gameLocalRecordNum = metrics.NewCustomCounter("battlerecord")
	_gameLocalRecordBytes = metrics.NewCustomCounter("battlerecordbyte")
	yidunRps = metrics.NewCustomMeter("yidun")
	yidunRequests = metrics.NewCustomCounter("yiduntotal")

	cacheWriteNum = metrics.NewCustomCounter("cacheWrite")
	cacheReadNum = metrics.NewCustomCounter("cacheRead")
}

func AddGameLocalRecordNum() {
	_gameLocalRecordNum.Inc(1)
}

func AddGameLocalRecordBytes(size int64) {
	_gameLocalRecordBytes.Inc(size)
}

func AddShardCCU() {
	if _shard_ccu != nil {
		_shard_ccu.Inc(1)
	}
}

func ReduceShardCCU() {
	if _shard_ccu != nil {
		_shard_ccu.Dec(1)
	}
}

func GetShardCCU() int32 {
	if _shard_ccu == nil {
		return 0
	}
	return int32(_shard_ccu.Count())
}

func YidunAddNewRequest(n int64) {
	if yidunRps != nil {
		yidunRps.Mark(n)
	}
	if yidunRequests != nil {
		yidunRequests.Inc(n)
	}
}

func AddCacheWriteNum(n int64) {
	if cacheWriteNum != nil && n > 0 {
		cacheWriteNum.Inc(n)
	}
}

func AddCacheReadNum(n int64) {
	if cacheReadNum != nil && n > 0 {
		cacheReadNum.Inc(n)
	}
}
