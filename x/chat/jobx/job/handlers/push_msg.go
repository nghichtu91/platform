package handler

import (
	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/x/chat/api/nats_cb"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

type PushMsgHandle struct {
	msg *pb.PushMsg
	Cb  nats_cb.JobNatsCallBack
}

func (req *PushMsgHandle) NewProtoMsg() proto.Message {
	req.msg = &pb.PushMsg{}
	return req.msg
}

func (req *PushMsgHandle) GetMsg() proto.Message {
	return req.msg
}

func (req *PushMsgHandle) Handle(fatherSpan opentracing.Span) proto.Message {
	//log.L().Infof("job server nats receive push msg, type:%v, key:%s, server:%s, room:%s",
	//	req.msg.GetType(), req.msg.GetKey(), req.msg.GetServer(), req.msg.GetRoom())

	if fatherSpan != nil {
		span := opentracing.GlobalTracer().StartSpan("GameChatMsg.job", opentracing.ChildOf(fatherSpan.Context()))
		defer span.Finish()
	}

	allmetrics.JobCountAddNewRequest(1)
	req.Cb.Push(fatherSpan, req.msg)
	return &pb.Empty{}
}
