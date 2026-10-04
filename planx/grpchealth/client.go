package grpchealth

import (
	"context"
	"time"

	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"google.golang.org/grpc"

	"github.com/nghichtu91/platform/share/planx/grpchealth/rpc"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type NotifyHealth func(isHealth bool)

/*
CheckGRPCHealth
grpc health检查的client端
参数：

	addr: ip:host
	notify: 是否health的回调函数
*/
func CheckGRPCHealth(addr, id string, notify NotifyHealth, quitChan chan struct{}) {
	go func() {
		defer tilogs.PanicCatcher("CheckGRPCHealth addr %s, id %s panic", addr, id)
		// Set up a connection to the server.
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		L := tilogs.L().With("check-addr", addr, "check-id", id)
		if err != nil {
			L.Errorf("CheckGRPCHealth did not connect")
			notify(false)
			return
		}
		defer func(conn *grpc.ClientConn) { _ = conn.Close() }(conn)

		client := rpc.NewGrpcHealthClient(conn)

		L.Infof("Start CheckGRPCHealth")
		timerChan := timeutil.TimerSec.After(time.Second * 5)
		for {
			select {
			case <-timerChan:
				ok, err := client.Check(context.Background(), id)
				if !ok || err != nil {
					L.Errorf("CheckGRPCHealth can't connect grpc server: err %v, code: %v", err, status.Code(err))
					notify(false)
				} else {
					notify(true)
				}
				timerChan = timeutil.TimerSec.After(time.Second * 5)
			case <-quitChan:
				L.Infof("CheckGRPCHealth quit")
				notify(false)
				return
			}
		}
	}()
}
