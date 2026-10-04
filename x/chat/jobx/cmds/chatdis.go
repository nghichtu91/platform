package cmds

import (
	"encoding/json"
	"fmt"
	"sync"

	comet "github.com/nghichtu91/platform/share/x/chat/api/discovery"
	"github.com/nghichtu91/platform/share/x/chat/jobx/config"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func CometDiscovery(etcd_root string, gid uint, handler comet.HandleCometDiscovery) {
	cometDiscovery = dis.NewDiscoveryMgr(dis.DiscoveryConfig{
		EtcdRoot: etcd_root,
		BeDiscoverySerTyp: dis.ServiceType{
			Gid:    fmt.Sprintf("%d", gid),
			SerTyp: etcd.Ser_ChatComet,
		},
		HasPrivateAddr: true,
	}, &cometCallBack{}, nil)

	disHandler = handler
}

func CometDiscoveryStart() error {
	if err := cometDiscovery.Start(); err != nil {
		return err
	}
	return nil
}

func CometDiscoveryStop() {
	cometDiscovery.Stop()
}

var (
	cometDiscovery   *dis.DiscoveryMgr
	CometServerInfos map[string]*comet.CometServerInfo
	lock             sync.RWMutex
	disHandler       comet.HandleCometDiscovery
)

func init() {
	CometServerInfos = make(map[string]*comet.CometServerInfo, 2)
}

type cometCallBack struct {
}

func (a *cometCallBack) OnAddService(info dis.ServiceInfo) {
	if info.SerId.SerId == config.Cfg.ServerId {
		return
	}

	tilogs.L().Infof("Add chat comet server %v", info)

	lock.Lock()
	defer lock.Unlock()

	serverInfo, ok := CometServerInfos[info.SerId.SerId]
	if !ok {
		serverInfo = &comet.CometServerInfo{}
		CometServerInfos[info.SerId.SerId] = serverInfo
	} else {
		//已经有一个连接了
		disHandler.OnServerDel(info.SerId.SerId)
	}

	extraInfo := &comet.CometServerExtraInfo{}
	err := json.Unmarshal([]byte(info.ExtraInfo), extraInfo)
	if err == nil {
		serverInfo.StartTime = extraInfo.ServerStartTime
		serverInfo.PrivateAddr = extraInfo.GrpcServerIp
	}

	extraTickInfo := &comet.CometServerExtraTickInfo{}
	err = json.Unmarshal([]byte(info.TickExtraInfo), extraTickInfo)
	if err == nil {
		serverInfo.IsOpen = extraTickInfo.IsOpen
	}

	// warn job发现前 comet上已经有人了 那么如果有消息会不会 丢失
	disHandler.OnServerAdd(info.SerId.SerId, serverInfo)
}

func (a *cometCallBack) OnDelService(serId dis.ServiceId) {
	if serId.SerId == config.Cfg.ServerId {
		return
	}

	tilogs.L().Infof("Del Chat Server %v", serId)
	lock.Lock()
	defer lock.Unlock()

	delete(CometServerInfos, serId.SerId)

	disHandler.OnServerDel(serId.SerId)
}

func (a *cometCallBack) OnChangeExtraInfo(info dis.ServiceInfo) {
	tilogs.L().Infof("chatCallBack OnChangeExtraInfo %v", info)

	lock.RLock()
	defer lock.RUnlock()

	serverInfo, ok := CometServerInfos[info.SerId.SerId]
	if !ok {
		return
	}

	if info.ExtraInfo != "" {
		extraInfo := &comet.CometServerExtraInfo{}
		err := json.Unmarshal([]byte(info.ExtraInfo), extraInfo)
		if err == nil {
			serverInfo.StartTime = extraInfo.ServerStartTime
		}
	}

	if info.TickExtraInfo != "" {
		extraTickInfo := &comet.CometServerExtraTickInfo{}
		err := json.Unmarshal([]byte(info.TickExtraInfo), extraTickInfo)
		if err == nil {
			serverInfo.IsOpen = extraTickInfo.IsOpen
		}
	}
}
