package crossx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/planx"

	"github.com/nghichtu91/platform/share/x/common/consts"

	"google.golang.org/grpc/metadata"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/x/common/msg/battlecheck/pb"

	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/util"
	ver "github.com/nghichtu91/platform/share/planx/version"
	"google.golang.org/grpc"
)

type GenBattleCheckSerHandler func(battleSerId string) IBattleCheckSerHandler

type IBattleCheckSerHandler interface {
	OnRevBattleCheckServMsg(msg *pb.BattleCheckMsg)
	OnServStop(battleSerId string)
}

// 根据task count，取压力最小的battle check
// 若返回""，则没有找到battle check
func GetBattleCheckServ() string {
	gBattleCheckServsMgr.lock.RLock()
	defer gBattleCheckServsMgr.lock.RUnlock()
	minTC := int64(math.MaxInt64)
	var res *BattleCheckServ
	for _, b := range gBattleCheckServsMgr.battleCheckServs {
		if b.privateAddr == "" {
			continue
		}
		if b.taskCount < minTC {
			minTC = b.taskCount
			res = b
		}
	}
	if res == nil {
		return ""
	}
	return res.servId
}

// 给指定servId的战斗检查服发协议msg
func Send2BattleCheckServ(battleCheckSerId string, msg *pb.BattleCheckMsg) bool {
	gBattleCheckServsMgr.lock.RLock()
	ser, ok := gBattleCheckServsMgr.battleCheckServs[battleCheckSerId]
	if !ok {
		gBattleCheckServsMgr.lock.RUnlock()
		return false
	}
	gBattleCheckServsMgr.lock.RUnlock()
	return ser.Send(msg)
}

var gBattleCheckServsMgr *battleCheckServerMgr

type battleCheckServerMgr struct {
	serverId         string
	battleCheckServs map[string]*BattleCheckServ // key: battleServId
	genHandler       GenBattleCheckSerHandler
	disMgr           *dis.DiscoveryMgr
	lock             sync.RWMutex
	wait             util.WaitGroupWrapper
}

func StartBattleCheckMgr(etcdServer string, gid uint, serverId string,
	genHandler GenBattleCheckSerHandler) error {
	gBattleCheckServsMgr = &battleCheckServerMgr{
		serverId:         serverId,
		battleCheckServs: make(map[string]*BattleCheckServ, 4),
		genHandler:       genHandler,
	}

	// 启动服务器发现battle check
	gBattleCheckServsMgr.disMgr = dis.NewDiscoveryMgr(dis.DiscoveryConfig{
		EtcdRoot: etcdServer,
		BeDiscoverySerTyp: dis.ServiceType{
			Version: ver.Version,
			Gid:     fmt.Sprintf("%d", gid),
			SerTyp:  etcd.Ser_BattleCheck,
		},
		HasPrivateAddr: true,
	}, gBattleCheckServsMgr, nil)
	if gBattleCheckServsMgr.disMgr == nil {
		return fmt.Errorf("StartBattleCheckMgr fail")
	}
	return gBattleCheckServsMgr.disMgr.Start()
}

func StopBattleCheckMgr() {
	gBattleCheckServsMgr.disMgr.Stop()
	gBattleCheckServsMgr.lock.Lock()
	for _, ser := range gBattleCheckServsMgr.battleCheckServs {
		ser.Stop()
	}
	gBattleCheckServsMgr.lock.Unlock()
	gBattleCheckServsMgr.wait.Wait()
}

func (mgr *battleCheckServerMgr) OnAddService(info dis.ServiceInfo) {
	mgr.lock.Lock()
	b, needInit := mgr._init_battle(info.SerId.SerId, info.PrivateAddr)
	mgr.lock.Unlock()
	if needInit {
		b.Start(mgr.serverId)
	}
}

func (mgr *battleCheckServerMgr) OnDelService(serId dis.ServiceId) {
	mgr.lock.Lock()
	if ser, ok := mgr.battleCheckServs[serId.SerId]; ok {
		ser.handler.OnServStop(serId.SerId)
		delete(mgr.battleCheckServs, serId.SerId)
		ser.Stop()
		tilogs.L().Infof("%s OnDelService", ser.prefix())
	}
	mgr.lock.Unlock()
}

func (mgr *battleCheckServerMgr) OnChangeExtraInfo(info dis.ServiceInfo) {
	extra := &consts.BattleCheckExtra{}
	err := json.Unmarshal([]byte(info.TickExtraInfo), extra)
	if err != nil {
		tilogs.L().Errorf("battleServsMgr OnChangeExtraInfo json.Unmarshal err %v", info)
		return
	}
	tilogs.L().Debugf("battleServsMgr task count change %s %s %d",
		info.SerId.SerTyp.Gid,
		info.SerId.SerId,
		extra.TaskCount)
	mgr.lock.Lock()
	b, needInit := mgr._init_battle(info.SerId.SerId, info.PrivateAddr)
	b.taskCount = extra.TaskCount
	mgr.lock.Unlock()
	if needInit {
		b.Start(mgr.serverId)
	}
}

// need in lock
func (mgr *battleCheckServerMgr) _init_battle(serverId, privateAddr string) (ser *BattleCheckServ, needInit bool) {
	b, ok := mgr.battleCheckServs[serverId]
	if !ok {
		b = &BattleCheckServ{
			servId:      serverId,
			privateAddr: privateAddr,
			handler:     mgr.genHandler(serverId),
			sendChan:    make(chan *pb.BattleCheckMsg, 16),
			wait:        &mgr.wait,
			quit:        make(chan struct{}, 1),
		}
		mgr.battleCheckServs[serverId] = b
	}
	return b, !ok
}

type BattleCheckServ struct {
	servId      string
	privateAddr string
	taskCount   int64
	handler     IBattleCheckSerHandler
	conn        *grpc.ClientConn
	sendChan    chan *pb.BattleCheckMsg
	wait        *util.WaitGroupWrapper
	quit        chan struct{}
}

func (bs *BattleCheckServ) Start(serverId string) bool {
	_conn, err := bs._conn()
	if err != nil {
		return false
	}
	md := metadata.Pairs("serverid", serverId)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	stream, err := pb.NewGameBattleCheckClient(_conn).OnPacket(ctx)
	if err != nil {
		tilogs.L().Errorf("[BattleCheckServ][start] pb.NewGameBattleCheckClient.OnPacket err %s", err.Error())
		_conn.Close()
		return false
	}
	bs.conn = _conn
	tilogs.L().Infof("BattleCheckServ start %s", bs.prefix())
	// rev
	bs.wait.Wrap(func() {
		defer func() {
			close(bs.quit)
			tilogs.L().Infof("BattleCheckServ rev close, servId %s", bs.servId)
		}()
		for {
			p, err := stream.Recv()
			if err != nil && (err == io.EOF ||
				strings.Contains(err.Error(), "code = Unavailable desc = transport is closing") ||
				strings.Contains(err.Error(), "code = Canceled desc = grpc: the client connection is closing")) {
				return
			}
			if err != nil {
				tilogs.L().Errorf("[BattleCheckServ] stream.Recv, servId %s, ip %s, err %s", bs.servId, bs.privateAddr, err.Error())
				return
			}
			bs.handler.OnRevBattleCheckServMsg(p)
		}
	})
	// send
	bs.wait.Wrap(func() {
		defer func() {
			tilogs.L().Infof("BattleCheckServ send close, servId %s", bs.servId)
		}()
		for {
			select {
			case msg := <-bs.sendChan:
				tilogs.L().Debugf("send to battle %s ", bs.servId)

				if err := stream.Send(msg); err != nil {
					tilogs.L().Errorf("[BattleCheckServ] stream.Send, servId %s, ip %s, err %s",
						bs.servId, bs.privateAddr, err.Error())
				}
			case <-bs.quit:
				return
			}
		}
	})
	return true
}

func (bs *BattleCheckServ) Stop() {
	bs.conn.Close()
}

func (bs *BattleCheckServ) _conn() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, bs.privateAddr, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithChainUnaryInterceptor(util.UnaryClientRecovery),
		grpc.WithChainStreamInterceptor(util.StreamClientRecovery))
	if err != nil {
		tilogs.L().Errorf("[BattleCheckServ][_conn] grpc.DialContext %s err %s", bs.privateAddr, err.Error())
		return nil, err
	}
	return conn, nil
}

func (bs *BattleCheckServ) Send(msg *pb.BattleCheckMsg) bool {
	if bs.conn == nil {
		tilogs.L().Errorf("[BattleCheckServ][send] but init fail, servId %s, ip %s", bs.servId, bs.privateAddr)
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), planx.DownTimeOut)
	defer cancel()
	select {
	case bs.sendChan <- msg:
	case <-ctx.Done():
		tilogs.L().Errorf("[BattleCheckServ][send] full timeout, servId %s, ip %s, msg %s", bs.servId, bs.privateAddr, msg)
		return false
	}
	return true
}

func (bs *BattleCheckServ) prefix() string {
	return fmt.Sprintf("[BattleCheckServ][%s][%s]", bs.servId, bs.privateAddr)
}
