package models

import (
	"strings"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/auth/config"
)

const (
	IPTYPE       = 1
	DEVICEIDTYPE = 2
	SDKIDTYPE    = 3
)

func InsertOrUpdateWhiteList(infos *WhiteListInfo) ([]*WhiteListInfo, error) {
	return db_interface.InsertOrUpdateWhiteList(infos)

}

func DeleteWhiteList(content string, typ uint) ([]*WhiteListInfo, error) {
	return db_interface.DeleteWhiteList(content, typ)
}

func QueryAllWhiteList() ([]*WhiteListInfo, error) {
	return db_interface.QueryAllWhiteList()
}

func QueryWhiteList(typ uint, param string) ([]*WhiteListInfo, error) {
	return db_interface.QueryWhiteList(typ, param)
}

func IsWhiteListForCurrentUser(devicdId string, ip string, sdkId string) bool {
	// 固定判断三种类型
	ret, err := QueryWhiteList(DEVICEIDTYPE, devicdId)
	if err == nil && len(ret) > 0 {
		tilogs.L().Infof("[WhiteList.IsWhiteList] devicdId check succ %s", devicdId)
		return true
	}

	ret, err = QueryWhiteList(SDKIDTYPE, sdkId)
	if err == nil && len(ret) > 0 {
		tilogs.L().Infof("[WhiteList.IsWhiteList] sdkId check succ %s", sdkId)
		return true
	}
	ret, err = QueryWhiteList(IPTYPE, ip)
	if err == nil && len(ret) > 0 {
		tilogs.L().Infof("[WhiteList.IsWhiteList] ip check succ %s", ip)
		return true
	}

	return false
}

var (
	WhiteListPwdInfos map[string]map[string]string // gid->ver->pwd
	lock              sync.RWMutex

	watchWLPwdQuit chan struct{}
)

func GetWhiteListPwd(gid, ver string) string {
	defer tilogs.L().Debugf("GetWhiteListPwd end gid %v ver %v", gid, ver)
	tilogs.L().Debugf("GetWhiteListPwd start gid %v ver %v", gid, ver)
	lock.RLock()
	defer lock.RUnlock()
	vers, ok := WhiteListPwdInfos[gid]
	if ok {
		return vers[ver]
	}
	return ""
}

func init() {
	watchWLPwdQuit = make(chan struct{}, 1)
}

func CloseWatchWhiteListPwd() {
	close(watchWLPwdQuit)
}

func StartWatchWhiteListPwd(wait *util.WaitGroupWrapper) {
	rev := updateWhiteListPwd()

	watchKey := etcd.GetVersionWLPwdWatchUpdateKey(config.Cfg.CommonCfg.EtcdServer)
	etcd.WatchWithRevNoPrevRetry(watchKey, rev, true, watchWLPwdQuit, wait, func(resp clientv3.WatchResponse) {
		for range resp.Events {
			updateWhiteListPwd()
		}
	})
}

func updateWhiteListPwd() (rev int64) {
	path := etcd.GetVersionWLPwdWatchUpdateKey(config.Cfg.CommonCfg.EtcdServer)
	kvs, rev, err := etcd.GetSubRecursiveWithRev(path)
	if err != nil {
		tilogs.L().Errorf("updateWhiteListPwd err %s", err.Error())
		return
	}
	infos := make(map[string]map[string]string, 16)
	for k, v := range kvs {
		k = strings.TrimPrefix(k, path)
		pg := strings.Split(k, "/") // gid/version
		gid := pg[0]
		version := pg[1]

		vers, ok := infos[gid]
		if !ok {
			vers = make(map[string]string, 4)
			infos[gid] = vers
		}

		infos[gid][version] = v
	}

	lock.Lock()
	WhiteListPwdInfos = infos
	lock.Unlock()
	tilogs.L().Infof("updateWhiteListPwd %v, revison: %v", infos, rev)
	return
}
