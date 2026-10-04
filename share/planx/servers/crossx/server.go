package crossx

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/etcd"
	discovery "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
)

var gSerReg *servDisForBattlex

type servDisForBattlex struct {
	serverId   string
	internalIp string
	serReg     *discovery.ServiceMgr
}

func StartRegServForBattlex(etcdServer string, gid uint, serverId, internalIp string) error {
	gSerReg = &servDisForBattlex{
		serverId:   serverId,
		internalIp: internalIp,
	}
	gSerReg.serReg = discovery.NewSeriveMgr(discovery.ServiceConfig{
		EtcdRoot: etcdServer,
		SerId: discovery.ServiceId{
			SerTyp: discovery.ServiceType{
				Gid:    fmt.Sprint(gid),
				SerTyp: etcd.Ser_Crossx,
			},
			SerId: serverId,
		},
		HasPrivateAddr: true,
		CheckHealth:    true,
	}, gSerReg)
	return gSerReg.serReg.Start()
}
func StopRegServForBattlex() {
	gSerReg.serReg.Stop()
}
func (s *servDisForBattlex) GetPublicAddr() string {
	return ""
}
func (s *servDisForBattlex) GetPrivateAddr() string {
	return fmt.Sprintf("%s:%s", s.internalIp, s.serverId) // 这里只用这个机制，让battle能感知cross的关闭，所以并不关心private的实际值，就用serverid了
}
func (s *servDisForBattlex) GetExtraInfo() string {
	return ""
}
func (s *servDisForBattlex) GetTickExtraInfo() string {
	return ""
}
