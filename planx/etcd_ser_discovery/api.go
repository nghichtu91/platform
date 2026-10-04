package etcd_ser_discovery

import (
	"fmt"
	"strings"
	"github.com/nghichtu91/platform/share/planx/etcd"
)

// IsGamexAlive 判断gamex是否可用
// e.g /server_v3/51/service_be_discovery_alive/default/gamex/510166/alive
func IsGamexAlive(etcdServer string, gid, sid uint) bool {
	key := fmt.Sprintf("%s/%d/%s/%s/%s/%d/%s", etcdServer, gid,
		KeyBeDiscoveryAlive, VerEmpty,
		etcd.Server_Gamex, sid, KeyBeDiscoveryAliveAlive)

	v, err := etcd.Get(key)
	if err != nil {
		return false
	}

	return v == ValueYes
}

func IsGamexAliveGetKey(etcdServer string, gid, sid uint) string {
	key := fmt.Sprintf("%s/%d/%s/%s/%s/%d/%s", etcdServer, gid,
		KeyBeDiscoveryAlive, VerEmpty,
		etcd.Server_Gamex, sid, KeyBeDiscoveryAliveAlive)
	return key
}


// GetServiceAliveNum 当前在线服务数量
func GetServiceAliveNum(etcdServer string, gid uint, srvType string) (int, error) {
	key := fmt.Sprintf("%s/%d/%s/%s/%s", etcdServer, gid,
		KeyBeDiscoveryAlive, VerEmpty,
		srvType)

	kvs, err := etcd.GetSubRecursive(key)
	if err != nil {
		return 0, err
	}

	num := 0
	for k, v := range kvs {
		if !strings.HasSuffix(k, KeyBeDiscoveryAliveAlive) {
			continue
		}
		if v == ValueYes {
			num++
		}
	}

	return num, nil
}
