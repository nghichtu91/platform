package battlex

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	discovery "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
)

// 战斗服回调
type ILogicBattleAgent interface {
	OnServStop(serId, serType string)
	OnServAdd(serId, serType, battleIpPort string)
}

type battle_agent struct {
	etcdServer       string
	gid              string
	battleAgentSerId string
	privateIp        string
	publicIpPort     string
	logicBattleAgent ILogicBattleAgent
	serReg           *etcd_ser_discovery.ServiceMgr
	disBattle        *etcd_ser_discovery.DiscoveryMgr
	region           string
	domainName       string
}

func NewBattleAgentSer(etcdServer string, gid, battleAgentSerId, privateIp, publicIpPort, region, domainName string, logicBattle ILogicBattleAgent) *battle_agent {
	return &battle_agent{
		etcdServer:       etcdServer,
		gid:              gid,
		battleAgentSerId: battleAgentSerId,
		privateIp:        privateIp,
		publicIpPort:     publicIpPort,
		logicBattleAgent: logicBattle,
		region:           region,
		domainName:       domainName,
	}
}

func (bSer *battle_agent) Run() error {
	// cross、gamex和battle之间能感知关服，做服务发现
	if err := bSer.regServ(); err != nil {
		return err
	}
	if err := bSer.discoverBattlex(); err != nil {
		return err
	}
	return nil
}

func (bSer *battle_agent) Downline() {

}

func (bSer *battle_agent) Stop() {
	bSer.serReg.Stop()
	bSer.disBattle.Stop()
}

// 注册服务器发现，注册战斗服代理
func (bSer *battle_agent) regServ() error {
	bSer.serReg = discovery.NewSeriveMgr(discovery.ServiceConfig{
		EtcdRoot: bSer.etcdServer,
		SerId: etcd_ser_discovery.ServiceId{
			SerTyp: etcd_ser_discovery.ServiceType{
				Gid:    bSer.gid,
				SerTyp: etcd.Ser_BattleAgent,
			},
			SerId: bSer.battleAgentSerId,
		},
		HasPrivateAddr: true,
		CheckHealth:    true,
		HasExtraInfo:   true,
	}, bSer)
	return bSer.serReg.Start()
}

//发现battle
func (bSer *battle_agent) discoverBattlex() error {
	bSer.disBattle = discovery.NewDiscoveryMgr(discovery.DiscoveryConfig{
		EtcdRoot: bSer.etcdServer,
		BeDiscoverySerTyp: discovery.ServiceType{
			Gid:    bSer.gid,
			SerTyp: etcd.Ser_Battle,
		},
		HasPrivateAddr: true,
	}, bSer, nil)
	if bSer.disBattle == nil {
		return fmt.Errorf("battle_server discoverBattlex NewDiscoveryMgr fail")
	}
	return bSer.disBattle.Start()
}

func (bSer *battle_agent) GetPublicAddr() string {
	return ""
}

//拼接ip 和 serverId
func (bSer *battle_agent) GetPrivateAddr() string {
	return fmt.Sprintf("%s:%s", bSer.privateIp, bSer.battleAgentSerId)
}

//提供外网连接的ip 和端口
func (bSer *battle_agent) GetExtraInfo() string {
	return fmt.Sprintf("%s:%s:%s", bSer.publicIpPort, bSer.region, bSer.domainName)
}
func (bSer *battle_agent) GetTickExtraInfo() string {
	return ""
}

func (bSer *battle_agent) OnAddService(info discovery.ServiceInfo) {
	tilogs.L().Infof("OnAddService:%v", info)
	if info.SerId.SerTyp.SerTyp == etcd.Ser_Battle {
		bSer.logicBattleAgent.OnServAdd(info.PrivateAddr, etcd.Server_Battle, info.ExtraInfo)
	}
}
func (bSer *battle_agent) OnDelService(serId discovery.ServiceId) {
	if serId.SerTyp.SerTyp == etcd.Ser_Battle {
		bSer.logicBattleAgent.OnServStop(serId.SerId, etcd.Server_Battle)
	}
}

func (bSer *battle_agent) OnChangeExtraInfo(info discovery.ServiceInfo) {

}
