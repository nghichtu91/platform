package zaplog

import (
	"sync"
	"testing"

	"github.com/getsentry/raven-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func TestZapLog(t *testing.T) {
	InitZapLog("", map[string]string{}, "")
	defer tilogs.Close()
	tilogs.L().Debugf("test debug")
	tilogs.L().Alarm("123213, %s, %d", "string", 456)
	tilogs.L().Errorf("123213, %s, %d", "string", 456)
}

func TestZapLogWith(t *testing.T) {
	logger := InitZapLog("log_dev.toml", map[string]string{
		"tag1": "100",
		"tag2": "200",
	}, "test")
	defer tilogs.Close()

	assert.NotNil(t, logger)

	tilogs.L().Errorf("hahaha %v %s %d", 100, "200", 300)
	var wg sync.WaitGroup
	const NUM = 100
	wg.Add(NUM)
	for i := 0; i < NUM; i++ {
		go func(i int) {
			defer wg.Done()
			tilogs.L().With("server_id", i, "server_ip", "localhost").Errorf("hahaha222 %v %s %d", 100, "200", 300)
		}(i)
	}
	wg.Wait()
}

func BenchmarkStack(b *testing.B) {

	const DEPTH = 15

	var caller func(i int, fn func())
	caller = func(i int, fn func()) {
		if i >= DEPTH {
			fn()
			return
		}
		caller(i+1, fn)
	}

	b.Run("raven", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			caller(0, func() {
				raven.NewStacktrace(0, 0, nil)
			})
		}
	})
	var s string
	b.Run("zap", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			caller(0, func() {
				s = zap.StackSkip("", 0).String
			})
		}
	})
	_ = s
}
