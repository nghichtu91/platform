package shard

import (
	"errors"
	"fmt"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

func LoadShardInfo(etcdDevops, etcdServer, gid string) map[string]consts.ShardInfoAll {
	info, _ := LoadShardInfoWithRev(etcdDevops, etcdServer, gid)
	return info
}

func LoadShardInfoWithRev(etcdDevops, etcdServer, gid string) (_ map[string]consts.ShardInfoAll, rev int64) {
	devopsKey := fmt.Sprintf("%s/%s/gamex", etcdDevops, gid)
	devopsInfo := make(map[string]consts.ShardInfoInDevops, 1024)
	err := etcd.Bind(devopsKey, devopsInfo)
	if err != nil {
		tilogs.L().Errorf(" %s load devops shard info err 1 %v", devopsKey, err)
		return
	}

	serverKey := fmt.Sprintf("%s/%s/shards", etcdServer, gid)
	serverInfo := make(map[string]consts.ShardInfoInServer, 1024)
	rev, err = etcd.BindWithRev(serverKey, serverInfo)
	if err != nil && !errors.Is(err, etcd.EmptyKvErr) {
		tilogs.L().Errorf("%s load server shard info err 2 %v", serverKey, err)
		return
	}
	shardAllInfo := make(map[string]consts.ShardInfoAll, 1024)
	for k, v := range devopsInfo {
		v1 := consts.ShardInfoAll{}
		v1.ShardInfoInDevops = v
		if v2, ok := serverInfo[k]; ok {
			v1.ShardInfoInServer = v2
		}
		shardAllInfo[k] = v1
	}
	tilogs.L().Debugf("{devops: %s, server: %s} load shard info %v, rev %v",
		devopsKey, serverKey, shardAllInfo, rev)
	return shardAllInfo, rev
}
