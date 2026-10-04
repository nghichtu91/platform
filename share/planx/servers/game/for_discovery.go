package game

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/etcd"
	discovery "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

const (
	keepAliveTTL = 6 // 服务发现的keepAlive ttl
)

type for_discovery struct {
	addr    string
	diss    []*discovery.ServiceMgr
	stop_ch chan struct{}
	e       discovery.GamexExtra
}

func NewForDiscovery(lis net.Listener, internalIp string) *for_discovery {
	addr := lis.Addr()
	port := fmt.Sprintf("%d", addr.(*net.TCPAddr).Port)
	return &for_discovery{
		addr:    net.JoinHostPort(internalIp, port),
		diss:    make([]*discovery.ServiceMgr, 0, 2),
		stop_ch: make(chan struct{}, 1),
	}
}

func (d *for_discovery) Start(etcdServer string, gid, sid uint, noCheckHealth bool, waitgroup *util.WaitGroupWrapper) bool {
	serId := discovery.ServiceId{
		SerTyp: discovery.ServiceType{
			Gid:    fmt.Sprintf("%d", gid),
			SerTyp: etcd.Ser_Gamex,
		},
		SerId: fmt.Sprintf("%d", sid),
	}
	_dis := discovery.NewSeriveMgr(discovery.ServiceConfig{
		EtcdRoot:       etcdServer,
		SerId:          serId,
		HasPrivateAddr: true,
		HasExtraInfo:   true,
		CheckHealth:    !noCheckHealth,
	}, d)

	if _dis == nil {
		return false
	}
	d.diss = append(d.diss, _dis)

	for _, _d := range d.diss {
		if err := _d.Start(); err != nil {
			return false
		}
	}
	// d.sync_ip(etcdServer, gid, sid, d.addr, waitgroup)
	return true
}

// StartMulti 用于gamex合服状态下同时注册多个sid
func (d *for_discovery) StartMulti(etcdServer string, gid uint, physicsSid uint, sids []uint,
	noCheckHealth bool, waitgroup *util.WaitGroupWrapper) bool {
	d.e.PhysicsSid = physicsSid
	for _, sid := range sids {
		serId := discovery.ServiceId{
			SerTyp: discovery.ServiceType{
				Gid:    strconv.Itoa(int(gid)),
				SerTyp: etcd.Ser_Gamex,
			},
			SerId: strconv.Itoa(int(sid)),
		}

		_dis := discovery.NewSeriveMgr(discovery.ServiceConfig{
			EtcdRoot:       etcdServer,
			SerId:          serId,
			HasPrivateAddr: true,
			HasExtraInfo:   true,
			CheckHealth:    !noCheckHealth,
		}, d)

		if _dis == nil {
			return false
		}

		d.diss = append(d.diss, _dis)

		if err := _dis.Start(); err != nil {
			return false
		}

		// d.sync_ip(etcdServer, gid, sid, d.addr, waitgroup)

		go d.watchSerStatus(etcdServer, gid, sid, waitgroup)
	}

	return true
}

func (d *for_discovery) Stop() {
	for _, _d := range d.diss {
		_d.Stop()
	}
	close(d.stop_ch)
}

func (d *for_discovery) GetPublicAddr() string {
	return ""
}
func (d *for_discovery) GetPrivateAddr() string {
	return d.addr
}
func (d *for_discovery) GetExtraInfo() string {
	bs, _ := json.Marshal(d.e)
	return string(bs)
}
func (d *for_discovery) GetTickExtraInfo() string {
	return ""
}

// Deprecated: JWS2-42263 gate检查gamex状态改为使用service_be_discovery_alive
// 弃用之前单独的键值
// func (d *for_discovery) sync_ip(etcdServer string, gid, sid uint, addr string, waitgroup *util.WaitGroupWrapper) bool {
// 	keyInternalIp := consts.GetGameInternalIp4Auth(etcdServer, gid, sid)
//
// 	if err := etcd.LeaseKeepAlive(keyInternalIp, addr, keepAliveTTL, d.stop_ch); err != nil {
// 		tilogs.L().Errorf("for_discovery sync_ip etcd3.KeepAlive err %s", err.Error())
// 		return false
// 	}
// 	return true
// }

// 监听服务器状态，当状态变为维护时，触动UpdateExtra逻辑
func (d *for_discovery) watchSerStatus(etcdServer string, gid, sid uint, wg *util.WaitGroupWrapper) {
	// server_v3/12/shards/110011/showState
	key := fmt.Sprintf("%s/%d/%s/%d/%s", etcdServer, gid, etcd.GamexDirs, sid, etcd.KeyShardState)

	v, rev, err := etcd.GetWithRev(key)
	if err != nil {
		tilogs.L().Errorf("for_discovery watchSerStatus failed, err %s", err.Error())
	}
	if v != "" {
		d.handleState(v)
	}

	etcd.WatchWithRevNoPrevRetry(key, rev, false, d.stop_ch, wg, func(resp clientv3.WatchResponse) {
		for _, event := range resp.Events {
			if event.Kv == nil {
				continue
			}
			tilogs.L().Infof("server dis watchSerStatus %s changed to %s", string(event.Kv.Key), string(event.Kv.Value))
			switch event.Type {
			case clientv3.EventTypePut:
				d.handleState(string(event.Kv.Value))
			}
		}
	})
}

// 根据当前状态和etcd状态执行维护相关逻辑
func (d *for_discovery) handleState(v string) {
	state, err := strconv.Atoi(v)
	if err != nil {
		tilogs.L().Errorf("for_discovery handleState Atoi failed, value %s, err %s", v, err.Error())
		return
	}

	tilogs.L().Infof("handleState cur maintaining %v, etcd state %d", d.e.IsMaintaining, state)

	switch state {
	case consts.ShowStateNotStart: // 半启动状态，啥也不做
	case consts.ShowStateMaintenanceNew, consts.ShowStateMaintenanceHot: // 进入维护状态
		if !d.e.IsMaintaining {
			d.e.IsMaintaining = true
			for _, dis := range d.diss {
				if err := dis.UpdateExtra(); err != nil {
					tilogs.L().Errorf("handleState failed, err %s", err.Error())
				}
			}
		}
	default: // 取消维护状态
		if d.e.IsMaintaining {
			d.e.IsMaintaining = false
			for _, dis := range d.diss {
				if err := dis.UpdateExtra(); err != nil {
					tilogs.L().Errorf("handleState failed, err %s", err.Error())
				}
			}
		}
	}
}
