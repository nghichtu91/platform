package cmds

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	cometDis "github.com/nghichtu91/platform/share/x/chat/api/discovery"
	"github.com/nghichtu91/platform/share/x/chat/logicx/config"
)

var etcdMgr *dis.ServiceMgr

type serverExtraProvider interface {
	GetConnNum() int32
	IsOpen() bool
}

var ServerExtraProvider serverExtraProvider

func LogicEtcdReg(serverId, internalAddr, httpAddr string, provider serverExtraProvider) error {
	ServerExtraProvider = provider
	etcdMgr = dis.NewSeriveMgr(
		dis.ServiceConfig{
			EtcdRoot: config.Cfg.EtcdServer,
			SerId: dis.ServiceId{
				SerTyp: dis.ServiceType{
					Gid:    fmt.Sprintf("%d", config.Cfg.Gid),
					SerTyp: etcd.Ser_ChatLogic,
				},
				SerId: serverId,
			},
			HasPrivateAddr:   true,
			HasPublicAddr:    true,
			HasExtraInfo:     true,
			HasTickExtraInfo: true,
			CheckHealth:      true,
			ExtraInfoTick:    time.Second * 5,
		}, &LogicEtcdServiceInfo{
			internalAddr: internalAddr,
			publicAddr:   "",
			httpAddr:     httpAddr,
		})
	return etcdMgr.Start()
}

func LogicEtcdStop() {
	tilogs.L().Infof(".......start LogicEtcdStop.......")
	if etcdMgr != nil {
		etcdMgr.Stop()
		tilogs.L().Infof("LogicEtcdStop stoped")
	}
}

type LogicEtcdServiceInfo struct {
	internalAddr string
	publicAddr   string
	httpAddr     string
}

func (info *LogicEtcdServiceInfo) GetPublicAddr() string {
	return info.publicAddr
}

func (info *LogicEtcdServiceInfo) GetPrivateAddr() string {
	return info.internalAddr
}

func (info *LogicEtcdServiceInfo) GetExtraInfo() string {
	extraInfo := &cometDis.LogicServerExtraInfo{}
	extraInfo.ServerStartTime = time.Now().Unix()
	extraInfo.HttpServerIp = info.httpAddr
	result, _ := json.Marshal(extraInfo)
	return string(result)
}

func (info *LogicEtcdServiceInfo) GetTickExtraInfo() string {
	extraTickInfo := &cometDis.LogicServerExtraTickInfo{}
	extraTickInfo.ConnNum = ServerExtraProvider.GetConnNum()
	extraTickInfo.IsOpen = ServerExtraProvider.IsOpen()
	result, _ := json.Marshal(extraTickInfo)
	return string(result)
}
