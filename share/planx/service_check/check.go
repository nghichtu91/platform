package service_check

import (
	"context"
	"errors"

	"github.com/rcrowley/go-metrics"

	metrics2 "github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/util"
)

type baseCheck struct {
	ctx          util.CancelCtx
	gauge        metrics.Gauge
	updateNotify chan struct{} // 用于通知更新
}

func createBaseCheck(ctx util.CancelCtx, attr, typ string) baseCheck {
	var check baseCheck
	check.ctx = ctx
	// register 自带的前缀: jws2.{server}.{gid}.{server_id}
	// 这里补充的是 check.{attr}.{typ}
	// 对应 jws2_server_index指标, index过滤条件为check
	check.gauge = metrics2.NewGauge("check." + attr + "." + typ)
	check.updateNotify = make(chan struct{}, 1)
	return check
}

func (a *baseCheck) Stop()                 { a.ctx.Stop() }
func (a *baseCheck) Done() <-chan struct{} { return a.ctx.Done() }
func (a *baseCheck) Value() int64          { return a.gauge.Value() }

func (a *baseCheck) Update(status int) {
	a.gauge.Update(int64(status))
	select {
	case a.updateNotify <- struct{}{}:
	default:
	}
}

// IsTimeout 检查是不是超时失败
func (a *baseCheck) IsTimeout() bool {
	select {
	case <-a.ctx.Done():
		return errors.Is(a.ctx.Err(), context.DeadlineExceeded)
	default:
		return false
	}
}

// OnUpdate 注册更新通知
func (a *baseCheck) OnUpdate() <-chan struct{} {
	return a.updateNotify
}
