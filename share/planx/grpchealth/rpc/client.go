package rpc

import (
	"context"

	"google.golang.org/grpc/status"

	"github.com/nghichtu91/platform/share/planx/grpchealth/proto"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// generate protobuffs
//   protoc --go_out=plugins=grpc,import_path=proto:. *.proto

type healthClient struct {
	client proto.HealthClient
	conn   *grpc.ClientConn
}

// NewGrpcHealthClient returns a new grpc Client.
func NewGrpcHealthClient(conn *grpc.ClientConn) Health {
	client := new(healthClient)
	client.client = proto.NewHealthClient(conn)
	client.conn = conn
	return client
}

func (c *healthClient) Close() error {
	return c.conn.Close()
}

func (c *healthClient) Check(ctx context.Context, id string) (bool, error) {
	var res *proto.HealthCheckResponse
	var err error
	req := new(proto.HealthCheckRequest)

	res, err = c.client.Check(ctx, req)
	if err == nil {
		if res.GetStatus() == proto.HealthCheckResponse_SERVING {
			if res.GetServiceId() == id {
				return true, nil
			} else {
				tilogs.L().Warnf("connect grpc server %s not match %s",
					res.GetServiceId(), id)
				return false, status.New(codes.Unavailable, "connect grpc server not match").Err()
			}
		}
		return false, nil
	}
	switch status.Code(err) {
	case
		codes.Aborted,
		codes.DataLoss,
		codes.DeadlineExceeded,
		codes.Internal,
		codes.Unavailable:
		// non-fatal errors
	default:
		return false, err
	}

	return false, err
}
