package servers

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/etcd"
	discovery "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
)

/*
	和战斗校验服正常的通信是nats，但战斗服的关闭，还是用etcd的服务发现来实现
*/
type battleCheckMgr struct {
	etcdServer         string
	gid                uint
	dis                *discovery.DiscoveryMgr
	onBattleCheckStop  OnServStopFunc
	onBattleCheckStart OnServStopFunc
}

func NewBattleCheckMgr(etcdServer string, gid uint, onBattleStart, onBattleStop OnServStopFunc) *battleCheckMgr {
	return &battleCheckMgr{
		etcdServer:         etcdServer,
		gid:                gid,
		onBattleCheckStop:  onBattleStop,
		onBattleCheckStart: onBattleStart,
	}
}

func (mgr *battleCheckMgr) Start() error {
	mgr.dis = discovery.NewDiscoveryMgr(discovery.DiscoveryConfig{
		EtcdRoot: mgr.etcdServer,
		BeDiscoverySerTyp: discovery.ServiceType{
			Gid:    fmt.Sprintf("%d", mgr.gid),
			SerTyp: etcd.Ser_BattleCheck,
		},
		HasPrivateAddr: true,
	}, mgr, nil)
	if mgr.dis == nil {
		return fmt.Errorf("battleCheckMgr discovery NewDiscoveryMgr fail")
	}
	return mgr.dis.Start()
}

func (mgr *battleCheckMgr) Stop() {
	mgr.dis.Stop()
}

func (mgr *battleCheckMgr) OnAddService(info discovery.ServiceInfo) {
	mgr.onBattleCheckStart(info.SerId.SerId, info.ExtraInfo)
}
func (mgr *battleCheckMgr) OnDelService(serId discovery.ServiceId) {
	mgr.onBattleCheckStop(serId.SerId, etcd.Server_BattleCheck)
}

func (mgr *battleCheckMgr) OnChangeExtraInfo(info discovery.ServiceInfo) {

}
