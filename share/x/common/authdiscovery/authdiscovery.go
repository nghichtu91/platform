package authdiscovery

import (
	"encoding/json"
	"strconv"
	"sync"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

func AuthDiscovery(etcd_root string, gid uint) {
	authDiscovery = dis.NewDiscoveryMgr(dis.DiscoveryConfig{
		EtcdRoot: etcd_root,
		BeDiscoverySerTyp: dis.ServiceType{
			Gid:    strconv.Itoa(int(gid)),
			SerTyp: etcd.Ser_Auth,
		},
		HasPrivateAddr: true,
	}, &authCallBack{}, nil)
}

func AuthDiscoveryStart() error {
	if err := authDiscovery.Start(); err != nil {
		return err
	}
	return nil
}

func AuthDiscoveryStop() {
	authDiscovery.Stop()
}

var (
	authDiscovery *dis.DiscoveryMgr
	authsExtra    map[string]*consts.AuthExtraInfo
	lock          sync.RWMutex
)

func init() {
	authsExtra = make(map[string]*consts.AuthExtraInfo, 2)
}

func GetAuthInfo() *consts.AuthExtraInfo {
	lock.RLock()
	defer lock.RUnlock()
	for _, info := range authsExtra {
		return info
	}
	return nil
}

type authCallBack struct{}

func (a *authCallBack) OnAddService(info dis.ServiceInfo) {
	tilogs.L().Infof("authCallBack OnAddService %+v", info)
	addOrChange(info)
}

func (a *authCallBack) OnDelService(info dis.ServiceId) {
	tilogs.L().Infof("authCallBack OnDelService %+v", info)
	lock.Lock()
	delete(authsExtra, info.SerId)
	lock.Unlock()
}

func (a *authCallBack) OnChangeExtraInfo(info dis.ServiceInfo) {
	tilogs.L().Infof("authCallBack OnChangeExtraInfo %+v", info)
	addOrChange(info)
}

func addOrChange(info dis.ServiceInfo) {
	extra := &consts.AuthExtraInfo{}
	if err := json.Unmarshal([]byte(info.ExtraInfo), extra); err != nil {
		tilogs.L().Errorf("authCallBack addOrChange json.Unmarshal err %s", err.Error())
		return
	}
	tilogs.L().Infof("authCallBack addOrChange %+v", extra)
	lock.Lock()
	authsExtra[info.SerId.SerId] = extra
	lock.Unlock()
}
