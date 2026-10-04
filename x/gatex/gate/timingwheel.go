package gate

import (
	"time"

	"github.com/siddontang/go/timingwheel"
)

var secondsTimer *timingwheel.TimingWheel

func init() {
	secondsTimer = timingwheel.NewTimingWheel(time.Second, 1800)
}
