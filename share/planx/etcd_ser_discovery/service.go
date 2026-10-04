package etcd_ser_discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	net2 "github.com/nghichtu91/platform/share/planx/net"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/planx/grpchealth"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/util"
)

/*
	这个文件是在服务器发现关系中，"要被发现的一方"
	注册自己的地址，附加信息，tag等
	其中tag只提供gm修改的接口，在const文件中；tag的过滤方式是只要有一个符合的tag就在通告之列
	注意：要保证服务正常关闭和异常关闭都要调用Stop
	为了防止网络抖动等情况，etcd的set没法及时更新，导致服务错误的被认为下线等情况。引入grpc的health checking机制。此service端是监听端
*/

var (
	ErrDuplicateAliveSerID = errors.New("duplicate alive serId found")
)

type GetServiceInfo interface {
	GetPublicAddr() string // addr一般是ip:port的格式
	GetPrivateAddr() string
	GetExtraInfo() string
	GetTickExtraInfo() string
}

type ServiceConfig struct {
	EtcdRoot string
	SerId    ServiceId

	HasPublicAddr  bool
	HasPrivateAddr bool

	HasExtraInfo bool

	HasTickExtraInfo bool
	ExtraInfoTick    time.Duration

	Tags []string // 提供匹配的TAGS

	CheckHealth bool // 是否提供health检查
}

type ServiceMgr struct {
	conf    ServiceConfig
	getInfo GetServiceInfo

	wait *util.WaitGroupWrapper
	quit chan struct{}
}

func NewSeriveMgr(conf ServiceConfig, getInfo GetServiceInfo) *ServiceMgr {
	if conf.SerId.SerTyp.Version == "" {
		conf.SerId.SerTyp.Version = VerEmpty
	}
	if !conf.SerId.SerTyp.valid() {
		tilogs.L().Errorf("NewSeriveMgr serTyp invalid %v", conf.SerId)
		return nil
	}
	if conf.HasTickExtraInfo && conf.ExtraInfoTick <= 0 {
		tilogs.L().Errorf("NewSeriveMgr conf.HasTickExtraInfo && conf.ExtraInfoTick <= 0")
		return nil
	}
	if conf.ExtraInfoTick > 0 && conf.ExtraInfoTick < 5*time.Second {
		tilogs.L().Errorf("NewSeriveMgr ExtraInfoTick < 5s %v", conf.ExtraInfoTick)
		return nil
	}
	return &ServiceMgr{
		conf:    conf,
		getInfo: getInfo,
		wait:    &util.WaitGroupWrapper{},
		quit:    make(chan struct{}, 1),
	}
}

func (mgr *ServiceMgr) genKey(key string) string {
	return mgr.conf.SerId.getEtcdKey(mgr.conf.EtcdRoot, key)
}

func (mgr *ServiceMgr) Start() error {
	keyPublic := mgr.genKey(KeyBeDiscoveryPublic)
	keyPrivate := mgr.genKey(KeyBeDiscoveryPrivate)
	keyExtra := mgr.genKey(KeyBeDiscoveryExtra)
	keyTickExtra := mgr.genKey(KeyBeDiscoveryTickExtra)
	keyAlive := mgr.conf.SerId.getEtcdAliveKey(mgr.conf.EtcdRoot, KeyBeDiscoveryAliveAlive)
	keyTag := mgr.genKey(KeyBeDiscoveryFilterFlag)
	keyHealth := mgr.genKey(KeyBeDiscoveryHealth)

	// 检查线上是否存在相同ID的keepAlive键值
	v, err := etcd.Get(keyAlive)
	if err != nil {
		tilogs.L().Errorf("ServiceMgr etcd.Get key %s err %s", keyAlive, err.Error())
		return err
	}
	if v == ValueYes || v == ValueNo {
		tilogs.L().Warnf("ServiceMgr duplicate serID %s alive found", mgr.conf.SerId.SerId)
		// return ErrDuplicateAliveSerID
	}

	etcd.Delete(keyExtra)
	etcd.Delete(keyPublic)
	etcd.Delete(keyPrivate)
	etcd.Delete(keyTickExtra)
	etcd.Delete(keyAlive)
	etcd.Delete(keyHealth)

	if mgr.conf.HasExtraInfo {
		if err := etcd.Put(keyExtra, mgr.getInfo.GetExtraInfo()); err != nil {
			tilogs.L().Errorf("ServiceMgr etcd.Put key %s err %s", keyExtra, err.Error())
			return err
		}
	}
	if mgr.conf.HasPublicAddr {
		if err := etcd.Put(keyPublic, mgr.getInfo.GetPublicAddr()); err != nil {
			tilogs.L().Errorf("ServiceMgr etcd.Put key %s err %s", keyPublic, err.Error())
			return err
		}
	}
	if mgr.conf.HasPrivateAddr {
		if err := etcd.Put(keyPrivate, mgr.getInfo.GetPrivateAddr()); err != nil {
			tilogs.L().Errorf("ServiceMgr etcd.Put key %s err %s", keyPrivate, err.Error())
			return err
		}
	}
	if mgr.conf.HasTickExtraInfo {
		if err := etcd.Put(keyTickExtra, mgr.getInfo.GetTickExtraInfo()); err != nil {
			tilogs.L().Errorf("ServiceMgr etcd.Put key %s err %s", keyTickExtra, err.Error())
			return err
		}
		if mgr.conf.ExtraInfoTick > 0 {
			mgr.wait.Wrap(func() {
				t := timeutil.TimerSec.After(mgr.conf.ExtraInfoTick - 1)
				for {
					select {
					case <-t:
						if err := etcd.Put(keyTickExtra, mgr.getInfo.GetTickExtraInfo()); err != nil {
							tilogs.L().Warnf("ServiceMgr etcd.Put key %s err %s", keyTickExtra, err.Error())
						}
						t = timeutil.TimerSec.After(mgr.conf.ExtraInfoTick - 1)
					case <-mgr.quit:
						return
					}
				}
			})
		}
	}

	if len(mgr.conf.Tags) > 0 {
		tagJson, err := json.Marshal(mgr.conf.Tags)
		if err != nil {
			tilogs.L().Errorf("json marshal tags error %v", err)
			return err
		}
		if err := etcd.Put(keyTag, string(tagJson)); err != nil {
			tilogs.L().Errorf("ServiceMgr etcd.Put key %s err %s", keyExtra, err.Error())
			return err
		}
		tilogs.L().Debugf("put tags %s = %s", keyTag, string(tagJson))
	}

	// alive
	if err := etcd.LeaseKeepAlive(keyAlive, ValueYes, aliveTick, mgr.quit); err != nil {
		tilogs.L().Errorf("ServiceMgr etcd.KeepAlive err %s", err.Error())
		return err
	}

	// grpc health checking server
	if mgr.conf.CheckHealth && mgr.conf.HasPrivateAddr {
		lis := net2.TryListenForServerHealth()
		if lis == nil {
			return fmt.Errorf("ServiceMgr TryListenForServerHealth don't found listen port")
		}
		grpchealth.GRPCHealthServer(lis, mgr.conf.SerId.String())
		// put health value to etcd
		_, port, err := net.SplitHostPort(lis.Addr().String())
		if err != nil {
			tilogs.L().Errorf("ServiceMgr net.SplitHostPort lis.Addr %s err %s", lis.Addr().String(), err.Error())
			return err
		}
		host, _, err := net.SplitHostPort(mgr.getInfo.GetPrivateAddr())
		if err != nil {
			tilogs.L().Errorf("ServiceMgr net.SplitHostPort GetPrivateAddr %s err %s", mgr.getInfo.GetPrivateAddr(), err.Error())
			return err
		}
		healthAddr := net.JoinHostPort(host, port)

		// key_health从带ttl改为不带ttl的了
		// 若etcd出问题了，出现象上看带ttl的key会失效，所以key_health不能设置为带ttl的键，因为会引起health也关闭，双重保证就没意义了
		if err := etcd.Put(keyHealth, healthAddr); err != nil {
			tilogs.L().Errorf("ServiceMgr etcd.Put key %s err %s", keyHealth, err.Error())
			return err
		}
		tilogs.L().Debugf("ServiceMgr reg health addr %s", healthAddr)
	}
	return nil
}

func (mgr *ServiceMgr) Stop() {
	close(mgr.quit)
	mgr.wait.Wait()

	if mgr.conf.HasPublicAddr {
		keyPublic := mgr.genKey(KeyBeDiscoveryPublic)
		err := etcd.Delete(keyPublic)
		if err != nil {
			tilogs.L().Errorf("etcd del %s err %s", keyPublic, err.Error())
		}
	}
	if mgr.conf.HasPrivateAddr {
		keyPrivate := mgr.genKey(KeyBeDiscoveryPrivate)
		err := etcd.Delete(keyPrivate)
		if err != nil {
			tilogs.L().Errorf("etcd del %s err %s", keyPrivate, err.Error())
		}
		keyHealth := mgr.genKey(KeyBeDiscoveryHealth)
		err = etcd.Delete(keyHealth)
		if err != nil {
			tilogs.L().Errorf("etcd del %s err %s", keyHealth, err.Error())
		}
	}
	if mgr.conf.HasExtraInfo {
		keyExtra := mgr.genKey(KeyBeDiscoveryExtra)
		err := etcd.Delete(keyExtra)
		if err != nil {
			tilogs.L().Errorf("etcd del %s err %s", keyExtra, err.Error())
		}
	}
	if mgr.conf.HasTickExtraInfo {
		keyTickExtra := mgr.genKey(KeyBeDiscoveryTickExtra)
		err := etcd.Delete(keyTickExtra)
		if err != nil {
			tilogs.L().Errorf("etcd del %s err %s", keyTickExtra, err.Error())
		}
	}

	// 服务停止删除keepAlive键值，避免restart失败
	keyAlive := mgr.conf.SerId.getEtcdAliveKey(mgr.conf.EtcdRoot, KeyBeDiscoveryAliveAlive)
	err := etcd.Delete(keyAlive)
	if err != nil {
		tilogs.L().Errorf("etcd del %s err %s", keyAlive, err.Error())
	}
}

// UpdateExtra 单独更新extra的接口
func (mgr *ServiceMgr) UpdateExtra() error {
	if mgr.conf.HasExtraInfo {
		keyExtra := mgr.genKey(KeyBeDiscoveryExtra)
		if err := etcd.Put(keyExtra, mgr.getInfo.GetExtraInfo()); err != nil {
			tilogs.L().Errorf("ServiceMgr etcd.Put key %s err %s", keyExtra, err.Error())
			return err
		}
	}

	return nil
}
