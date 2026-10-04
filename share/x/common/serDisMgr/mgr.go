package serDisMgr

import (
	"strconv"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var etcdMgr *dis.ServiceMgr

type SerDisOpt struct {
	EtcdRoot     string
	Gid          uint
	SrvType      etcd.ServiceTypeId
	SerID        string
	InternalAddr string
	PublicAddr   string
}

func EtcdReg(opt *SerDisOpt) error {
	etcdMgr = dis.NewSeriveMgr(
		dis.ServiceConfig{
			EtcdRoot: opt.EtcdRoot,
			SerId: dis.ServiceId{
				SerTyp: dis.ServiceType{
					Gid:    strconv.Itoa(int(opt.Gid)),
					SerTyp: opt.SrvType,
				},
				SerId: opt.SerID,
			},
			HasPrivateAddr: opt.InternalAddr != "",
			HasPublicAddr:  opt.PublicAddr != "",
		}, &EtcdServiceInfo{
			internalAddr: opt.InternalAddr,
			publicAddr:   opt.PublicAddr,
		})
	return etcdMgr.Start()
}

func EtcdStop() {
	tilogs.L().Infof(".......start EtcdStop.......")
	if etcdMgr != nil {
		etcdMgr.Stop()
		tilogs.L().Infof("EtcdStop stopped")
	}
}

type EtcdServiceInfo struct {
	internalAddr string
	publicAddr   string
}

func (info *EtcdServiceInfo) GetPublicAddr() string {
	return info.publicAddr
}

func (info *EtcdServiceInfo) GetPrivateAddr() string {
	return info.internalAddr
}

func (info *EtcdServiceInfo) GetExtraInfo() string {
	return ""
}

func (info *EtcdServiceInfo) GetTickExtraInfo() string {
	return ""
}
