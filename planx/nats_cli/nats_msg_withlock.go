package nats_cli

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx/nats_cli/pb"
	"github.com/nghichtu91/platform/share/planx/pbbuff"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tracing"
)

var (
	ErrNotInit = errors.New("nats not init")
)

func (mgr *HandlersMgr) RegWithLock(scope []string, h ...iHandler) {
	mgr.regWithLock(scope, false, h...)
}

func (mgr *HandlersMgr) RegQueueWithLock(scope []string, h ...iHandler) {
	mgr.regWithLock(scope, true, h...)
}

func (mgr *HandlersMgr) RegNoSubWithLock(scope []string, h iHandler) (*nats.Subscription, error) {
	return mgr.regNoSubWithLock(scope, h, false)
}

func (mgr *HandlersMgr) RegQueueNoSubWithLock(scope []string, h iHandler) (*nats.Subscription, error) {
	return mgr.regNoSubWithLock(scope, h, true)
}

// UnRegWithLock 注销特定主题的所有订阅
// 只对应RegWithLock和RegQueueWithLock
func (mgr *HandlersMgr) UnRegWithLock(scope []string) error {
	subj := strings.Join(scope, ".")

	mgrLock.Lock()
	defer mgrLock.Unlock()
	handlers, ok := mgr.mgr[subj]
	if ok {
		delete(mgr.mgr, subj)
		if handlers.subscription != nil {
			return handlers.subscription.Unsubscribe()
		}
	}
	tilogs.L().Infof("nats unreg %s", subj)
	return nil
}

func (mgr *HandlersMgr) regWithLock(scope []string, isQueue bool, h ...iHandler) {
	subj := strings.Join(scope, ".")

	mgrLock.Lock()
	subHandlersInfo, ok := mgr.mgr[subj]
	if !ok {
		subHandlersInfo = &handlers{}
		mgr.mgr[subj] = subHandlersInfo
	}
	mgrLock.Unlock()

	for _, handler := range h {
		subSubj := reflect.TypeOf(handler.GetMsg()).Elem().String()
		// subHandlersInfo.l.Lock()
		// defer subHandlersInfo.l.Unlock()
		_, ok := subHandlersInfo.subHandlers.Load(subSubj)
		if ok {
			// panic(fmt.Errorf("nats HandlersMgr Reg subSubj repeated, %s %s", subj, subSubj))
			tilogs.L().Errorf("nats HandlersMgr Reg subSubj repeated, %s %s", subj, subSubj) // 防止重复nats注册，上层逻辑的报错
		}
		subHandlersInfo.subHandlers.Store(subSubj, handler)
		tilogs.L().Infof("nats reg %s %s", subj, subSubj)
	}

	if !ok {
		if isQueue {
			mgr.registerMsgQueueGroupHandlerWithLock(subj, subHandlersInfo)
		} else {
			mgr.registerMsgHandlerWithLock(subj, subHandlersInfo)
		}
	}
}

func (mgr *HandlersMgr) registerMsgHandlerWithLock(subj string, hs *handlers) {
	if GetNatsClient() == nil {
		tilogs.L().Errorf("nats registerMsgHandler GetNatsClient() = nil")
		return
	}

	hs.subscription, _ = natsClient.Subscribe(subj, func(msg *nats.Msg) {
		defer tilogs.PanicCatcher("nats Handle resp err.subj=%s", subj)

		mgr.msgHandlersWithLock(subj, hs, msg)
	})
}

func (mgr *HandlersMgr) registerMsgQueueGroupHandlerWithLock(subj string, hs *handlers) {
	if GetNatsClient() == nil {
		tilogs.L().Errorf("nats registerMsgQueueGroupHandler GetNatsClient() = nil")
		return
	}
	hs.subscription, _ = natsClient.QueueSubscribe(subj, "queue", func(msg *nats.Msg) {
		defer tilogs.PanicCatcher("nats Handle queue resp err.subj=%s", subj)

		mgr.msgHandlersWithLock(subj, hs, msg)
	})
}

func (mgr *HandlersMgr) msgHandlersWithLock(subj string, hs *handlers, msg *nats.Msg) {
	var (
		w          *pb.NatsWrapper
		buff       *bytes.Buffer
		replyBytes []byte
		err        error
		span       opentracing.Span
	)

	// 下一步Unmarshal会直接Reset，这里就不需要额外的Reset了
	w = pbPool.Get().(*pb.NatsWrapper)
	defer func() {
		w.Reset()
		pbPool.Put(w)
	}()

	err = proto.UnmarshalMerge(msg.Data, w)
	if err != nil {
		tilogs.L().Errorf("nats msgHandlers [%s] Unmarshal err=%v", subj, err)
		return
	}

	// tracing
	if len(w.Ctx) > 0 {
		span, err = tracing.ExtractChildSpanFromBytes("nats.handler."+w.Name, w.GetCtx())
		if err != nil {
			tilogs.L().Errorf("msgHandlersWithLock ExtractChildSpanFromBytes err %v", err)
		} else {
			defer span.Finish()
		}
	}

	h, ok := hs.subHandlers.Load(w.Name)
	if !ok {
		tilogs.L().Errorf("nats msgHandlers [%s][%s] not found", subj, w.Name)
		return
	}
	handler := h.(iHandler)

	err = proto.UnmarshalMerge(w.Content, handler.NewProtoMsg())
	if err != nil {
		tilogs.L().Errorf("nats msgHandlers [%s][%s] Unmarshal err=%v", subj, w.Name, err)
		return
	}

	resp := handler.Handle(span)
	if resp == nil {
		return
	}

	buff = pbbuff.GetBuffer()
	defer pbbuff.PutBuffer(buff, &replyBytes)

	// 这里需要把protoMessageV1转为V2
	replyBytes, err = pmo.MarshalAppend(buff.Bytes(), proto.MessageV2(resp))
	if err != nil {
		tilogs.L().Errorf("nats msgHandlers [%s] Marshal err=%v", subj, err)
		return
	}

	if err := natsClient.Publish(msg.Reply, w.Name+"resp", replyBytes); err != nil {
		if err != nats.ErrBadSubject {
			tilogs.L().Alarm("nats msgHandlersWithLock Publish，handler %s err=%v", w.Name, err)
		}
	}

	// msg.Respond(replyBytes)
}

func (mgr *HandlersMgr) regNoSubWithLock(scope []string, h iHandler, isQueue bool) (*nats.Subscription, error) {
	if GetNatsClient() == nil {
		tilogs.L().Errorf("nats regNoSubWithLock GetNatsClient() = nil")
		return nil, ErrNotInit
	}

	subj := strings.Join(scope, ".")

	mgrLock.RLock()
	_, newSubj := mgr.mgr[subj]
	if newSubj {
		mgrLock.RUnlock()
		return nil, fmt.Errorf("%w,suj:%s", errors.New("subj reg repeated"), subj)
	}
	mgrLock.RUnlock()

	if isQueue {
		return natsClient.QueueSubscribe(subj, "queue", func(msg *nats.Msg) {
			defer tilogs.PanicCatcher("nats Handle queue resp err.subj=%s", subj)

			mgr.msgHandleWithLock(subj, h, msg)
		})
	} else {
		return natsClient.Subscribe(subj, func(msg *nats.Msg) {
			defer tilogs.PanicCatcher("nats Handle resp err.subj=%s", subj)

			mgr.msgHandleWithLock(subj, h, msg)
		})
	}
}

func (mgr *HandlersMgr) msgHandleWithLock(subj string, h iHandler, msg *nats.Msg) {
	var (
		w          *pb.NatsWrapper
		buff       *bytes.Buffer
		replyBytes []byte
		err        error
		span       opentracing.Span
	)

	// t := time.Now().UnixNano()
	// defer metrics.ReqTimeStatistics("natsMsgHandle", reflect.TypeOf(h).String(), t)

	w = pbPool.Get().(*pb.NatsWrapper)
	defer func() {
		w.Reset()
		pbPool.Put(w)
	}()

	err = proto.UnmarshalMerge(msg.Data, w)
	if err != nil {
		tilogs.L().Errorf("nats msgHandle [%s] Unmarshal err=%v", subj, err)
		return
	}

	// tracing
	if len(w.Ctx) > 0 {
		span, err = tracing.ExtractChildSpanFromBytes("nats.handler."+w.Name, w.GetCtx())
		if err != nil {
			tilogs.L().Errorf("msgHandleWithLock tracing.ExtractChildSpanFromBytes err %v", err)
		} else {
			defer span.Finish()
		}
	}

	err = proto.UnmarshalMerge(w.Content, h.NewProtoMsg())
	if err != nil {
		tilogs.L().Errorf("nats msgHandle [%s][%s] Unmarshal err=%v", subj, w.Name, err)
		return
	}

	resp := h.Handle(span)
	if resp == nil {
		return
	}

	buff = pbbuff.GetBuffer()
	defer pbbuff.PutBuffer(buff, &replyBytes)

	replyBytes, err = pmo.MarshalAppend(buff.Bytes(), proto.MessageV2(resp))
	if err != nil {
		tilogs.L().Errorf("nats msgHandle [%s] Marshal err=%v", subj, err)
		return
	}

	if err := natsClient.Publish(msg.Reply, "", replyBytes); err != nil {
		if err == nats.ErrBadSubject {
			tilogs.L().Debugf("nats msgHandleWithLock Publish, handler %s subj nil", w.Name)
		} else {
			tilogs.L().Alarm("nats msgHandleWithLock Publish，handler %s err=%v", w.Name, err)
		}
	}

	// msg.Respond(replyBytes)
}
