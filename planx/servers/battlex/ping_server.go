package battlex

import (
	"fmt"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
)

type ping_server struct {
	etcdServer   string
	gid          string
	pingServerId string
	privateIp    string
	publicIpPort string
	serReg       *etcd_ser_discovery.ServiceMgr
	disBattle    *etcd_ser_discovery.DiscoveryMgr
	region       string
	domainName   string
}

func NewPingServerSer(etcdServer string, gid, pingServerId, privateIp, publicIpPort, region, domainName string) *ping_server {
	return &ping_server{
		etcdServer:   etcdServer,
		gid:          gid,
		pingServerId: pingServerId,
		privateIp:    privateIp,
		publicIpPort: publicIpPort,
		region:       region,
		domainName:   domainName,
	}
}

func (bSer *ping_server) Run() error {
	// 只注册，能被其他服务发现（gamex）
	if err := bSer.regServ(); err != nil {
		return err
	}
	return nil
}

func (bSer *ping_server) Downline() {

}

func (bSer *ping_server) Stop() {
	bSer.serReg.Stop()
}

// 注册服务器发现，注册战斗服代理
func (bSer *ping_server) regServ() error {
	bSer.serReg = etcd_ser_discovery.NewSeriveMgr(etcd_ser_discovery.ServiceConfig{
		EtcdRoot: bSer.etcdServer,
		SerId: etcd_ser_discovery.ServiceId{
			SerTyp: etcd_ser_discovery.ServiceType{
				Gid:    bSer.gid,
				SerTyp: etcd.Ser_PingServer,
			},
			SerId: bSer.pingServerId,
		},
		HasPrivateAddr: true,
		CheckHealth:    true,
		HasExtraInfo:   true,
	}, bSer)
	return bSer.serReg.Start()
}

func (bSer *ping_server) GetPublicAddr() string {
	return ""
}

//拼接ip 和 serverId
func (bSer *ping_server) GetPrivateAddr() string {
	return fmt.Sprintf("%s:%s", bSer.privateIp, bSer.pingServerId)
}

//提供外网连接的ip 和端口
func (bSer *ping_server) GetExtraInfo() string {
	return fmt.Sprintf("%s:%s:%s", bSer.publicIpPort, bSer.region, bSer.domainName)
}

func (bSer *ping_server) GetTickExtraInfo() string {
	return ""
}

func (bSer *ping_server) OnChangeExtraInfo(info etcd_ser_discovery.ServiceInfo) {

}
