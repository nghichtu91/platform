package etcd_ser_discovery

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/planx/grpchealth"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type ServiceInfo struct {
	SerId ServiceId

	MatchTag []string // 匹配的tag
	HasTag   []string // 服务可提供的所有tag
	// addr一般是ip:port的格式
	PublicAddr  string
	PrivateAddr string
	HealthAddr  string

	GMSetOnline bool // 是否上线，若服务没有此功能，则此项一直为true
	Alive       bool // ttl set的，若expire证明已经挂了

	ExtraInfo     string
	TickExtraInfo string

	healthChecker *HealthChecker // health心跳检测

	lastOnline   bool             // 上次的在线状态
	setting      discoverySetting // 一些服务发现的设置
	chanStateChg chan etcdParam
	wait         *util.WaitGroupWrapper
	quit         chan struct{}

	lastUpdateKey string // for etcd update, 用来标记是哪个key更新
}

type discoverySetting struct {
	wantTags       []string
	tagMatchType   int
	hasPublicAddr  bool
	hasPrivateAddr bool
	hasExtraInfo   bool
	callBack       DiscoveryCallBack
}

type etcdParam struct {
	key   string // 若为空，则是health更新
	value string
	isDel bool
}

func (info *ServiceInfo) start() {
	info.wait.Wrap(func() {
		for {
			select {
			case param := <-info.chanStateChg:
				info.stateChg(param)
			case <-info.quit:
				tilogs.L().Infof("discover ServiceInfo %v quit", info.SerId)
				return
			}
		}
	})
}

func (info *ServiceInfo) onEtcdEtcd(param etcdParam) {
	t := timeutil.Timer10MS.After(time.Second)
	select {
	case info.chanStateChg <- param:
	case <-t:
		tilogs.L().Errorf("ServiceInfo onEtcdEtcd timeout, param %v", param)
	}
}

func (info *ServiceInfo) stateChg(param etcdParam) {
	setting := info.setting
	oldExtra := info.ExtraInfo
	oldTickExtra := info.TickExtraInfo
	if param.key != "" {
		if param.isDel {
			info.del(param.key, setting.wantTags)
		} else {
			info.update(param.key, param.value, setting.wantTags)
		}
	}
	info.SetUpdateKey(param.key) // 为空的话, 就是健康检查触发的.
	nowOnline := info.isOnline(setting)
	tilogs.L().Infof("DiscoveryMgr updateSerInfo info:%v, filters:%v old_online:%v isonline:%v",
		info, setting.wantTags, info.lastOnline, nowOnline)
	if nowOnline != info.lastOnline {
		if info.lastOnline {
			setting.callBack.OnDelService(info.SerId)
		} else {
			setting.callBack.OnAddService(*info)
			if info.ExtraInfo != "" || info.TickExtraInfo != "" {
				setting.callBack.OnChangeExtraInfo(*info)
			}
		}
		info.lastOnline = nowOnline
	}
	if (oldExtra != info.ExtraInfo ||
		oldTickExtra != info.TickExtraInfo) &&
		nowOnline {
		setting.callBack.OnChangeExtraInfo(*info)
	}
}

func (info *ServiceInfo) update(k, v string, wantTags []string) {
	switch k {
	case KeyBeDiscoveryPublic:
		info.PublicAddr = v
	case KeyBeDiscoveryPrivate:
		info.PrivateAddr = v
	case KeyBeDiscoveryHealth:
		if info.healthChecker != nil {
			info.healthChecker.stopHealth()
			info.healthChecker = nil
		}
		info.HealthAddr = v
		if info.HealthAddr != "" {
			info.healthChecker = checkHealth(info, info.HealthAddr)
		}
	case KeyBeDiscoveryOnline:
		if v == ValueYes {
			info.GMSetOnline = true
		} else {
			info.GMSetOnline = false
		}
	case KeyBeDiscoveryTickExtra:
		info.TickExtraInfo = v
	case KeyBeDiscoveryExtra:
		info.ExtraInfo = v
	case KeyBeDiscoveryFilterFlag:
		var res []string
		if err := json.Unmarshal([]byte(v), &res); err != nil {
			tilogs.L().Errorf("ServiceInfo update Unmarshal err %s %s %s", err.Error(), k, v)
			return
		}
		info.HasTag = res
		info.MatchTag = info.matchFilter(wantTags)
	case KeyBeDiscoveryAliveAlive:
		info.Alive = true
	default:
		tilogs.L().Errorf("ServiceInfo update k not define %s", k)
	}
}

func (info *ServiceInfo) del(k string, wantTags []string) {
	switch k {
	case KeyBeDiscoveryPublic:
		info.PublicAddr = ""
	case KeyBeDiscoveryPrivate:
		info.PrivateAddr = ""
	case KeyBeDiscoveryHealth:
		info.HealthAddr = ""
		if info.healthChecker != nil {
			info.healthChecker.stopHealth()
			info.healthChecker = nil
		}
	case KeyBeDiscoveryOnline:
		info.GMSetOnline = false
	case KeyBeDiscoveryExtra:
		info.ExtraInfo = ""
	case KeyBeDiscoveryTickExtra:
		info.TickExtraInfo = ""
	case KeyBeDiscoveryFilterFlag:
		info.HasTag = []string{}
		info.MatchTag = info.matchFilter(wantTags)
	case KeyBeDiscoveryAliveAlive:
		info.Alive = false
	default:
		tilogs.L().Errorf("ServiceInfo update k not define %s", k)
	}
}

func (info *ServiceInfo) matchFilter(wantTags []string) []string {
	matched := make([]string, 0, len(wantTags))
	for _, f := range wantTags {
		for _, bf := range info.HasTag {
			if bf == f {
				matched = append(matched, f)
			}
		}
	}
	return matched
}

func (info *ServiceInfo) isOnline(setting discoverySetting) bool {
	wantTags := setting.wantTags
	tagMatchType := setting.tagMatchType
	hasPublicAddr := setting.hasPublicAddr
	hasPrivateAddr := setting.hasPrivateAddr
	hasExtraInfo := setting.hasExtraInfo

	if !info.isMatched(wantTags, tagMatchType) {
		tilogs.L().Infof("server discovery %v:  !info.isMatched", info.SerId)
		return false
	}
	if !info.GMSetOnline {
		tilogs.L().Infof("server discovery %v:  !info.GMSetOnline", info.SerId)
		return false
	}
	if hasPublicAddr && info.PublicAddr == "" {
		tilogs.L().Infof("server discovery %v:  hasPublicAddr && info.PublicAddr == ", info.SerId)
		return false
	}
	if hasPrivateAddr && info.PrivateAddr == "" {
		tilogs.L().Infof("server discovery %v:  hasPrivateAddr && info.PrivateAddr == ", info.SerId)
		return false
	}
	if hasExtraInfo && info.ExtraInfo == "" {
		tilogs.L().Infof("server discovery %v:  hasExtraInfo && info.ExtraInfo == ", info.SerId)
		return false
	}
	/*
		alive和health只要有一个在线，则认为在线；
		alive和health都下线，才认为下线
	*/
	if !info.Alive {
		tilogs.L().Infof("server discovery %v:  !info.Alive", info.SerId)
		if info.healthChecker == nil {
			return false
		} else if !info.healthChecker.isHealth() {
			return false
		}
	}
	return true
}

func (info *ServiceInfo) isMatched(wantTags []string, tagMatchType int) bool {
	if len(wantTags) == 0 {
		return true
	} else {
		if tagMatchType == TagMatchType_All {
			return len(info.MatchTag) == len(wantTags)
		} else if tagMatchType == TagMatchType_Any {
			return len(wantTags) > 0
		}
		return false
	}
}

func (info *ServiceInfo) String() string {
	isHealth := false
	if info.healthChecker != nil {
		isHealth = info.healthChecker.isHealth()
	}
	return fmt.Sprintf("SerId:%v MatchTag:%v HasTag:%v PublicAddr:%v PrivateAddr:%v HealthAddr:%v "+
		"GMSetOnline:%v Alive:%v ExtraInfo:%v TickExtraInfo:%v isHealth:%v",
		info.SerId, info.MatchTag, info.HasTag, info.PublicAddr, info.PrivateAddr, info.HealthAddr,
		info.GMSetOnline, info.Alive, info.ExtraInfo, info.TickExtraInfo, isHealth)
}

// SetUpdateKey 设置更新的key，用来标记是哪个key更新
func (info *ServiceInfo) SetUpdateKey(key string) {
	info.lastUpdateKey = key
}

// GetUpdateKey 获取更新的key，用来标记是哪个key更新
func (info *ServiceInfo) GetUpdateKey() string {
	return info.lastUpdateKey
}

type HealthChecker struct {
	info       *ServiceInfo
	healthAddr string
	nHealth    int32 // is health, >0为健康

	chanStateChg chan etcdParam // 用来触发通知
	quitHealth   chan struct{}
}

func checkHealth(info *ServiceInfo, healthAddr string) *HealthChecker {
	if healthAddr != "" {
		hc := &HealthChecker{
			info:       info,
			healthAddr: healthAddr,
			quitHealth: make(chan struct{}, 1),
		}
		grpchealth.CheckGRPCHealth(hc.healthAddr, info.SerId.String(), hc.setHealth, hc.quitHealth)
		return hc
	}
	return nil
}

func (hc *HealthChecker) setHealth(isHealth bool) {
	isChg := false
	if isHealth {
		isChg = atomic.CompareAndSwapInt32(&hc.nHealth, 0, 1)
		// tilogs.L().Debugf("ServiceInfo serid:%v healthaddr:%s health true", hc.SerId, hc.HealthAddr)
	} else {
		isChg = atomic.CompareAndSwapInt32(&hc.nHealth, 1, 0)
		if isChg {
			tilogs.L().Infof("ServiceInfo serid:%v healthaddr:%s health false", hc.info.SerId, hc.healthAddr)
		}
	}
	if isChg {
		hc.info.onEtcdEtcd(etcdParam{})
	}
}

func (hc *HealthChecker) isHealth() bool {
	return atomic.LoadInt32(&hc.nHealth) > 0
}

func (hc *HealthChecker) stopHealth() {
	close(hc.quitHealth)
}
