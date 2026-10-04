package grpchealth

import (
	"context"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/nghichtu91/platform/share/planx/grpchealth/proto"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

/*
GRPCHealthServer
grpc health检查的server端
参数lis为监听建立好的listener，由于外面可能需要再一个范围试端口，所以直接传入net.Listener
*/
func GRPCHealthServer(lis net.Listener, id string) {
	s := grpc.NewServer()
	srv := newServer(id)
	proto.RegisterHealthServer(s, srv)
	// Register reflection service on gRPC server.
	reflection.Register(s)
	tilogs.L().Infof("GRPCHealthServer gRPC server %s is running on %s", id, lis.Addr().String())
	go func() {
		if err := s.Serve(lis); err != nil {
			tilogs.L().Errorf("GRPCHealthServer failed to serve %s: %s err %v", id, lis.Addr().String(), err)
		}
	}()
}

// server is used to implement gorush grpc server.
type server struct {
	id string
	mu sync.Mutex
	// statusMap stores the serving status of the services this server monitors.
	statusMap map[string]proto.HealthCheckResponse_ServingStatus
}

// newServer returns a new server.
func newServer(id string) *server {
	return &server{
		id:        id, // 服务的id
		statusMap: make(map[string]proto.HealthCheckResponse_ServingStatus),
	}
}

// Check implements `service Health`.
func (s *server) Check(_ context.Context, in *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.Service == "" {
		// check the server overall health state.
		return &proto.HealthCheckResponse{
			Status:    proto.HealthCheckResponse_SERVING,
			ServiceId: s.id,
		}, nil
	}
	if state, ok := s.statusMap[in.Service]; ok {
		return &proto.HealthCheckResponse{
			Status:    state,
			ServiceId: s.id,
		}, nil
	}
	return nil, status.Error(codes.NotFound, "unknown service")
}
