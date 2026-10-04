package gate

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
)

const (
	minConcurrencyCheck = 1                      // 每次检查的最小并发数
	maxConcurrencyCheck = 50                     // 每次检查的最大并发数
	minGameCheckTimeout = 200 * time.Millisecond // 每个 game 的检查的最短超时时间
	minCheckInterval    = time.Minute            // 两次检查的最短间隔
)

type GameCheckParam struct {
	GameCheckTimeout time.Duration // 每个 game 的检查的最长超时时间
	CheckConcurrency int           // 每次检查的最大并发数
	Force            bool          // 是否强制检查
}

type GameCheckRet int

const (
	GameCheckSuccess GameCheckRet = iota
	GameCheckFail
)

// Validate 检查参数是否合法
func (param *GameCheckParam) Validate() {
	if param.GameCheckTimeout < minGameCheckTimeout {
		param.GameCheckTimeout = minGameCheckTimeout
	}
	if param.CheckConcurrency < minConcurrencyCheck {
		param.CheckConcurrency = minConcurrencyCheck
	}
	if param.CheckConcurrency > maxConcurrencyCheck {
		param.CheckConcurrency = maxConcurrencyCheck
	}
}

// canHealthCheck 是否可以进行健康检查
func (mgr *gamex_mgr) canHealthCheck() bool {
	return mgr.running.Load() == false
}

// healthCheck 总的检查入口
func (mgr *gamex_mgr) healthCheck(param GameCheckParam) {

	if !mgr.running.CAS(false, true) {
		tilogs.L().Infof("gamex_mgr healthCheck is running, skip. req %+v", param)
		return
	}
	defer mgr.running.Store(false)

	// 参数检查
	param.Validate()

	tilogs.L().Infof("gamex_mgr healthCheck start, param %+v", param)

	start := time.Now()
	// 如果不是强制检查, 并且两次检查的间隔时间太短, 则跳过

	if !param.Force && start.Sub(mgr.lastCheckEnd) < minCheckInterval {
		tilogs.L().Infof("gamex_mgr healthCheck skip for too short interval. lastCheckEnd %v", mgr.lastCheckEnd)
		return
	}

	var allValidGameCount int

	defer func() {
		mgr.lastCheckEnd = time.Now()
		tilogs.L().Infof("check all physics gamex info end, count %d, cost %v", allValidGameCount, time.Since(start))
	}()

	// check all physics gamex info
	var allGame []*gamex_info
	mgr.lock.RLock()
	allGame = make([]*gamex_info, 0, len(mgr.games)/4)
	for _, game := range mgr.games {
		if game == nil {
			continue
		}
		if !game.isPhysics() {
			continue
		}
		allGame = append(allGame, game)
	}
	mgr.lock.RUnlock()

	allValidGameCount = len(allGame)

	if allValidGameCount == 0 {
		// why no physics game?
		return
	}

	// 是否有必要持久化持有这个 pool?

	var pool, err = ants.NewPool(param.CheckConcurrency, // 因为每个 game 进行检查时, 会启用若干个 goroutine, 这个限制是最多并行检查多少个 game
		ants.WithLogger(util.AntsLogger{}),      // 使用自定义的 logger
		ants.WithPreAlloc(false),                // 不进行预分配, 因为可能没有那么多的服务器需要检查
		ants.WithNonblocking(false),             // 如果没有空闲的 worker, 则阻塞
		ants.WithDisablePurge(false),            // 启用 worker 的清理
		ants.WithExpiryDuration(time.Second*10), // worker 的清理时间
		ants.WithMaxBlockingTasks(1),            // 只有当前协程操作 pool, 所以可以设置成 1
		ants.WithPanicHandler(func(i interface{}) {
			tilogs.L().Errorf("gamex_mgr healthCheck panic %v", i)
		}),
	)
	if err != nil {
		tilogs.L().Alarm("gamex_mgr healthCheck ants.NewPool err %v", err)
		return
	}

	var checkReq = mgr.gateCheckReq
	for _, game := range allGame {
		_game := game
		err := pool.Submit(func() {
			_game.healthCheck(checkReq, param.GameCheckTimeout)
		})
		if err != nil {
			tilogs.L().Alarm("gamex_mgr healthCheck %v pool.Submit err %v", _game.getLogicSid(), err)
		}
	}

	// 等待所有的检查完成. 因为 pool.Submit 是阻塞的, 所以这里的等待时间相当于单个 game 的检查的最大时间
	err = pool.ReleaseTimeout(param.GameCheckTimeout * 4)
	if err != nil {
		tilogs.L().Alarm("gamex_mgr healthCheck pool.ReleaseTimeout err %v", err)
	}
}

// healthCheck
func (g *gamex_info) healthCheck(checkReq gateCheckReq, timeout time.Duration) {
	if !g.isPhysics() {
		// skip not physics
		tilogs.L().Infof("gamex_info healthCheck %d skip not physics %v", g.getLogicSid(), g.getPhysicsSid())
		return
	}
	tilogs.L().Infof("gamex_info connect healthCheck %d", g.getLogicSid())
	var logicConn []*gpr_conn
	var batchConn *grpc.ClientConn
	g.lock.RLock()
	logicConn = make([]*gpr_conn, len(g.connsForClient))
	copy(logicConn, g.connsForClient)
	batchConn = g.connForBatch
	g.lock.RUnlock()

	// 一个 game 的总体的检查超时
	var checkCtx = util.NewCancelCtxWithTimeout(g.ctx, timeout)
	defer checkCtx.Stop()

	gameId := strconv.Itoa(int(g.getLogicSid()))

	const (
		TypeRPC      = "RPC"
		TypePush     = "Push"
		TypePushRPC  = TypePush + "RPC"
		TypePushWait = TypePush + "Wait"
	)

	var wg sync.WaitGroup

	// 逻辑 conn + batch conn
	wg.Add(len(logicConn) + 1)

	warp := func(idx int, typ string, f func()) {
		go func() {
			defer wg.Done()
			defer tilogs.PanicCatcher("gamex_info %v healthCheck %v %v panic", g.getLogicSid(), idx, typ)
			f()
		}()
	}

	for idx, conn := range logicConn {
		_conn := conn
		_connIdx := idx
		warp(idx, TypeRPC, func() {
			// check rpc
			err := checkGateConn(checkCtx, _conn.conn, checkReq.pingCheckReq)
			if isHealth(err) {
				// 检查成功, 直接返回
				return
			}
			// 其他错误, 记录
			tilogs.L().Alarm("gamex_info %v healthCheck logicConn %d err %v", g.getLogicSid(), _connIdx, err)
			metricsFailCount(gameId, strconv.Itoa(_connIdx)+"."+TypeRPC)
		})
	}
	warp(0, TypePush, func() {

		// 排空 batch push 的 channel, 防止旧的消息本次检测
		select {
		case <-g.batchPushCheckChan:
		default:
		}

		// check batch push
		err := checkGateConn(checkCtx, batchConn, checkReq.pushCheckReq)
		if !isHealth(err) {
			tilogs.L().Alarm("gamex_info %v healthCheck batchConn rpc err %v", g.getLogicSid(), err)
			metricsFailCount(gameId, TypePushRPC)
			return
		}
		// 异步等待 batch push 的返回
		select {
		case <-checkCtx.Done():
			if checkCtx.IsCancel() {
				tilogs.L().Infof("gamex_info %v healthCheck batchConn cancel", g.getLogicSid())
				return
			}
			if checkCtx.IsTimeout() {
				tilogs.L().Alarm("gamex_info %v healthCheck batchConn wait timeout", g.getLogicSid())
				metricsFailCount(gameId, TypePushWait)
			}
		case <-g.batchPushCheckChan:
			tilogs.L().Debugf("gamex_info %v healthCheck batchConn batchPushCheckChan success", g.getLogicSid())
		}
	})
	wg.Wait()
}

// check

var (
	ErrConnIsNil = status.Error(codes.Unavailable, "conn is nil")
)

func checkGateConn(ctx context.Context, conn *grpc.ClientConn, req *pb.CtlReq) error {
	if conn == nil {
		return ErrConnIsNil
	}
	client := pb.NewGateClient(conn)
	resp, err := client.CtlRpc(ctx, req)
	if err != nil {
		return fmt.Errorf("checkGateConn grpc err %w", err)
	}
	if resp.Res != pb.CtlRes_Ctl_Success {
		return fmt.Errorf("checkGateConn ret code %v, reason %v", resp.GetRes(), resp.GetReason())
	}
	return nil
}

// isHealth 判断是否是健康的错误
func isHealth(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return true
	}
	return false
}

// metricsFail 记录失败的 metrics
func metricsFailCount(uid, suffix string) {
	failKey := fmt.Sprintf("gamex%s.fail.%s", uid, suffix)
	allmetrics.ServerCheckFail(failKey)
}
