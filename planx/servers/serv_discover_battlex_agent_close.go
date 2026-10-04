package servers

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/etcd"
	discovery "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
)

/*
和战斗服正常的通信是nats，但战斗服的关闭，还是用etcd的服务发现来实现
*/
type battleAgentMgr struct {
	etcdServer       string
	gid              uint
	dis              *discovery.DiscoveryMgr
	onBattleStart    OnServStopFunc
	onBattleStop     OnServStopFunc
	agentEtcdWatch   chan struct{}
	BattleAgentDelay *sync.Map
	loadRevision     int64
}

func NewBattleAgentMgr(etcdServer string, gid uint, loadRevision int64, battleAgentDelay *sync.Map,
	onBattleStart, onBattleStop OnServStopFunc) *battleAgentMgr {
	return &battleAgentMgr{
		etcdServer:       etcdServer,
		gid:              gid,
		onBattleStart:    onBattleStart,
		onBattleStop:     onBattleStop,
		BattleAgentDelay: battleAgentDelay,
		agentEtcdWatch:   make(chan struct{}, 1),
		loadRevision:     loadRevision,
	}
}

func (mgr *battleAgentMgr) Start(w *util.WaitGroupWrapper) error {
	mgr.dis = discovery.NewDiscoveryMgr(discovery.DiscoveryConfig{
		EtcdRoot: mgr.etcdServer,
		BeDiscoverySerTyp: discovery.ServiceType{
			Gid:    fmt.Sprintf("%d", mgr.gid),
			SerTyp: etcd.Ser_BattleAgent,
		},
		HasPrivateAddr: true,
	}, mgr, nil)
	if mgr.dis == nil {
		return fmt.Errorf("battleAgentMgr discovery NewDiscoveryMgr fail")
	}

	watchKey := fmt.Sprintf("%s/%d/battlexagent/delay", mgr.etcdServer, mgr.gid)

	etcd.WatchWithRetry(watchKey, true, mgr.agentEtcdWatch, w, func(resp clientv3.WatchResponse) {
		for _, event := range resp.Events {
			if event.Kv == nil {
				continue
			}
			if event.Type == clientv3.EventTypePut {
				key := getGidSwitchKey(event.Kv.Key)
				tilogs.L().Debugf("battlexagent delay key :%v value %v", key, string(event.Kv.Value))
				mgr.BattleAgentDelay.Store(key, string(event.Kv.Value))
			}
		}
	})

	return mgr.dis.Start()
}

func (mgr *battleAgentMgr) Stop() {
	mgr.dis.Stop()
	close(mgr.agentEtcdWatch)
}

func (mgr *battleAgentMgr) OnAddService(info discovery.ServiceInfo) {
	mgr.onBattleStart(info.SerId.SerId, info.ExtraInfo)
}
func (mgr *battleAgentMgr) OnDelService(serId discovery.ServiceId) {
	mgr.onBattleStop(serId.SerId, etcd.Server_BattleAgent)
}

func (mgr *battleAgentMgr) OnChangeExtraInfo(info discovery.ServiceInfo) {

}

func getGidSwitchKey(k []byte) (key string) {
	sections := strings.Split(string(k), "/")
	if len(sections) == 0 {
		return
	}
	key = sections[len(sections)-1]
	return
}
