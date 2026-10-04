package cmds

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/nghichtu91/platform/share/x/common/allmetrics"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	gateconfig "github.com/nghichtu91/platform/share/x/gatex/config"
)

var SerReg *ServiceReg

type ServiceReg struct {
	ser_dis  *dis.ServiceMgr
	publicIP string
	rpcIP    string
	gateInfo IGateInfo
}

type IGateInfo interface {
	GetPublicIP() string
	GetPublicHost() string
}

func StartServiceReg(serverId string, info IGateInfo, rpcIP string) error {
	SerReg = &ServiceReg{
		publicIP: info.GetPublicIP(),
		rpcIP:    rpcIP,
		gateInfo: info,
	}
	SerReg.ser_dis = dis.NewSeriveMgr(dis.ServiceConfig{
		EtcdRoot: gateconfig.Cfg.GateConfig.EtcdServer,
		SerId: dis.ServiceId{
			SerTyp: dis.ServiceType{
				Gid:    fmt.Sprintf("%d", gateconfig.Cfg.GateConfig.Gid),
				SerTyp: etcd.Ser_Gate,
			},
			SerId: serverId,
		},
		HasPublicAddr:    true,
		HasPrivateAddr:   true,
		HasExtraInfo:     true,
		HasTickExtraInfo: true,
		ExtraInfoTick: time.Second *
			time.Duration(gateconfig.Cfg.CCUMetricsConfig.LoginConnectorTickTime),
		CheckHealth: true,
	}, SerReg)
	return SerReg.ser_dis.Start()
}

func StopServiceReg() {
	SerReg.ser_dis.Stop()
}

func (reg *ServiceReg) GetPublicAddr() string {
	return reg.publicIP
}
func (reg *ServiceReg) GetPrivateAddr() string {
	return reg.rpcIP
}
func (reg *ServiceReg) GetExtraInfo() string {
	var extra dis.GateExtra
	extra.Host = reg.gateInfo.GetPublicHost()
	bs, err := json.Marshal(extra)
	if err != nil {
		tilogs.L().Errorf("gate GetExtraInfo %+v json.Marshal err %s", extra, err.Error())
		return "{}"
	}
	return string(bs)
}
func (reg *ServiceReg) GetTickExtraInfo() string {
	ccu := allmetrics.GetCCUCount()
	info := consts.GateExtra{
		Ccu:    ccu,
		IsOpen: getIsOpen(),
	}
	bb, err := json.Marshal(info)
	if err != nil {
		tilogs.L().Errorf("gate GetTickExtraInfo json.Marshal err %s", err.Error())
		return ""
	}
	return string(bb)
}

// 用于gate缩容时关闭gate服务器, 1为开启0为关闭
var gateOpen = int32(1)

func DownLiner() {
	atomic.StoreInt32(&gateOpen, 0)
}

func getIsOpen() bool {
	return atomic.LoadInt32(&gateOpen) == 1
}
