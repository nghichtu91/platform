//go:build !windows
// +build !windows

package signalhandler

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nghichtu91/platform/share/planx/reloader"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

type Stoper interface {
	Stop()
}

type Reloader interface {
	Reload()
}

type Downliner interface {
	Downline()
}

type SIGUSR1Trigger interface {
	TrigSIGUSR1()
}

type ReloaderFunc func()

func (r ReloaderFunc) Reload() {
	r()
}

type StoperFunc func()

func (s StoperFunc) Stop() {
	s()
}

type DownlinerFunc func()

func (d DownlinerFunc) Downline() {
	d()
}

type SignalTrigFunc func()

// TrigSIGUSR1 SIGUSR1 触发事件
func (d SignalTrigFunc) TrigSIGUSR1() {
	d()
}

type ReloaderObj struct {
	name     string
	reloader Reloader
}

var (
	signalLock   sync.RWMutex
	stopers      []Stoper
	downliners   []Downliner
	usr1triggers []SIGUSR1Trigger
	quit         chan struct{}
	reloaders    []*ReloaderObj
	regOnce      sync.Once
	closeOnce    sync.Once
)

func init() {
	stopers = make([]Stoper, 0, 10)
	reloaders = make([]*ReloaderObj, 0)
	downliners = make([]Downliner, 0, 10)
	usr1triggers = make([]SIGUSR1Trigger, 0, 10)
	quit = make(chan struct{}, 1)
}

func SignalReloadFunc(name string, f func()) {
	SignalReloadHandler(name, ReloaderFunc(f))
}

func SignalReloadHandler(name string, r Reloader) {
	signalLock.Lock()
	defer signalLock.Unlock()
	reloaders = append(reloaders, &ReloaderObj{name: name, reloader: r})
}

func SignalDownlinerFunc(f func()) {
	SignalDownlinerHandler(DownlinerFunc(f))
}

func SignalDownlinerHandler(d Downliner) {
	signalLock.Lock()
	defer signalLock.Unlock()
	downliners = append(downliners, d)
}

// AddUSR1TriggerFunc 添加一个USR1信号触发的行为
func AddUSR1TriggerFunc(f func()) {
	signalLock.Lock()
	defer signalLock.Unlock()
	usr1triggers = append(usr1triggers, SignalTrigFunc(f))
}

func signalUSR1Handle() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1) // only for unix

	for {
		select {
		case <-quit:
			return
		case <-ch:
			tilogs.L().Infof("Got %v\n", "SIGUSR1")
			func() {
				signalLock.RLock()
				defer signalLock.RUnlock()
				for _, v := range downliners {
					func() {
						defer tilogs.PanicCatcher("signalUSR1Handle Downline")
						v.Downline()
					}()
				}
				for _, v := range usr1triggers {
					func() {
						defer tilogs.PanicCatcher("signalReloadHandle TrigSIGUSR1")
						v.TrigSIGUSR1()
					}()
				}
			}()
		}
	}
}

func signalReloadHandle() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR2) // only for unix

	for {
		select {
		case <-quit:
			return
		case <-ch:
			tilogs.L().Infof("Got %v\n", "SIGUSR2")
			func() {
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
	tilogs.L().Infof("SignalKillHandler stopper %s added to index %d", util.GetTypeName(s), len(stopers))
	stopers = append(stopers, s)
}

func SignalKillFunc(f func()) {
	tilogs.L().Infof("SignalKillFunc fn %s added ", util.GetFunctionRuntimeName(f))
	SignalKillHandler(StoperFunc(f))
}

func SignalIgnore() {
	signal.Ignore(syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)
}

func SignalKillHandle() {
	regOnce.Do(func() {

		// 只要调用了SignalKillHandle，默认会有一个default的serverId
		// 这里不能调用 set 接口, 因为 tilogs.L 可能会有 race condition
		serviceId.Store(timeId())
		stopTimeout.Store(defaultStopTimeout)
		writeDir.Store(".") // 默认写入到当前目录

		go signalReloadHandle()
		go signalUSR1Handle()

		stopch := make(chan os.Signal, 1)
		signal.Notify(stopch, syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)
		select {
		case sig := <-stopch:
			fmt.Printf("[stop] %s rev kill signal %v ...\n", time.Now().String(), sig)
		}

		OnClose()
	})
}

func OnClose() {
	signalLock.Lock()
	defer signalLock.Unlock()

	timeout := stopTimeout.Load()
	id := serviceId.Load()
	dir := writeDir.Load()

	// 这里不能使用 defer 去取消,
	// 因为有些停止逻辑不是通过 Stop() 来注册的, 而是通过 defer 在主函数中注册的
	// 这里只管启动超时检测, 不停止. 如果超时就会输出 goroutine 堆栈信息
	// 如果在超时事件内, 程序正常退出, 则不会输出 goroutine 堆栈信息
	TimeoutWriteGoRoutine(TimeoutStackConfig{
		StopTimeout: timeout,
		ServiceId:   id,
		WriteDir:    dir,
	})

	i := 0
	for _, v := range stopers {
		func(i int) {
			var start = time.Now()
			defer tilogs.PanicCatcher("SignalKillHandle Stop")
			v.Stop()
			// 跟踪每个stopper实际的执行时间，发生死锁时方便定位
			tilogs.L().Infof("SignalKillHandle %d stopper stopped cost %v", i, time.Since(start))
		}(i)
		i++
	}
	stopers = make([]Stoper, 0, 10)

	closeOnce.Do(func() {
		close(quit)
	})
}
