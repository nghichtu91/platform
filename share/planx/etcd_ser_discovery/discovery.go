package etcd_ser_discovery

import (
	"fmt"
	"strings"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

/*
DiscoveryCallBack 这个文件是在服务器发现关系中，"发现别的服务的一方"
会在被发现方服务启动，关闭，变化时，将符合tag过滤的服务调用回调
为了防止网络抖动等情况，etcd的set没法及时更新，导致服务错误的被认为下线等情况。引入grpc的health checking机制。此discovery端是client发请求端
*/
type DiscoveryCallBack interface {
	OnAddService(info ServiceInfo)
	OnDelService(serId ServiceId)
	OnChangeExtraInfo(info ServiceInfo)
}

type DiscoveryConfig struct {
	EtcdRoot string

	BeDiscoverySerTyp ServiceType // 被发现的服务类型
	BeDiscoverySerId  string      // 发现指定的serId

	HasPublicAddr  bool
	HasPrivateAddr bool
	HasExtraInfo   bool // 是否获取到额外信息才算上线
	HasGMOnline    bool // 是否可以gm控制开启/关闭

	TagMatchType int
}

type DiscoveryMgr struct {
	conf            DiscoveryConfig
	callBack        DiscoveryCallBack
	sers            map[ServiceId]*ServiceInfo
	ignoreServerIds map[ServiceId]bool
	wantTags        []string
	tagMatchType    int // 如：是全匹配 or 匹配任何一个

	lSers sync.RWMutex
	wait  util.WaitGroupWrapper
	quit  chan struct{}
}

func NewDiscoveryMgr(conf DiscoveryConfig, callBack DiscoveryCallBack, wantTags []string) *DiscoveryMgr {
	if conf.BeDiscoverySerTyp.Version == "" {
		conf.BeDiscoverySerTyp.Version = VerEmpty
	}
	if !conf.BeDiscoverySerTyp.valid() {
		tilogs.L().Errorf("NewDiscoveryMgr conf BeDiscoverySerTyp unvalid %v", conf)
		return nil
	}
	if !conf.HasPrivateAddr && !conf.HasPublicAddr {
		tilogs.L().Errorf("NewDiscoveryMgr HasPrivateAddr and HasPublicAddr all false, %v", conf)
		return nil
	}
	if wantTags == nil {
		wantTags = []string{}
	}
	tilogs.L().Infof("NewDiscoveryMgr filters %v", wantTags)
	return &DiscoveryMgr{
		conf:            conf,
		callBack:        callBack,
		sers:            make(map[ServiceId]*ServiceInfo, 16),
		wantTags:        wantTags,
		quit:            make(chan struct{}, 1),
		tagMatchType:    conf.TagMatchType,
		ignoreServerIds: map[ServiceId]bool{},
	}
}

func (mgr *DiscoveryMgr) AddIgnoreDiscoverSerId(serId ServiceId) {
	mgr.ignoreServerIds[serId] = true
}

func (mgr *DiscoveryMgr) Start() error {
	c := mgr.conf
	path := c.BeDiscoverySerTyp.getEtcdPath(c.EtcdRoot)
	tilogs.L().Debugf("server discovery watch path %s", path)
	pathAlive := c.BeDiscoverySerTyp.getEtcdAlivePath(c.EtcdRoot)
	if c.BeDiscoverySerId != "" {
		path = fmt.Sprintf("%s/%s", path, c.BeDiscoverySerId)
		pathAlive = fmt.Sprintf("%s/%s", pathAlive, c.BeDiscoverySerId)
	}
	// get all
	kvs, rev, err := etcd.GetSubRecursiveWithRev(path)
	if err != nil {
		tilogs.L().Errorf("DiscoveryMgr etcd3.GetSubRecursive err %s", err.Error())
		return err
	}
	for k, v := range kvs {
		mgr.updateSerInfo(k, v, false)
	}

	kvsAlive, revAlive, err := etcd.GetSubRecursiveWithRev(pathAlive)
	if err != nil {
		tilogs.L().Errorf("DiscoveryMgr etcd3.GetSubRecursive err %s", err.Error())
		return err
	}
	for k, v := range kvsAlive {
		mgr.updateSerInfo(k, v, false)
	}

	// watch
	etcd.WatchWithRevRetry(path, rev, false, true, mgr.quit, &mgr.wait, func(resp clientv3.WatchResponse) {
		mgr.watchEvent(&resp)
	})
	etcd.WatchWithRevRetry(pathAlive, revAlive, false, true, mgr.quit, &mgr.wait, func(resp clientv3.WatchResponse) {
		mgr.watchEvent(&resp)
	})

	return nil
}

func (mgr *DiscoveryMgr) watchEvent(resp *clientv3.WatchResponse) {
	for _, ev := range resp.Events {
		if ev == nil {
			continue
		}
		key := string(ev.Kv.Key)
		switch ev.Type {
		case clientv3.EventTypePut:
			mgr.updateSerInfo(key, string(ev.Kv.Value), false)
		case clientv3.EventTypeDelete:
			mgr.updateSerInfo(key, "", true)
		}
		// 忽略 tickExtraInfo
		if !strings.Contains(key, KeyBeDiscoveryTickExtra) {
			tilogs.L().Infof("watchEvent %s current %s prev %s",
				ev.Type, etcd.FormatKv(ev.Kv), etcd.FormatKv(ev.PrevKv))
		}
	}
}

func (mgr *DiscoveryMgr) Stop() {
	close(mgr.quit)
	mgr.wait.Wait()
}

func (mgr *DiscoveryMgr) updateSerInfo(key, value string, isDel bool) {
	serId, k := parseServiceId(mgr.conf.EtcdRoot, key)
	if serId == nil {
		tilogs.L().Errorf("DiscoveryMgr updateSerInfo err %s %s", key, value)
		return
	}

	_, isIgnore := mgr.ignoreServerIds[*serId]
	if isIgnore {
		return
	}

	mgr.lSers.RLock()
	info, ok := mgr.sers[*serId]
	mgr.lSers.RUnlock()
	if !ok {
		mgr.lSers.Lock()
		info, ok = mgr.sers[*serId]
		if !ok {
			info = &ServiceInfo{
				SerId:       *serId,
				HasTag:      []string{},
				GMSetOnline: !mgr.conf.HasGMOnline,
				setting: discoverySetting{
					wantTags:       mgr.wantTags,
					tagMatchType:   mgr.tagMatchType,
					hasPublicAddr:  mgr.conf.HasPublicAddr,
					hasPrivateAddr: mgr.conf.HasPrivateAddr,
					hasExtraInfo:   mgr.conf.HasExtraInfo,
					callBack:       mgr.callBack,
				},
				chanStateChg: make(chan etcdParam, 16),
				wait:         &mgr.wait,
				quit:         mgr.quit,
			}
			mgr.sers[*serId] = info
			info.start()
		}
		mgr.lSers.Unlock()
	}
	// notify
	info.onEtcdEtcd(etcdParam{
		key:   k,
		value: value,
		isDel: isDel,
	})
}

// GetSrvNum 获取当前可用的服务数量
func (mgr *DiscoveryMgr) GetSrvNum() int {
	mgr.lSers.RLock()
	defer mgr.lSers.RUnlock()
	return len(mgr.sers)
}
