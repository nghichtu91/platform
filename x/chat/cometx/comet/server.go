package comet

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/zhenjl/cityhash"
	"github.com/nghichtu91/platform/share/planx/rand_pool"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	dis "github.com/nghichtu91/platform/share/x/chat/api/discovery"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/cometx/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	minServerHeartbeat = time.Minute * 10
	maxServerHeartbeat = time.Minute * 30
	// grpc options
	grpcInitialWindowSize     = 1 << 24
	grpcInitialConnWindowSize = 1 << 24
	grpcMaxSendMsgSize        = 1 << 13 // 8192 考虑有可能有语音
	grpcMaxCallMsgSize        = 1 << 13 //
	grpcKeepAliveTime         = time.Second * 10
	grpcKeepAliveTimeout      = time.Second * 3
	grpcBackoffMaxDelay       = time.Second * 3
)

func newLogicClient(addr string) *grpc.ClientConn {
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
		}...)
	if err != nil {
		panic(err)
	}

	log.L().Infof("grpc comet client at %s", config.Cfg.RPCClient.Addr)
	return conn
}

// Server is comet server.
type Server struct {
	round     *Round    // accept round store
	buckets   []*Bucket // subkey bucket
	bucketIdx uint32

	tcpListener net.Listener
	serverID    string
	rpcConn     *grpc.ClientConn
	rpcClient   pb.LogicClient
	rpcLock     sync.RWMutex // lock rpcClient and rpcConn

	ConnNum     int32
	ConnNumLock sync.RWMutex

	isOpen bool
}

// NewServer returns a new Server.
func NewServer() *Server {
	s := &Server{
		round: NewRound(),
	}

	// init bucket
	s.buckets = make([]*Bucket, config.Cfg.Bucket.Size)
	s.bucketIdx = uint32(config.Cfg.Bucket.Size)
	for i := 0; i < config.Cfg.Bucket.Size; i++ {
		s.buckets[i] = NewBucket(&config.Cfg)
	}
	// s.serverID = c.Env.Host
	// go s.onlineproc()
	return s
}

// Buckets return all buckets.
func (s *Server) Buckets() []*Bucket {
	return s.buckets
}

// Bucket get the bucket by subkey.
func (s *Server) Bucket(subKey string) *Bucket {
	idx := cityhash.CityHash32([]byte(subKey), uint32(len(subKey))) % s.bucketIdx
	log.L().Debugf("%s hit channel bucket index: %d use cityhash", subKey, idx)
	return s.buckets[idx]
}

// RandServerHearbeat rand server heartbeat.
func (s *Server) RandServerHearbeat() time.Duration {
	return minServerHeartbeat + time.Duration(rand_pool.Int63n(int64(maxServerHeartbeat-minServerHeartbeat)))
}

// Close close the server.
func (s *Server) Close() (err error) {
	if s.tcpListener != nil {
		// s.tcpListener.Close()
	}
	s.rpcLock.RLock()
	defer s.rpcLock.RUnlock()
	if s.rpcConn != nil {
		s.rpcConn.Close()
	}
	return
}

// OnServerAdd 发现logic
func (s *Server) OnServerAdd(serverId string, serverInfo *dis.LogicServerInfo) {
	log.L().Infof("Discovery New logic server id is %s, info is %v", serverId, serverInfo)

	s.rpcLock.Lock()
	defer s.rpcLock.Unlock()
	// 如果连接还在 重新连接
	if s.rpcConn != nil && s.rpcClient != nil {
		_ = s.rpcConn.Close()
	}
	s.rpcConn = newLogicClient(serverInfo.PrivateAddr)
	s.rpcClient = pb.NewLogicClient(s.rpcConn)
}

// OnServerDel 发现logic
func (s *Server) OnServerDel(serverId string) {
	s.rpcLock.Lock()
	defer s.rpcLock.Unlock()
	if s.rpcConn != nil {
		_ = s.rpcConn.Close()
	}
	s.rpcConn = nil
	s.rpcClient = nil
}

func (s *Server) RPCClient() pb.LogicClient {
	s.rpcLock.RLock()
	defer s.rpcLock.RUnlock()
	return s.rpcClient
}
