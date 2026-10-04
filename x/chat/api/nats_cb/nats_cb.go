package nats_cb

import (
	"github.com/opentracing/opentracing-go"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

type JobNatsCallBack interface {
	Push(fatherSpan opentracing.Span, msg *pb.PushMsg)
}

type LogicNatsCallBack interface {
	Push(fatherSpan opentracing.Span, msg *pb.GamexChatMsg) *pb.GamexChatMsgRet
}
