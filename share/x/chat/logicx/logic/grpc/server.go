package grpc

import (
	"context"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"

	//log "github.com/nghichtu91/platform/share/planx/tilogs"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/logicx/config"
	"github.com/nghichtu91/platform/share/x/chat/logicx/logic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// New logic grpc server
func New(c *config.ChatConfig, listener net.Listener, l *logic.Logic) *grpc.Server {
	keepParams := grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle:     time.Second * time.Duration(c.RPCServer.IdleTimeout),
		MaxConnectionAgeGrace: time.Second * time.Duration(c.RPCServer.ForceCloseWait),
		Time:                  time.Second * time.Duration(c.RPCServer.KeepAliveInterval),
		Timeout:               time.Second * time.Duration(c.RPCServer.KeepAliveTimeout),
		MaxConnectionAge:      time.Second * time.Duration(c.RPCServer.MaxLifeTime),
	})
	srv := grpc.NewServer(keepParams)
	pb.RegisterLogicServer(srv, &server{l})

	go func() {
		if err := srv.Serve(listener); err != nil {
			panic(err)
		}
	}()
	return srv
}

type server struct {
	srv *logic.Logic
}

var _ pb.LogicServer = &server{}

// Connect connect a conn.
func (s *server) Connect(ctx context.Context, req *pb.ConnectReq) (*pb.ConnectReply, error) {
	var (
	/*
		reason  string
		endTime int64
	*/
	)
	err := s.srv.Connect(ctx, req.GetServer(), req.GetAcid(), req.GetToken(), req.GetSessionId())
	if err != nil {
		return nil, err
	}

	// 获得禁言信息
	/*
		reason, endTime, err = s.srv.GetForbiddenInfo(ctx, req.GetAcid())
		return &pb.ConnectReply{Acid: req.Acid, Reason: &reason, EndTime: &endTime}, nil
	*/
	return &pb.ConnectReply{Acid: req.Acid, Reason: proto.String(""), EndTime: proto.Int64(0)}, nil
}

func (s *server) GetHistoryMsg(ctx context.Context, req *pb.HistoryMsgReq) (resp *pb.ErrorInfo, err error) {
	resp = &pb.ErrorInfo{}
	clientReq := req.ClienReq
	if clientReq.GetType() == pb.ChatType_Personal {
		err = s.srv.GetPersonHistory(ctx, req.GetFromId(), clientReq.GetTargetId(), clientReq.GetIndexBegin(), clientReq.GetIndexEnd())
	} else if clientReq.GetType() == pb.ChatType_Room {
		err = s.srv.GetRoomHistory(ctx, req.GetFromId(), clientReq.GetTargetId(), clientReq.GetIndexBegin(), clientReq.GetIndexEnd())
	}
	if err != nil {
		resp.Err = proto.String(err.Error())
	}
	return resp, nil
}

// Disconnect disconnect a conn.
func (s *server) DisConnect(ctx context.Context, req *pb.DisconnectReq) (*pb.DisconnectReply, error) {
	has, err := s.srv.Disconnect(ctx, req.GetKey(), req.GetServer(), req.GetSessionId())
	if err != nil {
		return &pb.DisconnectReply{}, err
	}
	return &pb.DisconnectReply{Has: &has}, nil
}

// Heartbeat beartbeat a conn.
func (s *server) Heartbeat(ctx context.Context, req *pb.HeartbeatReq) (*pb.Empty, error) {
	/*
		if err := s.srv.Heartbeat(ctx, req.GetKey(), req.GetServer()); err != nil {
			return &pb.Empty{}, err
		}
	*/
	return &pb.Empty{}, nil
}

// 房间销毁
func (s *server) RoomDestroy(ctx context.Context, req *pb.RoomDestroyReq) (*pb.ErrorInfo, error) {
	resp := &pb.ErrorInfo{}
	//err := s.srv.RoomDestroy(ctx, req.RoomIds)
	//if err != nil {
	//	return nil, err
	//}

	return resp, nil
}

func (s *server) Translate(ctx context.Context, req *pb.TranslateWithTarget) (*pb.ErrorInfo, error) {
	resp := &pb.ErrorInfo{}
	_ = s.srv.PushTranslate(ctx, req)
	allmetrics.TranslateCountAddNewRequest(1)
	return resp, nil
}
