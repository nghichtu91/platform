package util

import (
	"fmt"
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type WaitGroupWrapper struct {
	sync.WaitGroup
}

func (w *WaitGroupWrapper) Wrap(cb func()) {
	fn := GetFunctionRuntimeName(cb)
	tilogs.L().Infof("WaitGroupWrapper Wrap %s", fn)

	w.Add(1)
	go func() {
		defer tilogs.PanicCatcher("WaitGroupWrapper Wrap panic")
		defer func() {
			tilogs.L().Infof("WaitGroupWrapper Wrap Done %s", fn)
			w.Done()
		}()
		cb()
	}()
}

func (w *WaitGroupWrapper) WrapRetErr(cb func(c chan error), c chan error) {
	fn := GetFunctionRuntimeName(cb)
	tilogs.L().Infof("WaitGroupWrapper WrapRetErr %s", fn)

	w.Add(1)
	go func(c chan error) {
		defer tilogs.PanicCatcher("WaitGroupWrapper WrapRetErr")
		defer func() {
			tilogs.L().Infof("WaitGroupWrapper WrapRetErr Done %s", fn)
			w.Done()
		}()
		cb(c)
	}(c)
}

func CheckGoroutineStartErr(chanErr chan error, format string, v ...interface{}) bool {
	select {
	case err := <-chanErr:
		msg := fmt.Sprintf(format, v...)
		tilogs.L().Errorf("%s, err:%s", msg, err.Error())
		return false
	case <-time.After(500 * time.Millisecond):
		return true
	}
}
