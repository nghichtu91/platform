package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"

	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/planx/signalhandler"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func main() {
	zaplog.InitZapLog("log_dev.toml", nil, "")
	defer tilogs.Close()

	stopCh := make(chan struct{}, 1)
	tilogs.L().Debugf("test %s log", "debug")
	tilogs.L().Infof("test %s log", "info")
	tilogs.L().Warnf("test %s log", "warn")
	tilogs.L().Errorf("test %s log", "error")

	l := tilogs.L().With("accountid", "101:10104:d6fe8969-471d-4e69-a2d1-d500b0766c4c")
	l.Debugf("test %s log with with", "debug")
	l.Infof("test %s log with with", "info")
	l.Warnf("test %s log with with", "warn")
	l.Errorf("test %s log with with", "error")

	lu := tilogs.L().WithUser("100:1000:TestUID", "OtherKey", "OtherValue")
	lu.Debugf("test %s log with sentry user", "debug")
	lu.Infof("test %s log with sentry user", "info")
	lu.Warnf("test %s log with sentry user", "warn")
	lu.Errorf("test %s log with sentry user", "error")

	log.Println("this is a log log ")

	go func() {
		defer tilogs.PanicCatcher("test catch panic")
		panic(fmt.Errorf("in go panic 1"))
	}()

	go func() {
		defer tilogs.PanicCatcherWithInfo(l, "test catch panic with with")
		panic(fmt.Errorf("in go panic 2"))
	}()

	var waitGroup util.WaitGroupWrapper
	signalhandler.SignalKillFunc(func() {
		close(stopCh)
	})
	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })

	for i := 0; i < 100; i++ {
		go func(id int) {
			t := time.NewTicker(time.Second)
			for {
				select {
				case <-t.C:
					tilogs.L().Debugf("%d tick!", id)
					tilogs.L().Infof("%d tick!", id)
				case <-stopCh:
					t.Stop()
					return
				}
			}
		}(i)
	}
	waitGroup.Wait()
}
