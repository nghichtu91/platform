package util

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCancelCtx(t *testing.T) {

	run := func(t *testing.T, i int, ctx context.Context) {
		t.Logf("[%v] run", i)
		defer t.Logf("[%v] stop", i)
		var newTimer = time.NewTimer(time.Duration(rand.Intn(3)+1) * time.Second)
		defer newTimer.Stop()
		select {
		case <-newTimer.C:
			t.Logf("[%v] run timeout", i)
		case <-ctx.Done():
			t.Logf("[%v] cancel, cause by:%v", i, ctx.Err())
		}
	}

	t.Run("Timeout", func(t *testing.T) {
		var timeoutCtx, cancel = context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		var ctx = NewCancelCtxWith(timeoutCtx)
		for i := 0; i < 10; i++ {
			idx := i
			go run(t, idx, ctx)
		}
		time.Sleep(time.Second * 4)
		assert.True(t, ctx.IsTimeout())
	})

	t.Run("Cancel", func(t *testing.T) {
		var ctx = NewCancelCtx()
		for i := 0; i < 10; i++ {
			idx := i
			go run(t, idx, ctx)
		}
		time.Sleep(time.Second)
		ctx.Stop()
		time.Sleep(time.Second)
		assert.True(t, ctx.IsCancel())
	})
}
