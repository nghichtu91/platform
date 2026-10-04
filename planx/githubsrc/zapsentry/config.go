package zapsentry

import (
	"go.uber.org/zap/zapcore"
)

// Configuration is a minimal set of parameters for Sentry integration.
type Configuration struct {
	Tags              map[string]string
	DisableStacktrace bool
	Level             zapcore.Level
}

func (c *Configuration) clone() *Configuration {
	if c == nil {
		return &Configuration{}
	}
	cp := *c
	// clone一份tag, 防止并发读写
	var tags = make(map[string]string, len(c.Tags))
	for k, v := range c.Tags {
		tags[k] = v
	}
	cp.Tags = tags
	return &cp
}
