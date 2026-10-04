package nats_cli

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"google.golang.org/protobuf/proto"
)

var (
	ErrorTimeoutShouldMoreThanUpTimeout = errors.New("timeout should more than UpTimeout")
	ErrorNilProtoMessage                = errors.New("nil proto message")
)

type Resp struct {
	shardID uint32
	resp    proto.Message
}

// BatchReqResp2Gamex 批量将请求发送到gamex，并等所有请求都返回后，汇集结果返回
// 这个接口适用于同一个req发给所有gamex的情况
func BatchReqResp2Gamex(gid uint, shards []uint32, req, resp proto.Message, timeout time.Duration) (map[uint32]proto.Message, error) {
	return batchReqResp2Gamex(gid, nil, shards, req, resp, timeout)
}

// BatchRequestsResp2Gamex 批量将请求发送到gamex，并等所有请求都返回后，汇集结果返回
// 这个接口适用于每个gamex有参数不同的req的情况
func BatchRequestsResp2Gamex(gid uint, req map[uint32]proto.Message, resp proto.Message, timeout time.Duration) (map[uint32]proto.Message, error) {
	return batchReqResp2Gamex(gid, req, nil, nil, resp, timeout)
}

// BatchReqResp2Gamex 批量将请求发送到gamex，并等所有请求都返回后，汇集结果返回
// resp: 会通过Clone resp为每个服生成对应resp
// 如果超时，会返回已经收到的结果
func batchReqResp2Gamex(gid uint, requests map[uint32]proto.Message, shards []uint32, req, resp proto.Message, timeout time.Duration) (map[uint32]proto.Message, error) {
	if timeout <= planx.UpTimeOut {
		return nil, ErrorTimeoutShouldMoreThanUpTimeout
	}

	if (req == nil && requests == nil) || resp == nil {
		return nil, ErrorNilProtoMessage
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	size := len(shards)
	if len(requests) > size {
		size = len(requests)
	}

	gidStr := strconv.Itoa(int(gid))
	retChan := make(chan Resp, size)
	ret := make(map[uint32]proto.Message, size)

	if len(requests) > 0 { // 多个请求群发
		for sid, msg := range requests {
			sid := sid
			msg := msg
			if msg == nil {
				continue
			}
			go reqRespWithBackoffContext(ctx,
				GamexSerSubj(gidStr, strconv.Itoa(int(sid))),
				msg, Resp{shardID: sid, resp: proto.Clone(resp)}, retChan)
		}
	} else { // 单个请求群发
		for _, sid := range shards {
			sid := sid
			go reqRespWithBackoffContext(ctx,
				GamexSerSubj(gidStr, strconv.Itoa(int(sid))),
				req, Resp{shardID: sid, resp: proto.Clone(resp)}, retChan)
		}
	}

	for {
		select {
		case r := <-retChan:
			ret[r.shardID] = r.resp
			if len(ret) == size {
				return ret, nil
			}
		case <-ctx.Done():
			return ret, ctx.Err()
		}
	}
}

func reqRespWithBackoffContext(ctx context.Context, scope []string, req proto.Message, resp Resp, ret chan Resp) {
	// 使用默认的重试策略
	b := backoff.NewExponentialBackOff()
	// 使用传入的context
	bc := backoff.WithContext(b, ctx)

	err := backoff.Retry(func() error {
		return RequestResponse(scope, req, resp.resp)
	}, bc)
	if err != nil {
		tilogs.L().Errorf("reqRespWithBackoffContext err %s, req %v, scope %v", err.Error(), req, scope)
		return
	}

	ret <- resp
}
