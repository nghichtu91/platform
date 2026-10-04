package etcd_ser_discovery

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/etcd"

	"github.com/nghichtu91/platform/share/planx/version"
)

/*
DiscoveryService负责发布服务
*/

type DiscoveryService struct {
	etcdRoot   string
	gid        uint
	Version    string //如果希望可以适用是有版本，version为空即可
	addr       string
	disService *ServiceMgr
	serType    uint
	serId      string
	Tags       []string
}

func (d *DiscoveryService) SetSerId(serId string) {
	d.serId = serId
}

//如果希望可以适用所有版本，version为空即可
func (d *DiscoveryService) SetVersion(ver string) {
	d.Version = ver
}

func (d *DiscoveryService) SetSerType(serType uint) {
	d.serType = serType
}

func (d *DiscoveryService) SetAddr(addr string) {
	d.addr = addr
}

func (d *DiscoveryService) SetGid(gid uint) {
	d.gid = gid
}

func (d *DiscoveryService) SetEtcdRoot(etcdRoot string) {
	d.etcdRoot = etcdRoot
}

func (ds *DiscoveryService) GetPublicAddr() string {
	return ""
}

func (ds *DiscoveryService) GetPrivateAddr() string {
	return ds.addr
}

func (ds *DiscoveryService) GetExtraInfo() string {
	return ""
}

func (ds *DiscoveryService) GetTickExtraInfo() string {
	return ""
}

func (ds *DiscoveryService) Start() bool {
	serviceId := ServiceId{
		SerTyp: ServiceType{
			Version: version.Version,
			Gid:     fmt.Sprint(ds.gid),
			SerTyp:  etcd.ServiceTypeId(ds.serType),
		},
		SerId: ds.serId,
	}

	dis := NewSeriveMgr(ServiceConfig{
		EtcdRoot:       ds.etcdRoot,
		SerId:          serviceId,
		HasPrivateAddr: true,
		Tags:           ds.Tags,
		CheckHealth:    true,
	}, ds)
	if dis == nil {
		return false
	}
	ds.disService = dis

	if err := ds.disService.Start(); err != nil {
		return false
	}
	return true
}

func (ds *DiscoveryService) Stop() {
	ds.disService.Stop()
}
