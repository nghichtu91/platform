package signalhandler_test

import (
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
)

func init() {
	zaplog.InitDebugLog()
}

func TestTimeoutWrite(t *testing.T) {

	timeout := func(id string) {
		signalhandler.DebugReset()
		cancel := signalhandler.TimeoutWriteGoRoutine(signalhandler.TimeoutStackConfig{
			StopTimeout: time.Second * 3,
			ServiceId:   id,
			WriteDir:    "",
		})
		defer cancel()
		time.Sleep(time.Second * 5)
	}
	stop := func(id string) {
		signalhandler.DebugReset()
		cancel := signalhandler.TimeoutWriteGoRoutine(signalhandler.TimeoutStackConfig{
			StopTimeout: time.Second * 3,
			ServiceId:   id,
			WriteDir:    "",
		})
		time.Sleep(time.Second * 1)
		cancel()
		time.Sleep(time.Second * 1)
	}

	var warp = func(f func(string), p string) func(*testing.T) {
		return func(t *testing.T) { f(p) }
	}

	t.Run("stopTimeoutFile", warp(timeout, "test"))
	t.Run("stopTimeoutStdout", warp(timeout, ""))

	t.Run("stopCancel", warp(stop, "test"))
	t.Run("stopCancelStdout", warp(stop, ""))
}

func TestReal(t *testing.T) {

	const TIMEOUT = time.Second * 3

	testF := func(timeout time.Duration) {
		signalhandler.DebugReset()
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			signalhandler.SignalKillHandle()
		}()

		time.Sleep(time.Second)

		signalhandler.SetTimeout(TIMEOUT)
		signalhandler.SetServiceId("test")
		signalhandler.SetWriteDir("./pprof")

		signalhandler.SignalKillFunc(func() {
			time.Sleep(timeout)
		})

		go func() {
			defer wg.Done()
			_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}()

		wg.Wait()
		time.Sleep(time.Second)
	}

	t.Run("Timeout", func(t *testing.T) {
		testF(TIMEOUT + time.Second)
	})

	t.Run("NoTimeout", func(t *testing.T) {
		testF(TIMEOUT - time.Second)
	})
}
