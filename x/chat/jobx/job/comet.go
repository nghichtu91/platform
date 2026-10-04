package job

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/nghichtu91/platform/share/planx/tracing"

	"github.com/opentracing/opentracing-go"

	log "github.com/nghichtu91/platform/share/planx/tilogs"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/jobx/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var (
	// grpc options
	grpcKeepAliveTime    = time.Duration(10) * time.Second
	grpcKeepAliveTimeout = time.Duration(3) * time.Second
	grpcBackoffMaxDelay  = time.Duration(3) * time.Second
	grpcMaxSendMsgSize   = 1 << 15
	grpcMaxCallMsgSize   = 1 << 15
)

const (
	// grpc options
	grpcInitialWindowSize     = 1 << 24
	grpcInitialConnWindowSize = 1 << 24
)

// 作为comet的客户端 连接comet grpc
func newCometClient(addr string) (pb.CometClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr,
		[]grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithInitialWindowSize(grpcInitialWindowSize),
			grpc.WithInitialConnWindowSize(grpcInitialConnWindowSize),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcMaxCallMsgSize)),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(grpcMaxSendMsgSize)),
			grpc.WithBackoffMaxDelay(grpcBackoffMaxDelay),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                grpcKeepAliveTime,
				Timeout:             grpcKeepAliveTimeout,
				PermitWithoutStream: true,
			}),
		}...,
	)
	if err != nil {
		return nil, err
	}
	return pb.NewCometClient(conn), err
}

// Comet is a comet.
type Comet struct {
	serverID      string
	client        pb.CometClient
	pushChan      []chan *pb.PushMsgReqWithSpan
	roomChan      []chan *pb.BroadcastRoomReqWithSpan
	broadcastChan chan *pb.BroadcastReq
	pushChanNum   uint64
	roomChanNum   uint64
	routineSize   uint64

	ctx    context.Context
	cancel context.CancelFunc
}

// NewComet new a comet.
func NewComet(c *config.ChatConfig, serverId, grpcAddr string) (*Comet, error) {
	cmt := &Comet{
		serverID:      serverId,
		pushChan:      make([]chan *pb.PushMsgReqWithSpan, c.Comet.RoutineSize),
		roomChan:      make([]chan *pb.BroadcastRoomReqWithSpan, c.Comet.RoutineSize),
		broadcastChan: make(chan *pb.BroadcastReq, c.Comet.RoutineSize),
		routineSize:   uint64(c.Comet.RoutineSize),
	}
	if grpcAddr == "" {
		return nil, fmt.Errorf("invalid grpc address:%v", grpcAddr)
	}
	var err error
	if cmt.client, err = newCometClient(grpcAddr); err != nil {
		return nil, err
	}
	cmt.ctx, cmt.cancel = context.WithCancel(context.Background())

	// cmt.pushChan cmt.roomChan都是数组 这里每一个元素建立了一个chan
	for i := 0; i < c.Comet.RoutineSize; i++ {
		cmt.pushChan[i] = make(chan *pb.PushMsgReqWithSpan, c.Comet.RoutineChan)
		cmt.roomChan[i] = make(chan *pb.BroadcastRoomReqWithSpan, c.Comet.RoutineChan)
		go cmt.process(cmt.pushChan[i], cmt.roomChan[i], cmt.broadcastChan)
	}
	return cmt, nil
}

// Push push a user message.
func (c *Comet) Push(arg *pb.PushMsgReqWithSpan) (err error) {
	if c == nil {
		return
	}

	idx := atomic.AddUint64(&c.pushChanNum, 1) % c.routineSize
	select {
	case c.pushChan[idx] <- arg:
	default:
		log.L().Errorf("Comet.Push Msg Miss Key:%s", arg.GetKey())
	}
	return
}

// BroadcastRoom broadcast a room message.
func (c *Comet) BroadcastRoom(arg *pb.BroadcastRoomReqWithSpan) (err error) {
	if c == nil {
		return
	}

	idx := atomic.AddUint64(&c.roomChanNum, 1) % c.routineSize
	select {
	case c.roomChan[idx] <- arg:
	default:
		log.L().Errorf("Comet.BroadcastRoom Msg Miss RoomId:%s", arg.GetRoomID())
	}
	return
}

// Broadcast broadcast a message.
func (c *Comet) Broadcast(arg *pb.BroadcastReq) (err error) {
	if c == nil {
		return
	}
	
	c.broadcastChan <- arg
	return
}

func (c *Comet) process(pushChan chan *pb.PushMsgReqWithSpan, roomChan chan *pb.BroadcastRoomReqWithSpan, broadcastChan chan *pb.BroadcastReq) {
	for {
		select {
		case broadcastArg := <-broadcastChan:
			_, err := c.client.Broadcast(context.Background(), &pb.BroadcastReq{
				Proto:   broadcastArg.Proto,
				ProtoOp: broadcastArg.ProtoOp,
				Speed:   broadcastArg.Speed,
			})
			if err != nil {
				log.L().Errorf("c.client.Broadcast(%s, reply) serverId:%s error(%v)", broadcastArg, c.serverID, err)
			}
		case roomArg := <-roomChan:
			ctx := context.Background()
			if roomArg.Span != nil {
				ctx = tracing.InjectSpanContext(ctx, opentracing.GlobalTracer(), roomArg.Span)
			}

			_, err := c.client.BroadcastRoom(ctx, &pb.BroadcastRoomReq{
				RoomID: roomArg.RoomID,
				Proto:  roomArg.Proto,
			})
			if err != nil {
				log.L().Errorf("c.client.BroadcastRoom(%s, reply) serverId:%s error(%v)", roomArg, c.serverID, err)
			}
			tracing.FinishSpanInCtx(ctx)
		case pushArg := <-pushChan:
			ctx := context.Background()
			if pushArg.Span != nil {
				ctx = tracing.InjectSpanContext(ctx, opentracing.GlobalTracer(), pushArg.Span)
			}

			_, err := c.client.PushMsg(ctx, &pb.PushMsgReq{
				Key:     pushArg.Key,
				Proto:   pushArg.Proto,
				ProtoOp: pushArg.ProtoOp,
			})
			if err != nil {
				log.L().Errorf("c.client.PushMsg(%s, reply) serverId:%s error(%v)", pushArg, c.serverID, err)
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// Close close the resouces.
func (c *Comet) Close() (err error) {
	finish := make(chan bool)
	go func() {
		for {
			n := len(c.broadcastChan)
			for _, ch := range c.pushChan {
				n += len(ch)
			}
			for _, ch := range c.roomChan {
				n += len(ch)
			}
			if n == 0 {
				finish <- true
				return
			}
			time.Sleep(time.Second)
		}
	}()
	select {
	case <-finish:
		log.L().Infof("close comet: finish", c.serverID)
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("close comet(server:%s push:%d room:%d broadcast:%d) timeout", c.serverID, len(c.pushChan), len(c.roomChan), len(c.broadcastChan))
	}
	c.cancel()
	return
}
