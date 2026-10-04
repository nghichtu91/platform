package etcd_ser_discovery

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/etcd"
)

const (
	KeyBeDiscovery           = "service_be_discovery"
	KeyBeDiscoveryPublic     = "public_addr"
	KeyBeDiscoveryPrivate    = "private_addr"
	KeyBeDiscoveryHealth     = "health_addr"
	KeyBeDiscoveryGrpc       = "grpc_addr"
	KeyBeDiscoveryTickExtra  = "tick_extra"
	KeyBeDiscoveryExtra      = "extra"
	KeyBeDiscoveryOnline     = "online"
	KeyBeDiscoveryFilterFlag = "filter_flag"
	KeyBeDiscoveryAlive      = "service_be_discovery_alive"
	KeyBeDiscoveryAliveAlive = "alive"
	ValueYes                 = "yes"
	ValueNo                  = "no"

	TagMatchType_All = 0 // tag 匹配类型， 全部匹配
	TagMatchType_Any = 1 // tag 匹配类型，任何一个匹配
)

const VerEmpty = "default"
const aliveTick = 10

type ServiceType struct {
	Version string
	Gid     string
	SerTyp  etcd.ServiceTypeId
}

// String
func (s *ServiceType) String() string {
	return fmt.Sprintf("%s/%s/%s", s.Version, s.Gid, etcd.SerTyp2Name[s.SerTyp])
}

func (s *ServiceType) valid() bool {
	_, ok := etcd.SerTyp2Name[s.SerTyp]
	return s.Version != "" && s.SerTyp > 0 && s.SerTyp < etcd.UnValid && ok
}

// /a4k/202/service_discovery/gate
func (s *ServiceType) getEtcdPath(etcdRoot string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", etcdRoot,
		s.Gid, KeyBeDiscovery, s.Version, etcd.SerTyp2Name[s.SerTyp])
}

// /a4k/202/service_discovery_alive/gate
func (s *ServiceType) getEtcdAlivePath(etcdRoot string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", etcdRoot,
		s.Gid, KeyBeDiscoveryAlive, s.Version, etcd.SerTyp2Name[s.SerTyp])
}

func (s *ServiceType) GetAllServiceInfos(etcdRoot string) (map[string]*ServiceInfo, error) {
	res := make(map[string]*ServiceInfo, 4)
	_ret, err := etcd.GetSubRecursive(s.getEtcdPath(etcdRoot))
	if err != nil {
		return nil, err
	}
	for k, v := range _ret {
		k = strings.TrimPrefix(k, etcdRoot)
		ss := strings.Split(k, "/")
		if len(ss) < countOffset {
			continue
		}
		serId := ss[serIdOffset]
		info, ok := res[serId]
		if !ok {
			typ, has := etcd.SerName2Typ[ss[serTypeOffset]]
			if !has {
				tilogs.L().Errorf("ServiceType GetAllServiceInfos type not found %v", k)
				continue
			}
			info = &ServiceInfo{
				SerId: ServiceId{
					SerTyp: ServiceType{
						Version: ss[verOffset],
						Gid:     ss[gidOffset],
						SerTyp:  typ,
					},
					SerId: serId,
				},
			}
			res[serId] = info
		}
		switch ss[keyOffset] {
		case KeyBeDiscoveryPublic:
			info.PublicAddr = v
		case KeyBeDiscoveryPrivate:
			info.PrivateAddr = v
		case KeyBeDiscoveryOnline:
			if v == ValueYes {
				info.GMSetOnline = true
			} else {
				info.GMSetOnline = false
			}
		case KeyBeDiscoveryExtra:
			info.ExtraInfo = v
		case KeyBeDiscoveryTickExtra:
			info.TickExtraInfo = v
		}
	}
	return res, nil
}

type ServiceId struct {
	SerTyp ServiceType
	SerId  string
}

// String
func (s *ServiceId) String() string {
	return fmt.Sprintf("%s/%s", s.SerTyp.String(), s.SerId)
}

// SetOnline gm工具用，设置服务的上线；需要etcd先被初始化好
func (s *ServiceId) SetOnline(etcdRoot string) error {
	k := s.getEtcdKey(etcdRoot, KeyBeDiscoveryOnline)
	return etcd.Put(k, ValueYes)
}

// SetOffline gm工具用，设置服务的下线；需要etcd先被初始化好
func (s *ServiceId) SetOffline(etcdRoot string) error {
	k := s.getEtcdKey(etcdRoot, KeyBeDiscoveryOnline)
	return etcd.Put(k, ValueNo)
}

// SetBeDiscoveryTag gm工具用，设置"要被发现"的服务的tag，用于服务发现过滤用
func (s *ServiceId) SetBeDiscoveryTag(etcdRoot string, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	_b, err := json.Marshal(tags)
	if err != nil {
		tilogs.L().Errorf("ServiceId SetBeDiscoveryTag json.Marshal err %s", err.Error())
		return err
	}
	k := s.getEtcdKey(etcdRoot, KeyBeDiscoveryFilterFlag)
	return etcd.Put(k, string(_b))
}

// /a4k/202/service_discovery/gate/1/key
func (s *ServiceId) getEtcdKey(etcdRoot, key string) string {
	return fmt.Sprintf("%s/%s/%s",
		s.SerTyp.getEtcdPath(etcdRoot), s.SerId, key)
}

func (s *ServiceId) getEtcdAliveKey(etcdRoot, key string) string {
	return fmt.Sprintf("%s/%s/%s",
		s.SerTyp.getEtcdAlivePath(etcdRoot), s.SerId, key)
}

func parseServiceId(etcdRoot, key string) (*ServiceId, string) {
	key = strings.TrimPrefix(key, etcdRoot)
	ss := strings.Split(key, "/")
	if len(ss) != countOffset {
		tilogs.L().Errorf("parseServiceId key err  %s", key)
		return nil, ""
	}
	gid := ss[gidOffset]
	ver := ss[verOffset]
	serTyp := ss[serTypeOffset]
	id := ss[serIdOffset]
	k := ss[keyOffset]

	typ, ok := etcd.SerName2Typ[serTyp]
	if !ok {
		tilogs.L().Errorf("parseServiceId key but typ not found %s", key)
		return nil, ""
	}
	return &ServiceId{
		SerTyp: ServiceType{
			Version: ver,
			Gid:     gid,
			SerTyp:  typ,
		},
		SerId: id}, k
}

const (
	gidOffset     = iota + 1 // 1
	dirOffset                // 2
	verOffset                // 3
	serTypeOffset            // 4
	serIdOffset              // 5
	keyOffset                // 6
	countOffset              // 7
)
