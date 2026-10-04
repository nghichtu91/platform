package util

import (
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type AntsLogger struct{}

// Printf must have the same semantics as log.Printf.
func (AntsLogger) Printf(format string, args ...interface{}) { tilogs.L().Infof(format, args...) }
