package grpc

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/nghichtu91/platform/share/planx/tracing"

	"github.com/golang/protobuf/proto"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/cometx/comet"
	"github.com/nghichtu91/platform/share/x/chat/cometx/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// New comet grpc server.
func New(c *config.ChatConfig, s *comet.Server, listener net.Listener) *grpc.Server {
	keepParams := grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle:     time.Second * time.Duration(c.RPCServer.IdleTimeout),
		MaxConnectionAgeGrace: time.Second * time.Duration(c.RPCServer.ForceCloseWait),
		Time:                  time.Second * time.Duration(c.RPCServer.KeepAliveInterval),
		Timeout:               time.Second * time.Duration(c.RPCServer.KeepAliveTimeout),
		MaxConnectionAge:      time.Second * time.Duration(c.RPCServer.MaxLifeTime),
	})
	srv := grpc.NewServer(keepParams)
	pb.RegisterCometServer(srv, &server{s})
	go func() {
		if err := srv.Serve(listener); err != nil {
			panic(err)
		}
	}()
	return srv
}

type server struct {
	srv *comet.Server
}

var _ pb.CometServer = &server{}

// Ping Service
func (s *server) Ping(ctx context.Context, req *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

// Close Service
func (s *server) Close(ctx context.Context, req *pb.Empty) (*pb.Empty, error) {
	// TODO: some graceful close
	return &pb.Empty{}, nil
}

// PushMsg push a message to specified sub keys.
func (s *server) PushMsg(ctx context.Context, req *pb.PushMsgReq) (reply *pb.PushMsgReply, err error) {
	tracer := opentracing.GlobalTracer()
	span, err := tracing.ExtractSpanContext(ctx, tracer)
	var fatherSpan opentracing.Span
	if err == nil {
		fatherSpan = tracer.StartSpan("GameChatMsg.comet.pushMsg", ext.RPCServerOption(span))
		defer fatherSpan.Finish()
	}

	if req.Proto == nil {
		return nil, errors.New(pb.ErrorCode_PushMsgArgError.String())
	}
	if channel := s.srv.Bucket(req.GetKey()).Channel(req.GetKey()); channel != nil {
		channel.Push(&pb.ClientPackageWithSpan{
			Span:          fatherSpan,
			ClientPackage: req.Proto,
		})
		// 如果是封禁消息
		if req.Proto.GetMessageId() == uint32(pb.ChatOperation_S2CForbidden) {
			forbiddenResp := &pb.ForbiddenResp{}
			if protoErr := proto.Unmarshal(req.Proto.RawData, forbiddenResp); protoErr == nil {
				atomic.StoreInt64(&channel.ForbiddenEndTime, forbiddenResp.GetEndTime())
			}
		}
		if req.Proto.GetMessageId() == uint32(pb.ChatOperation_S2CUnForbidden) {
			atomic.StoreInt64(&channel.ForbiddenEndTime, 0)
		}
	}

	return &pb.PushMsgReply{}, nil
}

// Broadcast broadcast msg to all user.
func (s *server) Broadcast(ctx context.Context, req *pb.BroadcastReq) (*pb.BroadcastReply, error) {
	/*
		if req.Proto == nil {
			return nil, errors.ErrBroadCastArg
		}
		// TODO use broadcast queue
		go func() {
			for _, bucket := range s.srv.Buckets() {
				bucket.Broadcast(req.GetProto(), req.ProtoOp)
				if req.Speed > 0 {
					t := bucket.ChannelCount() / int(req.Speed)
					time.Sleep(time.Duration(t) * time.Second)
				}
			}
		}()
	*/
	return &pb.BroadcastReply{}, nil
}

// BroadcastRoom broadcast msg to specified room.
func (s *server) BroadcastRoom(ctx context.Context, req *pb.BroadcastRoomReq) (*pb.BroadcastRoomReply, error) {
	tracer := opentracing.GlobalTracer()
	span, err := tracing.ExtractSpanContext(ctx, tracer)
	var fatherSpan opentracing.Span
	if err == nil {
		fatherSpan = tracer.StartSpan("GameChatMsg.comet.broadcastRoom", ext.RPCServerOption(span))
		defer fatherSpan.Finish()
	}

	if req.Proto == nil || req.GetRoomID() == "" {
		tilogs.L().Errorf("BroadcastRoom fail req.Proto == nil ? %v, req.GetRoomID() = %v", req.Proto == nil, req.GetRoomID())
		return nil, errors.New(pb.ErrorCode_PushRoomMsgArgError.String())
	}

	tilogs.L().Debugf("BroadcastRoom RoomID %v, MessageId %v", req.GetRoomID(), req.Proto.GetMessageId())
	reqWithSpan := &pb.BroadcastRoomReqWithSpan{
		Span:             fatherSpan,
		BroadcastRoomReq: req,
	}
	for _, bucket := range s.srv.Buckets() {
		bucket.BroadcastRoom(reqWithSpan)
	}
	return &pb.BroadcastRoomReply{}, nil
}

// Rooms gets all the room ids for the server.
func (s *server) Rooms(ctx context.Context, req *pb.RoomsReq) (*pb.RoomsReply, error) {
	var (
	//roomIds = make(map[string]bool)
	)
	/*
		for _, bucket := range s.srv.Buckets() {
			for roomID := range bucket.Rooms() {
				roomIds[roomID] = true
			}
		}
	*/
	return &pb.RoomsReply{}, nil
}
