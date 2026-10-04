package util

import (
	"context"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"google.golang.org/grpc"
)

// UnaryClientRecovery RPC 方法的异常保护和日志输出
func UnaryClientRecovery(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	var err error
	defer func() {
		if e := recover(); e != nil {
			tilogs.L().Errorf("UnaryClientRecovery recover panic %v", e)
		}
	}()
	err = invoker(ctx, method, req, reply, cc, opts...)
	return err
}

func StreamClientRecovery(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer,
	opts ...grpc.CallOption) (grpc.ClientStream, error) {
	var err error
	defer func() {
		if e := recover(); e != nil {
			tilogs.L().Errorf("StreamClientRecovery recover panic %v", e)
		}
	}()
	stream, err := streamer(ctx, desc, cc, method, opts...)
	return stream, err
}
