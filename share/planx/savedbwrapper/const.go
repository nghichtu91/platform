package savedbwrapper

import (
	"github.com/nghichtu91/platform/share/planx/timeutil"
)

const (
	// changed 默认长度
	// 从qa服日志看，大部分change只有一条
	// 部分change有不超过16条
	// 先将此长度设为16，后续有需要再优化
	defaultChangeLen = 16

	// 默认的debug日志数量
	// 某张表有sub change时会打印日志
	// 从日志看，这个值大部分情况为1，一般也不超过16
	debugLogLen = 16

	// 默认slice长度
	defaultSliceLen = 16

	// 启用自动过期时，键值更新后的自动过期时间
	autoExpireDefaultSeconds = 60 * timeutil.DaySec

	// 默认批量操作数量
	defaultBatchCount = 20
)
