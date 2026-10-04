package signalhandler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"time"

	"go.uber.org/atomic"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

var (
	stopTimeout        atomic.Duration
	serviceId          atomic.String
	writeDir           atomic.String
	stopTimeoutRunning atomic.Bool // 防止重复设置
)

const (
	// 目前线上服务的强杀超时时间是45s, 正常的服务停止时间30s肯定够了
	defaultStopTimeout = 30 * time.Second
)

func SetServiceId(id string) {
	if id == "" {
		tilogs.L().Infof("SignalKillHandle set server id failed, id is empty")
		return
	}
	tilogs.L().Infof("SignalKillHandle set server id %s", id)
	serviceId.Store(id)
}

func SetTimeout(t time.Duration) {
	if t < time.Second {
		// 默认最小超时时间为1s
		t = time.Second
	}
	tilogs.L().Infof("SignalKillHandle set stopTimeout %v", t)
	stopTimeout.Store(t)
}

func SetWriteDir(dir string) {
	// 允许为空, 相当于当前目录
	dir = filepath.Clean(dir)
	tilogs.L().Infof("SignalKillHandle set write dir '%s'", dir)
	writeDir.Store(dir)
}

// timeId 生成一个随机的id. 纳秒级别的时间戳... 大概率不会重复
func timeId() string {
	now := time.Now().UnixNano()
	// 十六进制的时间戳
	id := fmt.Sprintf("%x", now)
	return id
}

type TimeoutStackConfig struct {
	StopTimeout time.Duration
	ServiceId   string
	WriteDir    string
}

// TimeoutWriteGoRoutine 起一个goroutine, 如果超过了一定时间还未退出, 则抓取 goroutine 的堆栈信息 并写入到文件
// stopTimeout: 超时时间
// serviceId: 服务id, 如果不为空, 则尝试写入到文件
// return: 返回一个取消函数, 可以提前取消超时
func TimeoutWriteGoRoutine(cfg TimeoutStackConfig) func() {
	stopTimeout := cfg.StopTimeout
	serviceId := cfg.ServiceId
	dir := cfg.WriteDir

	if stopTimeout <= 0 {
		tilogs.L().Infof("stopTimeout <= 0, do nothing")
		return func() {}
	}

	if !stopTimeoutRunning.CAS(false, true) {
		tilogs.L().Infof(" already running, do nothing")
		return func() {}
	}

	tilogs.L().Infof("stop service id %s, timeout %v", serviceId, stopTimeout)

	ctx := util.NewCancelCtxWithTimeout(context.Background(), stopTimeout)

	go func(start time.Time) {
		defer func() {
			p := recover()
			if p != nil {
				tilogs.L().Errorf("TimeoutWriteGoRoutine panic %v, cost %v", p, time.Since(start))
			}
		}()
		// 整体是异步的, 所以并不能保证一定能够写入到文件
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				// 打印 goroutine 堆栈信息
				var write = os.Stdout
				var fName = "stdout"
				var debug = 2 // debug=2, 输出的是文本表示的堆栈信息

				if serviceId != "" {
					// 写入到文件
					name := fmt.Sprintf("goroutine_%s_%s.pb",
						time.Now().Format("20060102_150405"), serviceId,
					)
					name = path.Join(dir, name)
					if f, err := os.Create(name); err != nil {
						tilogs.L().Errorf(" create file %s failed. %v", name, err)
					} else {
						fName = name
						debug = 0 // debug=0, 输出的是 pprof 格式的堆栈信息
						write = f
						defer func(f *os.File) {
							err := f.Sync()
							if err != nil {
								tilogs.L().Errorf(" sync file %v failed. %v", name, err)
							}
							// 尝试关闭文件
							err = f.Close()
							if err != nil {
								tilogs.L().Errorf(" close file %v failed. %v", name, err)
							}
						}(f)
					}
				}

				err := pprof.Lookup("goroutine").WriteTo(write, debug)
				tilogs.L().Alarm("service %v stop timeout, write goroutine stacks to '%s', ret %v",
					serviceId, fName, err)
			} else {
				// do nothing
				tilogs.L().Infof(" Done. cost %v", time.Since(start))
			}
		}
	}(time.Now())

	return func() {
		tilogs.L().Infof(" TimeoutWriteGoRoutine cancel")
		ctx.Stop()
	}
}

// DebugReset 重置全局变量, 用于测试
func DebugReset() {
	stopTimeout.Store(defaultStopTimeout)
	serviceId.Store(timeId())
	writeDir.Store(".")
	stopTimeoutRunning.Store(false)
}
