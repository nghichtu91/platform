package util

import (
	"time"

	"google.golang.org/grpc"
	backoff2 "google.golang.org/grpc/backoff"
)

// 文档: https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md

func GRPCWithConnectParams() grpc.DialOption {
	return grpc.WithConnectParams(GRPCConnectParams())
}

func GRPCConnectParams() grpc.ConnectParams {
	return grpc.ConnectParams{
		// 实现: Exponential.Backoff
		Backoff: backoff2.Config{
			BaseDelay:  1 * time.Second,  // 1. 首次进行失败重试的时间
			Multiplier: 1.2,              // 2. 失败后重试的避退指数. 比如失败3次, 那么间隔为 BaseDelay*(Multiplier^3)
			Jitter:     0.2,              // 3. 随机散列的时长 设2计算的时间为t, 散列后的时长为  t*(1 + Jitter*(rand.Float64()*2-1))
			MaxDelay:   10 * time.Second, // 4. 等待时长的上限 设3计算的时间为t2, 然后会取 min(MaxDelay, t2)
		},
		// 使用的地方: addrConn.resetTransport
		// 设通过Backoff计算出来的等待时长为t3
		MinConnectTimeout: 5 * time.Second, // 最终值取max(MinConnectTimeout, t3). 等同net.DialContext, 设置的超时时长.
	}
}
