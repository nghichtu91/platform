package tracing

import (
	"errors"
)

const (
	Jaeger = "jaeger"
)

var (
	ErrGlobalTracerNotInit = errors.New("global tracer not init")
)
