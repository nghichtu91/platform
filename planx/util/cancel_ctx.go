package util

import (
	"context"
	"errors"
	"time"
)

type cancelContext struct {
	context.Context
	cancel context.CancelFunc
}

func (c *cancelContext) Stop() {
	c.cancel()
}

// IsCancel returns true if the context is canceled. use after Done().
func (c *cancelContext) IsCancel() bool {
	err := c.Err()
	return err != nil && errors.Is(err, context.Canceled)
}

// IsTimeout returns true if the context is timeout. use after Done().
func (c *cancelContext) IsTimeout() bool {
	err := c.Err()
	return err != nil && errors.Is(err, context.DeadlineExceeded)
}

type CancelCtx interface {
	context.Context
	Stop()
	IsCancel() bool  // call after Done()
	IsTimeout() bool // call after Done()
}

func NewCancelCtx() CancelCtx {
	return NewCancelCtxWith(context.Background())
}

func NewCancelCtxWith(parent context.Context) CancelCtx {
	ctx, cancel := context.WithCancel(parent)
	return &cancelContext{
		Context: ctx,
		cancel:  cancel,
	}
}

func NewCancelCtxWithTimeout(parent context.Context, timeout time.Duration) CancelCtx {
	ctx, cancel := context.WithTimeout(parent, timeout)
	return &cancelContext{
		Context: ctx,
		cancel:  cancel,
	}
}
