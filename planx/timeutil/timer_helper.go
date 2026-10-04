package timeutil

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/siddontang/go/timingwheel"
)

var (
	Timer10MS       = timingwheel.NewTimingWheel(10*time.Millisecond, 100*60) // 1分钟
	Timer50MS       = timingwheel.NewTimingWheel(50*time.Millisecond, 20)     // 1秒
	TimerSec        = timingwheel.NewTimingWheel(time.Second, 120)
	TimerMin        = timingwheel.NewTimingWheel(time.Minute, 240)
	Timer600S       = timingwheel.NewTimingWheel(time.Second, 600) // 以秒为间隔，最大10分钟
	TimerHour       = timingwheel.NewTimingWheel(time.Hour, 48)    // 以小时为间隔，最大1天
	BeijingLocation *time.Location
)

func init() {
	switch runtime.GOOS {
	case "windows":
		// 没有Go环境的Windows系统需要在执行目录下放置zone数据库。
		os.Setenv("ZONEINFO", "zoneinfo.zip")
	}

	BeijingLocation, _ = time.LoadLocation("Asia/Shanghai")
}

var TimeLocationEnv string // 记录TZ环境变量中读出的时区设置
var TimeLocation *time.Location

func CheckTimeLocation() {
	tz, ok := syscall.Getenv("TZ")
	if !ok {
		panic(fmt.Errorf("time location env TZ not set"))
	}
	log.Printf("tz %s", tz)
	location, err := time.LoadLocation(tz)
	if err != nil {
		panic(fmt.Errorf("time location from env TZ %s err %s", tz, err.Error()))
	}
	TimeLocationEnv = tz
	TimeLocation = location
	log.Printf("load time location from TZ %s success\n", tz)
}

func NewGTicker(t time.Duration) *GTicker {
	return &GTicker{
		t: t,
		c: make(chan struct{}, 1),
	}
}

type GTicker struct {
	ticker *time.Ticker
	sync.RWMutex
	t time.Duration
	c chan struct{}
}

func (gt *GTicker) Next() chan struct{} {
	gt.RLock()
	_c := gt.c
	gt.RUnlock()
	return _c
}

func (gt *GTicker) Run() {
	gt.ticker = time.NewTicker(gt.t)
	go func() {
		defer gt.ticker.Stop()
		for {
			select {
			case <-gt.ticker.C:
				gt.Lock()
				oc := gt.c
				gt.c = make(chan struct{}, 1)
				gt.Unlock()
				close(oc)
			}
		}
	}()
}

type Timer struct {
	*time.Timer
}

func NewTimer(duration time.Duration) *Timer {
	return &Timer{
		Timer: time.NewTimer(duration),
	}
}

func (t *Timer) Stop() {
	if t == nil || t.Timer == nil {
		return
	}
	if !t.Timer.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

func (t *Timer) Reset(duration time.Duration) {
	if t == nil || t.Timer == nil {
		return
	}
	t.Stop()
	t.Timer.Reset(duration)
}
