package service_check

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nghichtu91/platform/share/planx/3rd_party_api/feishu"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
)

func init() {
	zaplog.TestLogger()
}

func TestStartup(t *testing.T) {

	feishu.InitFeishu("fHSG3jh9OKOyrfKUXvVU7b")
	const webhook = "https://open.feishu.cn/open-apis/bot/v2/hook/ca0bee7d-c585-44ef-9d28-8dff02065578"

	t.Run("timeout1", func(t *testing.T) {
		var isTimeout bool
		var timeoutStatus int
		var startup = NewStartupCheck(DefaultStartupOption().
			OnTimeout(func(status int) {
				timeoutStatus = status
				isTimeout = true
				t.Logf("case 1 timeout status %d", status)
			}).
			Uid("timeout case").
			Timeout(time.Second * 2).  // 首次报警在2s
			WarnInterval(time.Second). // 然后持续报警
			AlarmOnTimeout(webhook),
		)
		_ = startup
		time.Sleep(time.Second * 5)
		assert.True(t, isTimeout)
		assert.Equal(t, StartupPrepare, timeoutStatus)

		startup.Running()
		time.Sleep(time.Second * 5) // 测试停止后是否还会报警
	})

	t.Run("timeout2", func(t *testing.T) {
		var isTimeout bool
		var timeoutStatus int
		var startup = NewStartupCheck(DefaultStartupOption().
			OnTimeout(func(status int) {
				timeoutStatus = status
				isTimeout = true
				t.Logf("case 1 timeout status %d", status)
			}).
			Uid("timeout2 case").
			Timeout(time.Second * 2). // 首次报警在2s
			WarnInterval(0).          // 停止持续报警
			AlarmOnTimeout(webhook),
		)
		_ = startup
		time.Sleep(time.Second * 3)
		assert.True(t, isTimeout)
		assert.Equal(t, StartupPrepare, timeoutStatus)
	})

	t.Run("timeout3", func(t *testing.T) {
		var isTimeout bool
		var timeoutStatus int
		var startup = NewStartupCheck(DefaultStartupOption().
			OnTimeout(func(status int) {
				timeoutStatus = status
				isTimeout = true
				t.Logf("case 1 timeout status %d", status)
			}).
			Uid("timeout3 case").
			Timeout(time.Millisecond * 500). // 首次报警在1s
			WarnInterval(1).                 // 停止持续报警
			AlarmOnTimeout(webhook),
		)
		_ = startup
		time.Sleep(time.Second * 4)
		startup.Stop() // 主动停止
		assert.True(t, isTimeout)
		assert.Equal(t, StartupPrepare, timeoutStatus)
		time.Sleep(time.Second)
	})

	t.Run("Cancel", func(t *testing.T) {
		var isTimeout bool
		var isStop bool
		var ctx, cancel = context.WithCancel(context.Background())
		var startup = NewStartupCheckContext(ctx,
			DefaultStartupOption().
				OnTimeout(nil).
				OnTimeout(func(status int) {
					assert.Fail(t, "should not timeout")
					isTimeout = true
				}).
				OnStop(nil).
				OnStop(func(status int) {
					t.Logf("case 2 stop status %d", status)
					assert.Equal(t, StartupPrepare, status)
					isStop = true
				}).
				Uid("parent cancel").
				Timeout(time.Second-1).
				WarnInterval(0), // 关闭报警
		)
		cancel()
		_ = startup
		time.Sleep(time.Second * 2)
		assert.True(t, isStop)
		assert.False(t, isTimeout)
	})

	t.Run("NoTimeout", func(t *testing.T) {
		var isStop bool
		var startup = NewStartupCheck(DefaultStartupOption().
			OnTimeout(func(i int) {
				assert.Fail(t, "should not timeout")
			}).
			Uid("no timeout").
			Timeout(0).
			OnStop(func(i int) {
				isStop = true
			}),
		)
		time.Sleep(time.Second)
		startup.Stop()
		time.Sleep(time.Second)
		assert.True(t, isStop)
	})

	t.Run("Stop", func(t *testing.T) {
		var isStop bool
		var startup = NewStartupCheck(DefaultStartupOption().
			OnStop(func(status int) {
				isStop = true
				t.Logf("case 3 stop status %d", status)
				assert.Equal(t, StartupPrepare, status)
			}).
			OnTimeout(func(i int) {
				assert.Fail(t, "should not timeout")
			}).
			Uid("self stop").
			Timeout(time.Second),
		)
		_ = startup
		startup.Stop()
		time.Sleep(time.Second * 2)
		assert.True(t, isStop)
	})

	t.Run("Running", func(t *testing.T) {
		var isStop bool
		var startup = NewStartupCheck(DefaultStartupOption().
			Uid("switch to running").
			OnStop(func(status int) {
				isStop = true
				t.Logf("case 4 stop status %d", status)
				assert.Equal(t, StartupRunning, status)
			}).
			OnTimeout(func(i int) {
				assert.Fail(t, "should not timeout")
			}).
			Timeout(time.Second),
		)
		_ = startup
		time.Sleep(time.Millisecond * 100)
		startup.Running()
		startup.Stop()
		startup.Stop()
		time.Sleep(time.Millisecond * 100)

		assert.True(t, isStop)
	})

	t.Run("Not Clog", func(t *testing.T) {
		var startup = NewStartupCheck(DefaultStartupOption().
			Uid("not clog"))
		startup.Running()
		startup.Running()
		startup.Running()
		startup.Running()
		startup.Running()
	})
}
