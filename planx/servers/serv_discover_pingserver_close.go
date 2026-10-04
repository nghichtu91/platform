package servers

import (
	"fmt"
	"sync"

	"github.com/nghichtu91/platform/share/planx/etcd"
	discovery "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
)

/*
	维护一个pingServer列表，在发现服务器启动和关闭的时候变更列表信息
*/
type pingServerMgr struct {
	etcdServer       string
	gid              uint
	dis              *discovery.DiscoveryMgr
	onBattleStart    OnServStopFunc
	onBattleStop     OnServStopFunc
	agentEtcdWatch   chan struct{}
	BattleAgentDelay *sync.Map
}

func NewPingServerMgr(etcdServer string, gid uint, battleAgentDelay *sync.Map, onBattleStart, onBattleStop OnServStopFunc) *pingServerMgr {
	return &pingServerMgr{
		etcdServer:       etcdServer,
		gid:              gid,
		onBattleStart:    onBattleStart,
		onBattleStop:     onBattleStop,
		BattleAgentDelay: battleAgentDelay,
		agentEtcdWatch:   make(chan struct{}, 1),
	}
}

func (mgr *pingServerMgr) Start() error {
	mgr.dis = discovery.NewDiscoveryMgr(discovery.DiscoveryConfig{
		EtcdRoot: mgr.etcdServer,
		BeDiscoverySerTyp: discovery.ServiceType{
			Gid:    fmt.Sprintf("%d", mgr.gid),
			SerTyp: etcd.Ser_PingServer,
		},
		HasPrivateAddr: true,
	}, mgr, nil)
	if mgr.dis == nil {
		return fmt.Errorf("battleAgentMgr discovery NewDiscoveryMgr fail")
	}
	return mgr.dis.Start()
}

func (mgr *pingServerMgr) Stop() {
	mgr.dis.Stop()
	close(mgr.agentEtcdWatch)
}

func (mgr *pingServerMgr) OnAddService(info discovery.ServiceInfo) {
	mgr.onBattleStart(info.SerId.SerId, info.ExtraInfo)
}
func (mgr *pingServerMgr) OnDelService(serId discovery.ServiceId) {
	mgr.onBattleStop(serId.SerId, etcd.Server_PingServer)
}

func (mgr *pingServerMgr) OnChangeExtraInfo(info discovery.ServiceInfo) {

}
