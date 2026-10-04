//go:build windows
// +build windows

package signalhandler

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	//"github.com/nghichtu91/platform/share/planx/util/logs"
	"time"

	"github.com/nghichtu91/platform/share/planx/reloader"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type Stoper interface {
	Stop()
}

type Reloader interface {
	Reload()
}

type ReloaderFunc func()

func (r ReloaderFunc) Reload() {
	r()
}

type StoperFunc func()

func (s StoperFunc) Stop() {
	s()
}

type ReloaderObj struct {
	name     string
	reloader Reloader
}

var (
	signalLock sync.RWMutex
	stopers    []Stoper
	reloaders  []*ReloaderObj
	quit       chan struct{}
	quitReload chan struct{}
	once       sync.Once
	closeOnce  sync.Once
)

func init() {
	stopers = make([]Stoper, 0, 10)
	reloaders = make([]*ReloaderObj, 0)
	quit = make(chan struct{}, 1)
	quitReload = make(chan struct{}, 1)
}

func SignalReloadFunc(name string, f func()) {
	SignalReloadHandler(name, ReloaderFunc(f))
}

func SignalReloadHandler(name string, r Reloader) {
	signalLock.Lock()
	defer signalLock.Unlock()
	reloaders = append(reloaders, &ReloaderObj{name: name, reloader: r})
}

func SignalClose() {
	closeOnce.Do(func() {
		close(quit)
	})
}

func signalReloadHandle() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT) // only for windows

	for {
		select {
		case <-quitReload:
			return
		case <-ch:
			fmt.Printf("Got %v\n", "SIGUSR2")
			go func() {
				signalLock.RLock()
				defer signalLock.RUnlock()
				var reloadTomlHandler Reloader
				for _, handler := range reloaders {
					if strings.Contains(handler.name, reloader.ReloadFileName) {
						reloadTomlHandler = handler.reloader
					}
				}
				if reloadTomlHandler != nil {
					func() {
						defer tilogs.PanicCatcher("signalReloadHandle reload toml panic")
						reloadTomlHandler.Reload()
					}()
				}

				//如果reload.toml没有指定热更文件名，则所有reloadhandler都会执行， 否则将只热更指定文件
				//不能出现配置文件重名情况，例如：/conf/config.toml和/confd/config.toml；因为这里判断只是文件名，不是全路径
				for _, v := range reloaders {
					if strings.Contains(v.name, reloader.ReloadFileName) {
						continue
					}
					if len(reloader.ReLoadConfig.ReloadHandler) > 0 {
						var exist bool
						for _, reloadName := range reloader.ReLoadConfig.ReloadHandler {
							if strings.Contains(v.name, reloadName) {
								exist = true
							}
						}
						if exist {
							func() {
								defer tilogs.PanicCatcher("signalReloadHandle Reload")
								v.reloader.Reload()
							}()
						}
					} else {
						func() {
							defer tilogs.PanicCatcher("signalReloadHandle Reload")
							v.reloader.Reload()
						}()
					}
				}

			}()
		}
	}
}

func SignalKillHandler(s Stoper) {
	signalLock.Lock()
	defer signalLock.Unlock()
	stopers = append(stopers, s)
}

func SignalKillFunc(f func()) {
	SignalKillHandler(StoperFunc(f))
}

func SignalKillHandle() {
	once.Do(func() {
		go signalReloadHandle()

		stopch := make(chan os.Signal, 1)
		signal.Notify(stopch, syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)
		select {
		case sig := <-stopch:
			fmt.Printf("[stop] %s rev kill signal %v ...\n", time.Now().String(), sig)
		case <-quit:
			fmt.Printf("SignalKillHandle quit\n")
		}

		func() {
			signalLock.RLock()
			defer signalLock.RUnlock()
			i := 0
			for _, v := range stopers {
				func() {
					defer tilogs.PanicCatcher("SignalKillHandle Stop")
					v.Stop()
				}()
				i++
			}
			close(quitReload)
		}()
	})
}
