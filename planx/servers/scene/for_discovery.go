package scene

import (
	"strconv"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
)

/*
	先只支持一个scene服务于一个gamexshard
*/
var SerReg *ServiceReg

type ServiceReg struct {
	ser_dis   *dis.ServiceMgr
	privateIP string
}

func StartServiceReg(etcdServer string, gid, shardId uint, srvTyp etcd.ServiceTypeId, privateIP string, noCheckHealth bool) error {
	SerReg = &ServiceReg{
		privateIP: privateIP,
	}
	SerReg.ser_dis = dis.NewSeriveMgr(dis.ServiceConfig{
		EtcdRoot: etcdServer,
		SerId: dis.ServiceId{
			SerTyp: dis.ServiceType{
				Gid:    strconv.Itoa(int(gid)),
				SerTyp: srvTyp,
			},
			SerId: strconv.Itoa(int(shardId)),
		},
		HasPrivateAddr: true,
		CheckHealth:    !noCheckHealth,
	}, SerReg)
	return SerReg.ser_dis.Start()
}

func StopServiceReg() {
	if SerReg != nil {
		SerReg.ser_dis.Stop()
	}
}

func (reg *ServiceReg) GetPublicAddr() string {
	return ""
}

func (reg *ServiceReg) GetPrivateAddr() string {
	return reg.privateIP
}
func (reg *ServiceReg) GetExtraInfo() string {
	return ""
}
func (reg *ServiceReg) GetTickExtraInfo() string {
	return ""
}
