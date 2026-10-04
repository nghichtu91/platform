package service_check

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/nghichtu91/platform/share/planx/3rd_party_api/feishu"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

const (
	StartupIdle = iota
	StartupPrepare
	StartupRunning
)

const (
	StartupAttr = "startup"
	StartupType = "status" // 为了遵循原有的命名规范. 暂时没有额外的用途
)

type StartupOption struct {
	uid          string
	timeout      time.Duration
	onStop       func(int)
	onTimeout    func(int)
	warnInterval time.Duration // 如果发生了超时, 后续持续报警的间隔, 直到结束或者手动停止

	// 内部统计用
	startAt time.Time
}

func DefaultStartupOption() *StartupOption {
	return &StartupOption{
		startAt:      time.Now(),
		onStop:       emptyFunc,
		onTimeout:    emptyFunc,
		warnInterval: time.Second * 30, // 默认30s
	}
}

// Timeout 设置超时时间
func (o *StartupOption) Timeout(timeout time.Duration) *StartupOption {
	if timeout != 0 && timeout <= time.Second {
		// 最低1s
		timeout = time.Second
	}
	o.timeout = timeout
	return o
}

// Uid 设置uid, 用于标识
func (o *StartupOption) Uid(uid string) *StartupOption {
	o.uid = uid
	return o
}

// OnStop 设置停止时的回调. 会覆盖之前的设置
func (o *StartupOption) OnStop(onStop func(int)) *StartupOption {
	if onStop == nil {
		onStop = emptyFunc
	}
	o.onStop = onStop
	return o
}

// OnTimeout 设置超时时的回调. 会覆盖之前的设置
func (o *StartupOption) OnTimeout(onTimeout func(int)) *StartupOption {
	if onTimeout == nil {
		onTimeout = emptyFunc
	}
	o.onTimeout = onTimeout
	return o
}

// WarnInterval 设置超时后的报警间隔
func (o *StartupOption) WarnInterval(interval time.Duration) *StartupOption {
	if interval != 0 && interval <= time.Second {
		// 最低1s
		// 0 表示不报警
		interval = time.Second
	}
	o.warnInterval = interval
	return o
}

// AlarmOnTimeout 启用超时报警. 这个应该放置在 OnTimeout 后调用.
// url: 飞书的url, 如果不为空, 会发送飞书报警
func (o *StartupOption) AlarmOnTimeout(url string) *StartupOption {
	var old = o.onTimeout
	o.OnTimeout(
		func(status int) {
			content := fmt.Sprintf("%s startupCheck timeout. status %v. startAt %v, passed %v",
				o.uid, status, o.startAt.Format(time.RFC3339Nano), time.Since(o.startAt))
			tilogs.L().Alarm(content)
			if old != nil {
				old(status)
			}
			if url != "" {
				err := feishu.SendFeishu(url, fmt.Sprintf(`
启动超时报警!
服务器uid: %s
结束状态:  %d
超时时长:  %s
开始时间:  %s
执行时间:  %s
`,
					o.uid, status, o.timeout, o.startAt.Format(time.RFC3339Nano), time.Since(o.startAt)), false)
				if err != nil {
					tilogs.L().Errorf(
						"%s startupCheck AlarmOnTimeout SendFeishu err %s. content %s",
						o.uid, err.Error(), content,
					)
				}
			}
		},
	)
	return o
}

type IStartupCheck interface {
	Running()              // 切换到运行状态
	Stop()                 // 主动停止
	Done() <-chan struct{} // 完成通知
}

func NewStartupCheck(option *StartupOption) IStartupCheck {
	return NewStartupCheckContext(context.Background(), option)
}

func NewStartupCheckContext(ctx context.Context, opt *StartupOption) IStartupCheck {
	var baseCtx util.CancelCtx
	var checkCtx util.CancelCtx
	if opt.timeout != 0 {
		// 主动取消放在checkCtx
		checkCtx = util.NewCancelCtxWith(ctx)
		// 超时放在baseCtx, 并且接受checkCtx的控制
		baseCtx = util.NewCancelCtxWithTimeout(checkCtx, opt.timeout)
	} else {
		// 没有超时时间, 只有主动取消, 二者都是checkCtx
		baseCtx = util.NewCancelCtxWith(ctx)
		checkCtx = baseCtx
	}
	var base = createBaseCheck(baseCtx, StartupAttr, StartupType)
	base.Update(StartupIdle)
	check := &startupCheck{
		base: base,
		ctx:  checkCtx,
		opt:  opt,
	}
	check.Prepare() // 自动准备, 保证只会启动一次
	return check
}

type startupCheck struct {
	base baseCheck
	ctx  util.CancelCtx // 自身的ctx, 仅用来主动取消
	opt  *StartupOption
}

func (a *startupCheck) Prepare() {
	a.base.Update(StartupPrepare)
	tilogs.L().Infof("%s startupCheck.Prepare. passed %v", a.opt.uid, time.Since(a.opt.startAt))
	a.runCheck()
}

func (a *startupCheck) Running() {
	a.base.Update(StartupRunning)
	tilogs.L().Infof("%s startupCheck.Running. passed %v", a.opt.uid, time.Since(a.opt.startAt))
	a.Stop() // 一旦进入Running状态，就停止检查
}

// Stop 主动停止
func (a *startupCheck) Stop() {
	// 不管如何, 这个ctx Stop 都会传递到 baseCheck
	a.ctx.Stop()
}

// Done 是否完成
func (a *startupCheck) Done() <-chan struct{} {
	return a.ctx.Done()
}

// runCheck
func (a *startupCheck) runCheck() {
	opt := a.opt
	go func() {
		defer tilogs.PanicCatcher("%s startupCheck.runCheck panic ", opt.uid)
		// 这里到了, 可能是超时, 也可能是取消
		<-a.base.Done()
		val := a.base.Value()
		isTimeout := a.base.IsTimeout()
		tilogs.L().Infof("%s startupCheck.runCheck is timeout %v status %d, passed %v",
			opt.uid, isTimeout, val, time.Since(opt.startAt))
		if a.base.IsTimeout() {
			// 超时, 启动报警
			a.warnTimeout(int(val))
		} else {
			// 自然结束
			val := a.base.Value()
			if opt.onStop != nil {
				opt.onStop(int(val))
			}
		}
	}()
}

// warnTimeout
func (a *startupCheck) warnTimeout(status int) {
	// 到达这个函数, 说明已经超时, 此时baseCheck已经是超时状态

	opt := a.opt
	warpStatusCallback := func(callback func(int)) func(int) {
		return func(status int) {
			defer tilogs.PanicCatcher("%s startupCheck.warnTimeout panic ", opt.uid)
			callback(status)
		}
	}

	var timeoutFunc = emptyFunc
	if opt.onTimeout != nil {
		timeoutFunc = warpStatusCallback(opt.onTimeout)
	}
	var stopFunc = emptyFunc
	if opt.onStop != nil {
		stopFunc = warpStatusCallback(opt.onStop)
	}

	// 首次报警
	timeoutFunc(status)

	if opt.warnInterval < time.Second {
		// 不需要后续的持续报警, 直接返回
		tilogs.L().Infof("%s startupCheck.warnTimeout stop. no warn", opt.uid)
		return
	}

	// success or dead.
	var ticker = time.NewTicker(opt.warnInterval)
	defer ticker.Stop()

	isRunning := func() bool {
		val := a.base.Value()
		if val == int64(StartupRunning) {
			tilogs.L().Infof("%s startupCheck.warnTimeout stop. success, passed %v",
				opt.uid, time.Since(opt.startAt))
			stopFunc(int(val))
			return true
		}
		return false
	}

	// 定时check, 或者有更新时检查一下
	for {
		select {
		case <-a.ctx.Done():
			// 主动停止
			tilogs.L().Infof("%s startupCheck.warnTimeout stop. done, passed %v",
				opt.uid, time.Since(opt.startAt))
			stopFunc(int(a.base.Value()))
			return
		case <-ticker.C:
			// 定时检查
			if isRunning() {
				return
			}
			timeoutFunc(int(a.base.Value()))
		case <-a.base.OnUpdate():
			// value 更新
			if isRunning() {
				return
			}
		}
	}
}

func emptyFunc(int) { /* do nothing*/ }
