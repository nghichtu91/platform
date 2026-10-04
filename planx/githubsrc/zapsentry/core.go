package zapsentry

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/getsentry/raven-go"
	"go.uber.org/zap/zapcore"
)

const (
	traceContextLines = 0 // 去处上下文
	traceSkipFrames   = 4 // 增加一些skip caller
)

func NewCore(cfg Configuration, factory SentryClientFactory) (zapcore.Core, error) {
	client, err := factory()
	if err != nil {
		return zapcore.NewNopCore(), err
	}
	return &core{
		client:       client,
		cfg:          &cfg,
		LevelEnabler: cfg.Level,
		fields:       make(map[string]interface{}),
	}, nil
}

func (c *core) With(fs []zapcore.Field) zapcore.Core {
	cc := c.with(fs)

	// 转换成tags(以最新的kv为准)
	cc.cfg = cc.cfg.clone()
	enc := encField(fs)
	for key, field := range enc.Fields {
		cc.cfg.Tags[key] = fmt.Sprintf("%v", field)
	}
	return cc
}

func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.cfg.Level.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *core) Write(ent zapcore.Entry, fs []zapcore.Field) error {
	clone := c.with(fs)

	//var extraTags map[string]string
	//
	//// 将附加数据作为tag
	//if clone.fields != nil {
	//	extraTags = make(map[string]string, len(clone.fields))
	//	for key, value := range clone.fields {
	//		_, ok := value.(string)
	//		if ok {
	//			extraTags[strings.ToLower(key)] = value.(string)
	//		}
	//	}
	//}

	packet := &raven.Packet{
		Message:   ent.Message,
		Timestamp: raven.Timestamp(ent.Time),
		Level:     ravenSeverity(ent.Level),
		Platform:  "Golang",
		Extra:     clone.fields,
	}

	//if extraTags != nil {
	//	packet.AddTags(extraTags)
	//}

	if !c.cfg.DisableStacktrace {
		trace := raven.NewStacktrace(traceSkipFrames, traceContextLines, nil)
		if trace != nil {
			for _, frame := range trace.Frames {
				if frame == nil {
					continue
				}
				// 不需要InApp标识, 否则可能会导致sentry识别错误
				frame.InApp = false
			}
			packet.Interfaces = append(packet.Interfaces, trace)
		}
	}

	_, _ = c.client.Capture(packet, c.cfg.Tags)

	// We may be crashing the program, so should flush any buffered events.
	if ent.Level > zapcore.ErrorLevel {
		c.client.Wait()
	}
	return nil
}

func (c *core) Sync() error {

	var waitSignal = make(chan struct{})

	var ctx, cancel = context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("sentry sync panic:", r)
			}
			close(waitSignal)
		}()
		c.client.Wait()
	}()

	select {
	case <-ctx.Done():
		log.Println("sentry sync timeout")
	case <-waitSignal:
		log.Println("sentry sync success")
	}
	return nil
}

func (c *core) with(fs []zapcore.Field) *core {
	// Copy our map.
	m := make(map[string]interface{}, len(c.fields))
	for k, v := range c.fields {
		m[k] = v
	}

	// Add fields to an in-memory encoder.
	enc := encField(fs)

	// Merge the two maps.
	for k, v := range enc.Fields {
		m[k] = v
	}

	return &core{
		client:       c.client,
		cfg:          c.cfg,
		fields:       m,
		LevelEnabler: c.LevelEnabler,
	}
}

func encField(fs []zapcore.Field) *zapcore.MapObjectEncoder {
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fs {
		f.AddTo(enc)
	}
	return enc
}

type ClientGetter interface {
	GetClient() *raven.Client
}

func (c *core) GetClient() *raven.Client {
	return c.client
}

type core struct {
	client *raven.Client
	cfg    *Configuration
	zapcore.LevelEnabler

	fields map[string]interface{}
}
