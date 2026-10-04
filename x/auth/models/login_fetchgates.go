package models

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"

	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

type gate struct {
	gateId  string
	ccu     int64
	addr    string
	host    string
	rpcAddr string
	isOpen  bool
}

type gate_mgr struct {
	gid_gate map[string]map[string]*gate
	diss     map[string]*dis.DiscoveryMgr

	lock_gate sync.RWMutex
}

var (
	GateMgr *gate_mgr
)

func init() {
	GateMgr = &gate_mgr{
		diss:     make(map[string]*dis.DiscoveryMgr, 2),
		gid_gate: make(map[string]map[string]*gate, 2),
	}
}

func (mgr *gate_mgr) GetOneGate(gid string) (gate, bool) {
	mgr.lock_gate.RLock()
	defer mgr.lock_gate.RUnlock()

	gates, ok := mgr.gid_gate[gid]
	if !ok || len(gates) <= 0 {
		tilogs.L().Errorf("GetOneGate gid %s gates %v", gid, gates)
		return gate{}, false
	}
	var minCcu int64
	minCcu = math.MaxInt64
	var res *gate
	for _, g := range gates {
		if g.addr == "" || g.rpcAddr == "" {
			continue
		}
		if !g.isOpen {
			continue
		}
		if g.ccu < minCcu {
			minCcu = g.ccu
			res = g
		}
	}
	if res == nil {
		return gate{}, false
	}
	tilogs.L().Debugf("gate_mgr GetOneGate %v", res)
	return *res, true
}

func (mgr *gate_mgr) Init() {
	gid := fmt.Sprintf("%d", config.Cfg.CommonCfg.Gid)
	_dis := dis.NewDiscoveryMgr(dis.DiscoveryConfig{
		EtcdRoot: config.Cfg.CommonCfg.EtcdServer,
		BeDiscoverySerTyp: dis.ServiceType{
			Gid:    gid,
			SerTyp: etcd.Ser_Gate,
		},
		HasPublicAddr:  true,
		HasPrivateAddr: true,
	}, mgr, nil)
	mgr.diss[gid] = _dis
}

func (mgr *gate_mgr) Start() error {
	for _, _dis := range mgr.diss {
		if err := _dis.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (mgr *gate_mgr) Stop() {
	for _, v := range mgr.diss {
		v.Stop()
	}
}

func (mgr *gate_mgr) delGate(gid string, gateId string) {
	mgr.lock_gate.Lock()
	defer mgr.lock_gate.Unlock()
	mgr._del_gate(gid, gateId)
}

// need in lock
func (mgr *gate_mgr) _init_gate(gid string, gateId string) *gate {
	gates, ok := mgr.gid_gate[gid]
	if !ok {
		gates = make(map[string]*gate, 2)
		mgr.gid_gate[gid] = gates
	}
	g, ok := gates[gateId]
	if !ok {
		g = &gate{
			gateId: gateId,
		}
		gates[gateId] = g
	}
	return g
}

// need in lock
func (mgr *gate_mgr) _del_gate(gid string, gateId string) {
	gates, ok := mgr.gid_gate[gid]
	if ok {
		delete(gates, gateId)
	}
}

func (mgr *gate_mgr) OnAddService(info dis.ServiceInfo) {
	mgr.lock_gate.Lock()
	defer mgr.lock_gate.Unlock()
	gate := mgr._init_gate(info.SerId.SerTyp.Gid, info.SerId.SerId)
	gate.addr = info.PublicAddr
	gate.rpcAddr = info.PrivateAddr
	tilogs.L().Infof("OnAddService gate %v", info)
}

func (mgr *gate_mgr) OnDelService(serId dis.ServiceId) {
	mgr.lock_gate.Lock()
	defer mgr.lock_gate.Unlock()
	mgr._del_gate(serId.SerTyp.Gid, serId.SerId)
	tilogs.L().Infof("OnDelService gate %v", serId)
}

func (mgr *gate_mgr) OnChangeExtraInfo(info dis.ServiceInfo) {
	// 因为 tick 的存在, 这里触发的几率挺频繁的...
	ccu := &consts.GateExtra{}
	// 现有的服务发现机制一定是一个合法的json
	err := json.Unmarshal(util.Str2Bytes(info.TickExtraInfo), ccu)
	if err != nil {
		tilogs.L().Errorf("gate_mgr OnChangeExtraInfo tick json.Unmarshal err %v", info.String())
		return
	}
	host := &dis.GateExtra{}
	err = json.Unmarshal(util.Str2Bytes(info.ExtraInfo), host)
	if err != nil {
		tilogs.L().Errorf("gate_mgr OnChangeExtraInfo json.Unmarshal err %v", info.String())
		return
	}
	tilogs.L().Debugf("gate ccu change %s %s [%s] %d",
		info.SerId.SerTyp.Gid,
		info.SerId.SerId,
		host.Host,
		ccu.Ccu)
	mgr.lock_gate.Lock()
	defer mgr.lock_gate.Unlock()
	gate := mgr._init_gate(info.SerId.SerTyp.Gid, info.SerId.SerId)
	gate.ccu = ccu.Ccu
	gate.isOpen = ccu.IsOpen
	gate.host = host.Host
}
