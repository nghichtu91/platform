package servers

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/etcd"
	discovery "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
)

/*
	和战斗服正常的通信是nats，但战斗服的关闭，还是用etcd的服务发现来实现
*/
type battleMgr struct {
	etcdServer   string
	gid          uint
	dis          *discovery.DiscoveryMgr
	onBattleStop OnServStopFunc
}

func NewBattleMgr(etcdServer string, gid uint, onBattleStop OnServStopFunc) *battleMgr {
	return &battleMgr{
		etcdServer:   etcdServer,
		gid:          gid,
		onBattleStop: onBattleStop,
	}
}

func (mgr *battleMgr) Start() error {
	mgr.dis = discovery.NewDiscoveryMgr(discovery.DiscoveryConfig{
		EtcdRoot: mgr.etcdServer,
		BeDiscoverySerTyp: discovery.ServiceType{
			Gid:    fmt.Sprintf("%d", mgr.gid),
			SerTyp: etcd.Ser_Battle,
		},
		HasPrivateAddr: true,
	}, mgr, nil)
	if mgr.dis == nil {
		return fmt.Errorf("battleMgr discovery NewDiscoveryMgr fail")
	}
	return mgr.dis.Start()
}

func (mgr *battleMgr) Stop() {
	mgr.dis.Stop()
}

func (mgr *battleMgr) OnAddService(info discovery.ServiceInfo) {

}
func (mgr *battleMgr) OnDelService(serId discovery.ServiceId) {
	mgr.onBattleStop(serId.SerId, etcd.Server_Battle)
}

func (mgr *battleMgr) OnChangeExtraInfo(info discovery.ServiceInfo) {

}
