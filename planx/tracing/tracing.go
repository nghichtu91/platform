package tracing

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc/metadata"

	"github.com/uber/jaeger-client-go"

	"github.com/opentracing/opentracing-go"
	jaegerCfg "github.com/uber/jaeger-client-go/config"
	jaegerLog "github.com/uber/jaeger-client-go/log"
	config2 "github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

// Start 加载指定的配置文件
func Start(cfgName string) error {
	var c Config
	cfg := config2.NewConfigToml(cfgName, &c)
	if cfg == nil {
		return fmt.Errorf("LocalConfig Read Error")
	}
	if err := c.Init(); err != nil {
		return err
	}

	tilogs.L().Infof("tracing config %+v loaded", c)
	if c.Jaeger != nil {
		tilogs.L().Infof("tracing config jaeger %+v loaded", c.Jaeger)
	}
	return nil
}

// CreateTracer 这里是预留的接口，用于之后可能需要替换jaeger的情况
func CreateTracer() (opentracing.Tracer, io.Closer, error) {
	return NewJaegerByCfg(jCfg).NewTracer(jaegerCfg.Logger(jaegerLog.StdLogger))
}

// InitGlobalTracer 初始化全局tracer
func InitGlobalTracer(logger jaeger.Logger) (io.Closer, error) {
	return NewJaegerByCfg(jCfg).InitGlobalTracer(jCfg.ServiceName, jaegerCfg.Logger(logger))
}

// FinishSpanInCtx 从context中获取span并结束
// 一般情况下不会直接使用
func FinishSpanInCtx(ctx context.Context) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span.Finish()
	}
}

// InjectSpanBytes 自定义的Inject方法，用于将携带Span的Context转换为binary，并通过nats传输
// 将context注入指定的carrier，这里的carrier是bytes.Buffer
// 后续传递[]byte到其他服务
func InjectSpanBytes(sm opentracing.SpanContext) ([]byte, error) {
	if opentracing.GlobalTracer() == nil {
		return nil, ErrGlobalTracerNotInit
	}

	buff := buffPool.Get().(*bytes.Buffer)
	buff.Reset()
	defer buffPool.Put(buff)

	err := opentracing.GlobalTracer().Inject(sm, opentracing.Binary, buff)
	if err != nil {
		return nil, err
	}

	ret := make([]byte, len(buff.Bytes()))
	copy(ret, buff.Bytes())

	return ret, nil
}

// InjectCtxBytes 自定义的Inject方法，用于nats传输
func InjectCtxBytes(ctx context.Context) ([]byte, error) {
	if ctx == nil || opentracing.SpanFromContext(ctx) == nil {
		return nil, nil
	}
	return InjectSpanBytes(opentracing.SpanFromContext(ctx).Context())
}

// ExtractBytes 自定义的Extract方法，用于将nats传输的binary转换回context或者span
// 从carrier读context，这里的carrier是bytes.Reader
// 返回spanContext
func ExtractBytes(bs []byte) (opentracing.SpanContext, error) {
	if opentracing.GlobalTracer() == nil {
		return nil, ErrGlobalTracerNotInit
	}

	rd := readerPool.Get().(*bytes.Reader)
	rd.Reset(bs)
	defer readerPool.Put(rd)

	return opentracing.GlobalTracer().Extract(opentracing.Binary, rd)
}

// ExtractChildSpanFromBytes 从bytes中生成Span
func ExtractChildSpanFromBytes(opName string, bs []byte) (opentracing.Span, error) {
	sm, err := ExtractBytes(bs)
	if err != nil {
		return nil, err
	}

	return opentracing.GlobalTracer().StartSpan(opName, opentracing.ChildOf(sm)), nil
}

func FromContext2SpanBytes(ctx context.Context) []byte {
	if ctx != nil && opentracing.SpanFromContext(ctx) != nil {
		bs, err := InjectCtxBytes(ctx)
		if err != nil {
			tilogs.L().Errorf("FromContext2SpanBytes tracing.InjectCtxBytes err %v", err)
		} else {
			return bs
		}
	}
	return nil
}

// 将tracing注入到ctx，用来grpc传输
func InjectSpanContext(ctx context.Context, tracer opentracing.Tracer, clientSpan opentracing.Span) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	mdWriter := MetadataReaderWriter{md}
	err := tracer.Inject(clientSpan.Context(), opentracing.HTTPHeaders, mdWriter)
	// We have no better place to record an error than the Span itself :-/
	if err != nil {
		tilogs.L().Errorf("InjectSpanContext err %v", err)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// 从ctx中提取tracing，用来从grpc中提取
func ExtractSpanContext(ctx context.Context, tracer opentracing.Tracer) (opentracing.SpanContext, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	return tracer.Extract(opentracing.HTTPHeaders, MetadataReaderWriter{md})
}
