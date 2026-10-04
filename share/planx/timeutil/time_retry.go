package timeutil

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

// RetryIfErr 如果retryFunc返回err就按照第二个参数的重试策略进行重试
// 重试期间执行成功了，则立刻返回nil
// 若重试策略全执行完，依然没成功，则返回err
func RetryIfErr(retryFunc backoff.Operation, b backoff.BackOff) error {
	return backoff.Retry(retryFunc, b)
}

const (
	// 3~4次, 总时间2~4s
	defaultInitialInterval     = 500 * time.Millisecond // 初始间隔时间
	defaultRandomizationFactor = 0.5
	defaultMultiplier          = 2               // 指数翻倍系数
	defaultMaxInterval         = 2 * time.Second // 最长重试间隔
	defaultMaxElapsedTime      = 2 * time.Second // 最长重试时间
)

func New2SecBackOff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     defaultInitialInterval,
		RandomizationFactor: defaultRandomizationFactor,
		Multiplier:          defaultMultiplier,
		MaxInterval:         defaultMaxInterval,
		MaxElapsedTime:      defaultMaxElapsedTime,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	if b.RandomizationFactor < 0 {
		b.RandomizationFactor = 0
	} else if b.RandomizationFactor > 1 {
		b.RandomizationFactor = 1
	}
	b.Reset()
	return b
}

func NewBackOffWithMaxElapsedTime(maxElapsedTime time.Duration) *backoff.ExponentialBackOff {
	if maxElapsedTime <= 0 {
		maxElapsedTime = defaultMaxElapsedTime
	}
	b := &backoff.ExponentialBackOff{
		InitialInterval:     defaultInitialInterval,
		RandomizationFactor: defaultRandomizationFactor,
		Multiplier:          defaultMultiplier,
		MaxInterval:         defaultMaxInterval,
		MaxElapsedTime:      maxElapsedTime,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	if b.RandomizationFactor < 0 {
		b.RandomizationFactor = 0
	} else if b.RandomizationFactor > 1 {
		b.RandomizationFactor = 1
	}
	b.Reset()
	return b
}
