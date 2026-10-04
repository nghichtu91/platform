package tracing

import (
	"errors"
)

var (
	ErrInvalidTracerType = errors.New("invalid tracer type")
)

// Config 配置文件
// 这里也预留了接口为了后续随时替换
type Config struct {
	TracerType string

	Jaeger *JaegerConfig
}

func (c *Config) Init() error {
	switch c.TracerType {
	case Jaeger:
		InitJaeger(c.Jaeger)
	default:
		return ErrInvalidTracerType
	}

	return nil
}
