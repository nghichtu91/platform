package cmds

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	cometDis "github.com/nghichtu91/platform/share/x/chat/api/discovery"
	"github.com/nghichtu91/platform/share/x/chat/cometx/config"
)

var etcdMgr *dis.ServiceMgr

type serverExtraProvider interface {
	GetConnNum() int32
	IsOpen() bool
}

var ServerExtraProvider serverExtraProvider

func CometEtcdReg(serverId, internal_addr, public_addr, grpc_addr string, provider serverExtraProvider) error {
	ServerExtraProvider = provider
	etcdMgr = dis.NewSeriveMgr(
		dis.ServiceConfig{
			EtcdRoot: config.Cfg.EtcdServer,
			SerId: dis.ServiceId{
				SerTyp: dis.ServiceType{
					Gid:    fmt.Sprintf("%d", config.Cfg.Gid),
					SerTyp: etcd.Ser_ChatComet,
				},
				SerId: serverId,
			},
			HasPrivateAddr: true,
			HasPublicAddr:  true,
			HasExtraInfo:   true,
			//HasTickExtraInfo: true,
			//CheckHealth:      true,
			//ExtraInfoTick:    time.Second * 5,
		}, &CometEtcdServiceInfo{
			internal_addr: internal_addr,
			public_addr:   public_addr,
			grpc_addr:     grpc_addr,
		})
	return etcdMgr.Start()
}

func CometEtcdStop() {
	tilogs.L().Infof(".......start CometEtcdStop.......")
	if etcdMgr != nil {
		etcdMgr.Stop()
		tilogs.L().Infof("CometEtcdStop stoped")
	}
}

type CometEtcdServiceInfo struct {
	internal_addr string
	public_addr   string
	grpc_addr     string
}

func (info *CometEtcdServiceInfo) GetPublicAddr() string {
	return info.public_addr
}

func (info *CometEtcdServiceInfo) GetPrivateAddr() string {
	return info.internal_addr
}

func (info *CometEtcdServiceInfo) GetExtraInfo() string {
	extraInfo := &cometDis.CometServerExtraInfo{}
	extraInfo.ServerStartTime = time.Now().Unix()
	extraInfo.GrpcServerIp = info.grpc_addr
	result, _ := json.Marshal(extraInfo)

	return string(result)
}

func (info *CometEtcdServiceInfo) GetTickExtraInfo() string {

	extraTickInfo := &cometDis.CometServerExtraTickInfo{}
	extraTickInfo.ConnNum = ServerExtraProvider.GetConnNum()
	extraTickInfo.IsOpen = ServerExtraProvider.IsOpen()

	result, _ := json.Marshal(extraTickInfo)

	return string(result)
}
