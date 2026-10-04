package tracing

import (
	"errors"
	"time"

	jaegerCfg "github.com/uber/jaeger-client-go/config"
)

var (
	jCfg *JaegerConfig
)

var (
	ErrJaegerNotInit = errors.New("jaeger not init")
)

// JaegerConfig jaeger的初始化参数
// 和jaeger包是一一对应的，不明白参数含义可以直接看源码
// 这里只是封装了一下方便配置
type JaegerConfig struct {
	ServiceName string
	Disabled    bool

	SamplerType             string  // 取样类型
	SamplerParam            float64 // 取样参数
	SamplingServerURL       string  //策略取样服务地址
	SamplingRefreshInterval int32   //策略取样刷新间隔
	// reporter
	LogSpans      bool
	AgentHostPort string
}

func InitJaeger(cfg *JaegerConfig) {
	jCfg = cfg
}

func NewJaegerByCfg(jc *JaegerConfig) *jaegerCfg.Configuration {
	return &jaegerCfg.Configuration{
		ServiceName: jc.ServiceName,
		Disabled:    jc.Disabled,
		Sampler: &jaegerCfg.SamplerConfig{
			Type:                    jc.SamplerType,
			Param:                   jc.SamplerParam,
			SamplingServerURL:       jc.SamplingServerURL,
			SamplingRefreshInterval: time.Duration(jc.SamplingRefreshInterval) * time.Second,
		},
		Reporter: &jaegerCfg.ReporterConfig{
			LogSpans:           jc.LogSpans,
			LocalAgentHostPort: jc.AgentHostPort,
		},
	}
}
