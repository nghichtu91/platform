package timeutil

import (
	"sync/atomic"
	"time"
)

// TimeCache 时间戳缓存
// 高并发场景下使用Time.Now()会有一定性能消耗
// 对于时间精度需求不高的场景，譬如gate的过期计时等，使用缓存的时间戳可以提升性能
// Note:
// 需要精确时间戳确定先后关系，譬如游戏内日志，不要使用cache
// 有调整时间需求的场景，不要使用cache
// 用于定时任务的时间戳，会人为产生波峰，不要使用cache
type TimeCache struct {
	ts   int64
	quit chan struct{}
}

// NewTimeCache 新建时间戳缓存
// 每隔step * resetTimes，使用time.Now()同步一次时间
// 每隔step, 时间戳自增对应数字
// 内部使用了Timer10MS实现，step必须大于10ms
func NewTimeCache(step time.Duration, resetTimes int) *TimeCache {
	tc := &TimeCache{
		ts:   time.Now().UnixNano(),
		quit: make(chan struct{}, 1),
	}

	go tc.loop(step, resetTimes, tc.quit)

	return tc
}

// GetCachedTimeNow 获取缓存的时间戳
func (tc *TimeCache) GetCachedTimeNow() int64 {
	return atomic.LoadInt64(&tc.ts) / int64(time.Second)
}

// GetCachedTimeNowNano 获取缓存的时间戳
func (tc *TimeCache) GetCachedTimeNowNano() int64 {
	return atomic.LoadInt64(&tc.ts)
}

func (tc *TimeCache) loop(step time.Duration, resetTimes int, quit chan struct{}) {
	sTick := Timer10MS.After(step)
	counter := resetTimes
	for {
		select {
		case <-sTick:
			counter--
			sTick = Timer10MS.After(step)

			if counter == 0 {
				atomic.StoreInt64(&tc.ts, time.Now().UnixNano())
				counter = resetTimes
				continue
			}
			atomic.AddInt64(&tc.ts, int64(step))
		case <-quit:
			return
		}
	}
}

func (tc *TimeCache) Stop() {
	close(tc.quit)
}
