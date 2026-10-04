package nats_cli

import (
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"

	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/golang/protobuf/proto"
)

/*
	nats的主题注册都没加锁，默认认为注册都是启服务器阶段，不在多线程情况下的
*/

type iHandler interface {
	// NewProtoMsg 新建proto message
	// 高频使用的协议，创建协议时可以考虑使用对象池
	NewProtoMsg() proto.Message

	// GetMsg 返回iHandler本身的协议
	GetMsg() proto.Message

	// Handle 逻辑处理部分
	// 如果不需要返回值，或者发送端为SendMsg，返回nil
	Handle(fatherSpan opentracing.Span) proto.Message
}

func GetHandlersMgr() *HandlersMgr {
	if handlersMgr == nil {
		handlersMgr = &HandlersMgr{
			mgr: make(map[string]*handlers, 4),
		}
	}
	return handlersMgr
}

var (
	handlersMgr *HandlersMgr
	mgrLock     sync.RWMutex
)

type HandlersMgr struct {
	mgr map[string]*handlers
}

type handlers struct {
	subHandlers sync.Map // key:msg name
	// l           sync.RWMutex
	util.NoCopy

	subscription *nats.Subscription
}
