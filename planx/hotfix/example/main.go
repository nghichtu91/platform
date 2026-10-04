package main

import (
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"

	"github.com/nghichtu91/platform/share/planx/hotfix/example/core"

	"github.com/nghichtu91/platform/share/planx/hotfix"
)

func main() {
	zaplog.InitZapLog("", nil, "")
	go core.Run()

	go func() {
		for {
			time.Sleep(5 * time.Second)
			hotfix.TryHotfixLocal()
		}
	}()

	for {
		time.Sleep(5 * time.Second)
	}
}
