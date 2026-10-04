package promemetrics

import (
	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/nghichtu91/platform/share/planx/pmetrics"
)

const (
	RedisPoolCapacityGaugeName    = "redis_capacity"
	RedisPoolAvailableGaugeName   = "redis_available"
	RedisPoolMaxCapGaugeName      = "redis_maxcap"
	RedisPoolWaitCountGaugeName   = "redis_waitcount"
	RedisPoolWaitTimeGaugeName    = "redis_waittime"
	RedisPoolIdleTimeoutGaugeName = "redis_idletimeout"
)

var (
	redisPoolCapacityGauge    prometheus.Gauge
	redisPoolAvailableGauge   prometheus.Gauge
	redisPoolMaxCapGauge      prometheus.Gauge
	redisPoolWaitCountGauge   prometheus.Gauge
	redisPoolWaitTimeGauge    prometheus.Gauge
	redisPoolIdleTimeoutGauge prometheus.Gauge
)

func InitRedisMetrics() error {

	capacityGauge, capacityErr := pmetrics.NewGauge(pmetrics.GetCollectorName(pmetricslog.JobName, RedisPoolCapacityGaugeName))
	if capacityErr != nil {
		return capacityErr
	}
	redisPoolCapacityGauge = capacityGauge

	availableGauge, availableErr := pmetrics.NewGauge(pmetrics.GetCollectorName(pmetricslog.JobName, RedisPoolAvailableGaugeName))
	if availableErr != nil {
		return availableErr
	}
	redisPoolAvailableGauge = availableGauge

	maxCapGauge, maxCapErr := pmetrics.NewGauge(pmetrics.GetCollectorName(pmetricslog.JobName, RedisPoolMaxCapGaugeName))
	if maxCapErr != nil {
		return maxCapErr
	}
	redisPoolMaxCapGauge = maxCapGauge

	waitCountGauge, waitCountErr := pmetrics.NewGauge(pmetrics.GetCollectorName(pmetricslog.JobName, RedisPoolWaitCountGaugeName))
	if waitCountErr != nil {
		return waitCountErr
	}
	redisPoolWaitCountGauge = waitCountGauge

	waitTimeGauge, waitTimeErr := pmetrics.NewGauge(pmetrics.GetCollectorName(pmetricslog.JobName, RedisPoolWaitTimeGaugeName))
	if waitTimeErr != nil {
		return waitTimeErr
	}
	redisPoolWaitTimeGauge = waitTimeGauge

	idleTimeoutGauge, idleTimeoutErr := pmetrics.NewGauge(pmetrics.GetCollectorName(pmetricslog.JobName, RedisPoolIdleTimeoutGaugeName))
	if idleTimeoutErr != nil {
		return idleTimeoutErr
	}
	redisPoolIdleTimeoutGauge = idleTimeoutGauge

	return nil
}

// 更新。
func UpdateRedisPoolCapacityGauge(value int64) {
	if redisPoolCapacityGauge == nil {
		return
	}
	redisPoolCapacityGauge.Set(float64(value))
}

// 更新。
func UpdateRedisPoolAvailableGaugeName(value int64) {
	if redisPoolAvailableGauge == nil {
		return
	}
	redisPoolAvailableGauge.Set(float64(value))
}

// 更新。
func UpdateRedisPoolMaxCapGaugeName(value int64) {
	if redisPoolMaxCapGauge == nil {
		return
	}
	redisPoolMaxCapGauge.Set(float64(value))
}

// 更新。
func UpdateRedisPoolWaitCountGaugeName(value int64) {
	if redisPoolWaitCountGauge == nil {
		return
	}
	redisPoolWaitCountGauge.Set(float64(value))
}

// 更新。
func UpdateRedisPoolWaitTimeGaugeName(value int64) {
	if redisPoolWaitTimeGauge == nil {
		return
	}
	redisPoolWaitTimeGauge.Set(float64(value))
}

// 更新。
func UpdateRedisPoolIdleTimeoutGaugeName(value int64) {
	if redisPoolIdleTimeoutGauge == nil {
		return
	}
	redisPoolIdleTimeoutGauge.Set(float64(value))
}
