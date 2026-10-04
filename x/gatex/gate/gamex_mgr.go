package gate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	atomic2 "go.uber.org/atomic"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/nghichtu91/platform/share/planx/client"
	"github.com/nghichtu91/platform/share/planx/etcd"
	discovery "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/gatex/config"

	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
)

type gamex_mgr struct {
	gate_server  *GateServer
	games        map[uint]*gamex_info
	discoveryMgr *discovery.DiscoveryMgr

	lock sync.RWMutex

	// 检查和 game 之间的 ping 和 push 的请求
	running      atomic2.Bool // 是否正在检查
	lastCheckEnd time.Time    // 上次检查结束时间
	gateCheckReq
}

type gateCheckReq struct {
	pingCheckReq *pb.CtlReq
	pushCheckReq *pb.CtlReq
}

func NewGameMgr(gate_server *GateServer) *gamex_mgr {
	return &gamex_mgr{
		gate_server: gate_server,
		games:       make(map[uint]*gamex_info, 128),
	}
}

func (mgr *gamex_mgr) StartGameMgr(internalIpPort string) error {
	if err := mgr.discovery(internalIpPort); err != nil {
		return err
	}
	mgr.pingCheckReq = &pb.CtlReq{
		Typ:   pb.CtlTyp_PingCheck,
		Param: config.Cfg.GateConfig.ServerId,
	}
	mgr.pushCheckReq = &pb.CtlReq{
		Typ:   pb.CtlTyp_PushCheck,
		Param: config.Cfg.GateConfig.ServerId,
	}
	return nil
}

func (mgr *gamex_mgr) getConn(acid string) (*gpr_conn, error) {
	account, err := db.ParseAccount(acid)
	if err != nil {
		return nil, err
	}
	mgr.lock.RLock()
	game, ok := mgr.games[account.ShardId]
	if !ok {
		mgr.lock.RUnlock()
		return nil, fmt.Errorf("gamex_mgr game sid %d not found", account.ShardId)
	}
	physSid := game.getPhysicsSid()
	logicSid := game.etcd_info.Sid
	if physSid != logicSid {
		game, ok = mgr.games[physSid]
		if !ok {
			mgr.lock.RUnlock()
			return nil, fmt.Errorf("gamex_mgr sid %d game %d physics %d not found",
				account.ShardId, logicSid, physSid)
		}
	}
	mgr.lock.RUnlock()
	return game.getConn(), nil
}

func (mgr *gamex_mgr) OnAddService(info discovery.ServiceInfo) {
	ok, gid, logicSid := mgr.commonCheck(info.SerId)
	if !ok {
		return
	}

	extraInfo := parseGamexExtra(info)
	if extraInfo.PhysicsSid == 0 {
		extraInfo.PhysicsSid = logicSid
	}

	physicsSid := extraInfo.PhysicsSid

	// 鉴于可能存在 race 问题, 每次都是用一个新的对象, 并在初始化完成之后再写入到 map 中
	logicInfo := newGamexInfoWithConn(newGamexConnInfo(logicSid))

	// 更新基础信息
	etcdInfo := &logicInfo.etcd_info
	etcdInfo.Gid = gid
	etcdInfo.Sid = logicSid
	etcdInfo.PhysicsSid = extraInfo.PhysicsSid
	etcdInfo.IsMaintaining = extraInfo.IsMaintaining
	etcdInfo.Addr = info.PrivateAddr

	if logicSid == physicsSid {
		// 只有物理服才需要创建conn
		logicInfo.batchPushCheckChan = make(chan struct{}, 1)
		if err := logicInfo.createConn(); err != nil {
			tilogs.L().Errorf("gamex_mgr.OnAddService serId %+v, createConn err %s", info.SerId, err.Error())
			return
		}
		if len(logicInfo.connsForClient) <= 0 {
			tilogs.L().Errorf("gamex_mgr.OnAddService serId %+v, createConn len(logicInfo.connsForClient) is 0",
				info.SerId)
			return
		}
		logicInfo.revBatchMsg()
	} else {
		tilogs.L().Infof("gamex_mgr.OnAddService logicSid %d physicsSid %d  not equal. do nothing",
			logicSid, physicsSid)
	}

	// 注册信息时使用逻辑服的sid, 方便上层使用
	mgr.gate_server.onAddShard(logicSid)

	tilogs.L().Infof("gamex_mgr OnAddService gamex %d info %+v, conn sid %v",
		logicSid, logicInfo.etcd_info, logicInfo.getPhysicsSid())

	mgr.lock.Lock()
	mgr.games[logicSid] = logicInfo
	mgr.lock.Unlock()
}

func (mgr *gamex_mgr) OnDelService(SerId discovery.ServiceId) {
	ok, _, sid := mgr.commonCheck(SerId)
	if !ok {
		return
	}
	mgr.lock.Lock()
	old, ok := mgr.games[sid]
	if ok {
		old.closeConn()
		delete(mgr.games, sid)
		mgr.gate_server.onDelShard(sid)
	}
	mgr.lock.Unlock()
	if ok {
		tilogs.L().Infof("gamex_mgr delGamex gamex %d", sid)
	}
}

func (mgr *gamex_mgr) OnChangeExtraInfo(info discovery.ServiceInfo) {
	tilogs.L().Infof("OnChangeExtraInfo ser id %v, extra info %v", info.SerId, info.ExtraInfo)

	ok, _, sid := mgr.commonCheck(info.SerId)
	if !ok {
		return
	}
	mgr.lock.RLock()
	gi, ok := mgr.games[sid]
	mgr.lock.RUnlock()

	if ok {
		extra := parseGamexExtra(info)
		isMaintaining := extra.IsMaintaining
		// 未维护->维护，踢掉gate当前所有链接
		if !gi.etcd_info.IsMaintaining && isMaintaining {
			tilogs.L().Infof("OnChangeExtraInfo kicking connections of shard %d", sid)
			mgr.gate_server.onKickShard(sid)
		}
		gi.etcd_info.IsMaintaining = isMaintaining
		if gi.getPhysicsSid() != extra.PhysicsSid {
			// 因为此时连接已经建立了, 所以无需转移、停止. 但是需要上报一下这种情况。
			tilogs.L().Errorf(
				"OnChangeExtraInfo physicsSid. old %d, new %d, sid %d",
				gi.getPhysicsSid(), extra.PhysicsSid, sid)
		}
	}
}

func parseGamexExtraMaintaining(info discovery.ServiceInfo) bool {
	return parseGamexExtra(info).IsMaintaining
}

func parseGamexExtra(info discovery.ServiceInfo) (_ discovery.GamexExtra) {
	if info.ExtraInfo == "" {
		return
	}
	e := discovery.GamexExtra{}
	err := json.Unmarshal([]byte(info.ExtraInfo), &e)
	if err != nil {
		tilogs.L().Errorf("parseGamexExtraMaintaining %s failed, err %s", info.ExtraInfo, err.Error())
		return
	}
	return e
}

/*
此用来发现gamex
*/
func (mgr *gamex_mgr) discovery(internalIpPort string) error {
	cfg := config.Cfg.GateConfig
	mgr.discoveryMgr = discovery.NewDiscoveryMgr(discovery.DiscoveryConfig{
		EtcdRoot: cfg.EtcdServer,
		BeDiscoverySerTyp: discovery.ServiceType{
			Gid:    fmt.Sprintf("%d", cfg.Gid),
			SerTyp: etcd.Ser_Gamex,
		},
		HasPrivateAddr: true,
	}, mgr, nil)
	if mgr.discoveryMgr == nil {
		return fmt.Errorf("gamex_mgr discovery NewDiscoveryMgr fail")
	}
	return mgr.discoveryMgr.Start()
}

func (mgr *gamex_mgr) Stop() {
	for _, info := range mgr.games {
		info.closeConn()
	}
	mgr.discoveryMgr.Stop()
}

func (mgr *gamex_mgr) commonCheck(SerId discovery.ServiceId) (ok bool, gid, sid uint) {
	if SerId.SerTyp.SerTyp != etcd.Ser_Gamex ||
		SerId.SerTyp.Gid != fmt.Sprintf("%d", config.Cfg.GateConfig.Gid) {
		tilogs.L().Errorf("gamex_mgr commonCheck typ or gid err %v", SerId)
		return
	}
	_sid, err := strconv.Atoi(SerId.SerId)
	if err != nil {
		tilogs.L().Errorf("gamex_mgr commonCheck strconv.Atoi sid err %v", SerId)
		return
	}
	_gid, err := strconv.Atoi(SerId.SerTyp.Gid)
	if err != nil {
		tilogs.L().Errorf("gamex_mgr commonCheck strconv.Atoi gid err %v", SerId)
		return
	}
	return true, uint(_gid), uint(_sid)
}

type gamex_info struct {
	etcd_info consts.GamexEtcdInternalInfo

	*gamexConnInfo

	// 注意: 如果 game 发生了消息堆积, 理论上可能接收到上次 check 的推送值.
	// 不过这个问题不大, 因为只要能收到推送, 说明 game 还是正常的.
	batchPushCheckChan chan struct{} // 用于异步检查batch push的连通性
}

type gamexConnInfo struct {
	// toml文件里配置了每个gamex和gatex之间存在多少条gprc链接
	// 每次获取conn时，会挑选ccu最低的grpc链接
	connsForClient []*gpr_conn
	connForBatch   *grpc.ClientConn

	lock      sync.RWMutex
	ctx       util.CancelCtx
	closeOnce sync.Once
}

// getConn 获取当前链接所属的物理服的sid
func (g *gamex_info) getPhysicsSid() uint {
	if g == nil {
		return 0
	}
	return g.etcd_info.PhysicsSid
}

// isPhysics 是不是物理服
func (g *gamex_info) isPhysics() bool {
	return g != nil && g.etcd_info.Sid == g.getPhysicsSid()
}

// getLogicSid 获取逻辑服的sid
func (g *gamex_info) getLogicSid() uint {
	if g == nil {
		return 0
	}
	return g.etcd_info.Sid
}

func newGamexConnInfo(sid uint) *gamexConnInfo {
	return &gamexConnInfo{
		ctx: util.NewCancelCtx(),
	}
}

func newGamexInfoWithConn(conn *gamexConnInfo) *gamex_info {
	return &gamex_info{gamexConnInfo: conn}
}

func (g *gamex_info) revBatchMsg() {
	go func() {
		defer func() {
			tilogs.L().Infof("gamex_info revBatchMsg stopped, %d", g.getLogicSid())
		}()
		// 一直重试
		bo := backoff.NewExponentialBackOff()
		// scene支持跨服场景后，需要给xscene留出充裕的时间，处理旧链接断开的问题
		bo.InitialInterval = 1 * time.Second
		bo.MaxElapsedTime = 0

		err := backoff.Retry(func() error {
			select {
			case <-g.ctx.Done():
				// 当关闭时, 不需要进行重试了
				tilogs.L().Infof("gamex_info revBatchMsg ctx done %d", g.getLogicSid())
				return nil
			default:
			}
			// 创建stream
			md := metadata.Pairs("serverid", config.Cfg.GateConfig.ServerId)
			ctx := metadata.NewOutgoingContext(g.ctx, md)
			g.lock.RLock()
			if g.connForBatch == nil {
				tilogs.L().Alarm("g.connForBatch is nil, create stream failed, sid %d", g.getLogicSid())
				g.lock.RUnlock()
				return nil
			}
			tilogs.L().Infof("gamex_info revBatchMsg start %d", g.getLogicSid())
			stream, err := pb.NewGateClient(g.connForBatch).OnBatchPacket(ctx)
			g.lock.RUnlock()
			if err != nil {
				tilogs.L().Errorf("connForBatch gate_client.OnBatchPacket err %s", err.Error())
				return err
			}
			for {
				p, err := stream.Recv()
				// 通过gprc stream错误判断是否需要重连
				// 目前的设计：server正常断开连接，client自身关闭不重连，其他情况进行重连
				if err != nil {
					tilogs.L().Alarm("gamex_info revBatchMsg Recv err %s, srvID %d", err.Error(), g.etcd_info.Sid)
					if errors.Is(err, io.EOF) {
						// CloseSend会让Recv获取到一个io.EOF错误
						tilogs.L().Infof("gamex_info revBatchMsg Recv EOF, srvID %d", g.getLogicSid())
						return err
					}

					// 不需要重试的情况
					code := status.Code(err)
					switch code {
					// 当conn连接不上服务器时, 相关的错误码就是 codes.Unavailable
					// 因为启用了grpc自己的重连机制, 所以这种情况应该属于临时的错误
					case codes.Canceled /*codes.Unavailable,*/, codes.Unimplemented:
						tilogs.L().Infof("gamex_info revBatchMsg Recv stop, srvID %d, code %d err %s", g.etcd_info.Sid, code, err)
						return err
					}

					g.lock.RLock()
					if g.connForBatch.GetState() != connectivity.Shutdown {
						g.lock.RUnlock()
						// retry
						tilogs.L().Infof("gamex_info revBatchMsg Recv retry, srvID %d, state %s, err %s", g.etcd_info.Sid,
							g.connForBatch.GetState().String(), err.Error())
						return err
					}
					g.lock.RUnlock()
					tilogs.L().Infof("gamex_info revBatchMsg Recv err %s, srvID %d", err.Error(), g.getLogicSid())
					return status.Error(codes.Unknown, "")
				}
				if len(p.GetAcids()) == 0 {
					pkg := p.GetPacket()
					if pkg != nil && pkg.GetPacketId() == client.PacketIDBatchCheck {
						// 这是一个异步的检查包, 只关注连通性, 不需要处理逻辑
						select {
						case g.batchPushCheckChan <- struct{}{}:
							tilogs.L().Debugf("gamex_info revBatchMsg batchPushCheck, srvID %d", g.getLogicSid())
						default:
							tilogs.L().Errorf("gamex_info revBatchMsg batchPushCheckChan full, srvID %d", g.getLogicSid())
						}
						continue
					}
				}
				gGamexMgr.SendBatchPacket(p)
			}
		}, bo)
		if err != nil {
			tilogs.L().Errorf("backoff retry failed, err %s", err.Error())
		}
	}()
}

func (g *gamex_info) getConn() *gpr_conn {
	g.lock.RLock()
	defer g.lock.RUnlock()
	if g.connsForClient == nil {
		return nil
	}
	var minCcu int32
	minCcu = math.MaxInt32
	var minConn *gpr_conn
	var index int
	for i, conn := range g.connsForClient {
		c := conn.getCcu()
		if minCcu > c {
			minCcu = c
			minConn = conn
			index = i
		}
	}
	if minConn != nil {
		tilogs.L().Debugf("gamex_info %v getConn %d ccu %v", g.getLogicSid(), index, minCcu)
	}
	return minConn
}

func (g *gamex_info) createConn() error {
	g.lock.Lock()
	defer g.lock.Unlock()
	conn, err := _conn(g.etcd_info.Addr)
	if err != nil {
		return err
	}
	g.connForBatch = conn.conn
	g.connsForClient = make([]*gpr_conn, 0, config.Cfg.GateConfig.GameConnNum)
	for i := 0; i < int(config.Cfg.GateConfig.GameConnNum); i++ {
		conn, err := _conn(g.etcd_info.Addr)
		if err != nil {
			continue
		}
		g.connsForClient = append(g.connsForClient, conn)
	}
	return nil
}

// closeConn 关闭连接. closeSid是调用关联的sid, 如果不是当前物理服的请求, 不需要关闭
func (g *gamex_info) closeConn() {
	if !g.isPhysics() {
		// 不是当前物理服的请求, 不需要关闭
		tilogs.L().Infof("gamex_info closeConn logicSid %d but physicsSid %d. do nothing",
			g.getLogicSid(), g.getPhysicsSid())
		return
	}
	g.lock.Lock()
	defer g.lock.Unlock()

	g.closeOnce.Do(func() {
		tilogs.L().Infof("gamex_info closeConn %d", g.getLogicSid())
		g.ctx.Stop()
		if g.connForBatch != nil {
			_ = g.connForBatch.Close()
			// g.connForBatch = nil
		}
		if g.connsForClient != nil && len(g.connsForClient) > 0 {
			for _, c := range g.connsForClient {
				if err := c.conn.Close(); err != nil {
					tilogs.L().Errorf("gamex_info %v close err %s", g.etcd_info, err.Error())
				}
			}
			g.connsForClient = nil
		}
	})
}

func _conn(addr string) (*gpr_conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr,
		// grpc.WithBlock(), 默认地址是有效的. 这里不使用block处理, 基于grpc后台的自动重连机制自行判断
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(util.UnaryClientRecovery),
		grpc.WithChainStreamInterceptor(util.StreamClientRecovery),
		util.GRPCWithConnectParams(),
	)
	if err != nil {
		tilogs.L().Errorf("gamex_info createConn err %s", err.Error())
		return nil, err
	}
	return &gpr_conn{conn: conn}, nil
}

type gpr_conn struct {
	conn *grpc.ClientConn
	ccu  int32
}

func (c *gpr_conn) getClientConn() *grpc.ClientConn {
	return c.conn
}

func (c *gpr_conn) getCcu() int32 {
	return atomic.LoadInt32(&c.ccu)
}

func (c *gpr_conn) addCcu() {
	atomic.AddInt32(&c.ccu, 1)
}

func (c *gpr_conn) redCcu() {
	atomic.AddInt32(&c.ccu, -1)
}
