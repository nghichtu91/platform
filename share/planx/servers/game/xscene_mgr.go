package game

import (
	"strconv"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

// xSceneMgr cross scene服务发现监听
type xSceneMgr struct {
	etcdRoot string
	gid      uint
	sid      uint

	// 只有创建的时候写，后续都只读，不需要加锁
	sceneMaps map[int32]int32
	warzoneID uint64

	mgr *dis.DiscoveryMgr

	quit chan struct{}
}

func (mgr *xSceneMgr) OnAddService(info dis.ServiceInfo) {
	// err := AddXSceneSrv(info.SerId.SerId, info.PrivateAddr, mgr.sceneMaps)
	err := AddXSceneSrv(&etcd.XSceneInfo{
		SerID:    info.SerId.SerId,
		Addr:     info.PrivateAddr,
		ZoneID:   mgr.warzoneID,
		SceneIDs: mgr.sceneMaps,
	})
	if err != nil {
		tilogs.L().Errorf("xSceneMgr OnAddService AddXSceneSrv failed, err %s", err.Error())
	}
}

func (mgr *xSceneMgr) OnDelService(serId dis.ServiceId) {
	err := DelXSceneSrv()
	if err != nil {
		tilogs.L().Errorf("xSceneMgr OnDelService DelXSceneSrv failed, err %s", err.Error())
	}
}

func (mgr *xSceneMgr) OnChangeExtraInfo(info dis.ServiceInfo) {
	mgr.OnDelService(info.SerId)
	mgr.OnAddService(info)
}

func NewXSceneMgr(etcdRoot string, gid uint, info *etcd.XSceneInfo) *xSceneMgr {
	mgr := &xSceneMgr{
		sceneMaps: info.SceneIDs,
		warzoneID: info.ZoneID,
		quit:      make(chan struct{}, 1),
	}

	mgr.mgr = dis.NewDiscoveryMgr(dis.DiscoveryConfig{
		EtcdRoot: etcdRoot,
		BeDiscoverySerTyp: dis.ServiceType{
			Gid:    strconv.Itoa(int(gid)),
			SerTyp: etcd.SerXScene,
		},
		BeDiscoverySerId: info.SerID, // 只监听自己的
		HasPrivateAddr:   true,
	}, mgr, nil)

	return mgr
}

func (mgr *xSceneMgr) loop() {
	err := mgr.mgr.Start()
	if err != nil {
		tilogs.L().Errorf("start xSceneMgr loop failed, err %s", err.Error())
		return
	}

	for {
		select {
		case <-mgr.quit:
			mgr.mgr.Stop()
			return
		}
	}
}

func (mgr *xSceneMgr) stop() {
	tilogs.L().Infof("xSceneMgr stop")
	close(mgr.quit)
}
