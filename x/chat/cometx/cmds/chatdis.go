package cmds

import (
	"encoding/json"
	"fmt"
	"sync"

	logic "github.com/nghichtu91/platform/share/x/chat/api/discovery"
	"github.com/nghichtu91/platform/share/x/chat/cometx/config"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func LogicDiscovery(etcd_root string, gid uint, handler logic.HandleLogicDiscovery) {
	logicDiscovery = dis.NewDiscoveryMgr(dis.DiscoveryConfig{
		EtcdRoot: etcd_root,
		BeDiscoverySerTyp: dis.ServiceType{
			Gid:    fmt.Sprintf("%d", gid),
			SerTyp: etcd.Ser_ChatLogic,
		},
		//BeDiscoverySerId: config.Cfg.LogicId,
		HasPrivateAddr: true,
	}, &logicCallBack{}, nil)

	disHandler = handler
}

func LogicDiscoveryStart() error {
	if err := logicDiscovery.Start(); err != nil {
		return err
	}
	return nil
}

func LogicDiscoveryStop() {
	logicDiscovery.Stop()
}

var (
	logicDiscovery   *dis.DiscoveryMgr
	logicServerInfos map[string]*logic.LogicServerInfo
	lock             sync.RWMutex
	disHandler       logic.HandleLogicDiscovery
)

func init() {
	logicServerInfos = make(map[string]*logic.LogicServerInfo, 2)
}

type logicCallBack struct {
}

func (a *logicCallBack) OnAddService(info dis.ServiceInfo) {
	// 过滤 只连指定的logic
	if info.SerId.SerId != config.Cfg.LogicId {
		return
	}

	tilogs.L().Infof("Add chat logic server %v", info)

	lock.Lock()
	defer lock.Unlock()

	serverInfo, ok := logicServerInfos[info.SerId.SerId]
	if !ok {
		serverInfo = &logic.LogicServerInfo{}
		logicServerInfos[info.SerId.SerId] = serverInfo
	} else {
		//已经有一个连接了
		disHandler.OnServerDel(info.SerId.SerId)
	}

	serverInfo.PrivateAddr = info.PrivateAddr
	extraInfo := &logic.LogicServerExtraInfo{}
	err := json.Unmarshal([]byte(info.ExtraInfo), extraInfo)
	if err == nil {
		serverInfo.StartTime = extraInfo.ServerStartTime
		serverInfo.HttpAddr = extraInfo.HttpServerIp
	}

	extraTickInfo := &logic.LogicServerExtraTickInfo{}
	err = json.Unmarshal([]byte(info.TickExtraInfo), extraTickInfo)
	if err == nil {
		serverInfo.IsOpen = extraTickInfo.IsOpen
	}

	// warn job发现前 logic上已经有人了 那么如果有消息会不会 丢失
	disHandler.OnServerAdd(info.SerId.SerId, serverInfo)
}

func (a *logicCallBack) OnDelService(serId dis.ServiceId) {
	// 过滤 只连指定的logic
	if serId.SerId != config.Cfg.LogicId {
		return
	}

	tilogs.L().Infof("Del Chat Server %v", serId)
	lock.Lock()
	defer lock.Unlock()

	delete(logicServerInfos, serId.SerId)

	disHandler.OnServerDel(serId.SerId)
}

func (a *logicCallBack) OnChangeExtraInfo(info dis.ServiceInfo) {
	tilogs.L().Infof("chatCallBack OnChangeExtraInfo %v", info)

	lock.RLock()
	defer lock.RUnlock()

	serverInfo, ok := logicServerInfos[info.SerId.SerId]
	if !ok {
		return
	}

	if info.ExtraInfo != "" {
		extraInfo := &logic.LogicServerExtraInfo{}
		err := json.Unmarshal([]byte(info.ExtraInfo), extraInfo)
		if err == nil {
			serverInfo.StartTime = extraInfo.ServerStartTime
		}
	}

	if info.TickExtraInfo != "" {
		extraTickInfo := &logic.LogicServerExtraTickInfo{}
		err := json.Unmarshal([]byte(info.TickExtraInfo), extraTickInfo)
		if err == nil {
			serverInfo.IsOpen = extraTickInfo.IsOpen
		}
	}
}
