package merger

import (
	"net"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/msg/merger/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	gs *grpc.Server
	ms pb.MergerServer
	l  net.Listener
	wg util.WaitGroupWrapper
}

func NewServer(l net.Listener, ms pb.MergerServer) *Server {
	return &Server{
		ms: ms,
		l:  l,
	}
}

func (s *Server) Run(runMode string) {
	s.gs = grpc.NewServer(grpc.MaxConcurrentStreams(1024),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(
				grpc_recovery.WithRecoveryHandler(func(p interface{}) (err error) {
					tilogs.L().Errorf("UnaryServer recover panic %v", p)
					return err
				})))),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_recovery.StreamServerInterceptor(
				grpc_recovery.WithRecoveryHandler(func(p interface{}) (err error) {
					tilogs.L().Errorf("StreamServer recover panic %v", p)
					return err
				}),
			))))

	if !planx.IsRunProd(runMode) {
		reflection.Register(s.gs)
	}

	pb.RegisterMergerServer(s.gs, s.ms)

	s.wg.Wrap(func() {
		err := s.gs.Serve(s.l)
		if err != nil {
			tilogs.L().Errorf("merger grpc server serve err, %s", err.Error())
		}
	})
}

func (s *Server) Stop() {
	s.gs.Stop()
	s.wg.Wait()
	tilogs.L().Infof("merger stopped")
}
