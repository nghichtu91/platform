package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/nghichtu91/platform/share/x/common/notice"

	"github.com/nghichtu91/platform/share/planx/util"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

var (
	GonggaoInfos map[string]map[string]notice.MaintenanceInfo // gid->ver->info
	endPointMux  sync.RWMutex

	quit chan struct{}
)

func init() {
	quit = make(chan struct{}, 1)
}

func GetGonggao(gid, ver string) notice.MaintenanceInfo {
	defer tilogs.L().Debugf("GetGonggao end gid %v ver %v", gid, ver)
	tilogs.L().Debugf("GetGonggao start gid %v ver %v", gid, ver)
	endPointMux.RLock()
	defer endPointMux.RUnlock()
	vers, ok := GonggaoInfos[gid]
	if ok {
		return vers[ver]
	}
	return notice.MaintenanceInfo{}
}

func StartWatchGongGao(wait *util.WaitGroupWrapper) {
	rev := updateGonggao()

	watchKey := fmt.Sprintf("%s/%s/", config.Cfg.CommonCfg.EtcdServer, consts.KeyEndPoint)
	etcd.WatchWithRevNoPrevRetry(watchKey, rev, true, quit, wait, func(resp clientv3.WatchResponse) {
		for range resp.Events {
			updateGonggao()
		}
	})
}

func CloseWatchGongGao() {
	close(quit)
}

func updateGonggao() (rev int64) {
	path := fmt.Sprintf("%s/%s/", config.Cfg.CommonCfg.EtcdServer, consts.KeyEndPoint)
	kvs, rev, err := etcd.GetSubRecursiveWithRev(path)
	if err != nil {
		tilogs.L().Errorf("updateGonggao getallgid err %s", err.Error())
		return
	}
	infos := make(map[string]map[string]notice.MaintenanceInfo, 4)
	for k, v := range kvs {
		k = strings.TrimPrefix(k, path)
		pg := strings.Split(k, "/") // gid/version
		gid := pg[0]
		version := pg[1]

		vers, ok := infos[gid]
		if !ok {
			vers = make(map[string]notice.MaintenanceInfo, 4)
			infos[gid] = vers
		}

		etcdValue := &notice.MaintenanceInfo{}
		json.Unmarshal([]byte(v), etcdValue)
		infos[gid][version] = *etcdValue
	}

	endPointMux.Lock()
	GonggaoInfos = infos
	endPointMux.Unlock()
	tilogs.L().Infof("updateGonggao %v, revision %v", infos, rev)
	return
}
