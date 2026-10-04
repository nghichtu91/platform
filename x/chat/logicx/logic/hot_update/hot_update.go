package hot_update

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nghichtu91/platform/share/x/chat/web/logic/chatSystem"

	clientv3 "go.etcd.io/etcd/client/v3"

	//"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/chat/logicx/config"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

var (
	quit chan struct{}
)

func init() {
	quit = make(chan struct{}, 1)
}

func StopConfigWatch() {
	close(quit)
}

func LoadDbConfig(wait *util.WaitGroupWrapper) error {
	config.EtcdConfig[chatSystem.Time] = make(map[string]int, 16)
	config.EtcdConfig[chatSystem.Common] = make(map[string]int, 16)
	// 监听大区下的聊天保存数目。
	watchKey := fmt.Sprintf("%s/%d/%s", config.Cfg.EtcdServer, config.Cfg.Gid, consts.KeyChatConfigPath)
	err := watchEtcd(watchKey, chatSystem.Common, wait)
	if err != nil {
		return err
	}
	// 监听大区下的聊天保存时间。
	watchKey = fmt.Sprintf("%s/%d/%s", config.Cfg.EtcdServer, config.Cfg.Gid, consts.KeyChatStoreTimePath)
	err = watchEtcd(watchKey, chatSystem.Time, wait)
	if err != nil {
		return err
	}
	return nil
}

func watchEtcd(watchKey string, infoType chatSystem.EtcdInfoType, wait *util.WaitGroupWrapper) error {
	// 监听大区下的聊天保存数目。
	rev, err := LoadEtcdConfig(watchKey, infoType)
	if err != nil {
		return err
	}
	etcd.WatchWithRevNoPrevRetry(watchKey, rev, true, quit, wait, func(resp clientv3.WatchResponse) {
		for _, event := range resp.Events {
			if event.Kv == nil {
				continue
			}
			if event.Type == clientv3.EventTypePut {
				updateValue(string(event.Kv.Key), string(event.Kv.Value), infoType)
			} else if event.Type == clientv3.EventTypeDelete {
				deleteValue(string(event.Kv.Key), infoType)
			}
		}
	})
	return nil
}

func LoadEtcdConfig(watchKey string, infoType chatSystem.EtcdInfoType) (int64, error) {
	keys, rev, err := etcd.GetSubKeyOnlyWithRev(watchKey)
	if err != nil {
		return rev, fmt.Errorf("logic load etcd config err:%v", err)
	}

	return rev, UpdateLocalConfig(keys, infoType)
}

func UpdateLocalConfig(keys []string, infoType chatSystem.EtcdInfoType) error {
	for _, key := range keys {
		value, err := etcd.Get(key)
		if err != nil {
			return fmt.Errorf("logic update local config,get etcd config failed, key:%s", key)
		}

		if err = updateValue(key, value, infoType); err != nil {
			return err
		}
	}

	return nil
}

func updateValue(key, value string, infoType chatSystem.EtcdInfoType) error {
	config.EtcdConfigMutex.Lock()
	defer config.EtcdConfigMutex.Unlock()

	index := strings.LastIndex(key, "/")
	if index == -1 || (index+1) >= len(key) {
		return fmt.Errorf("logic update key.LastIndex return error, key:%s", key)
	}
	keyType := key[index+1:]
	v, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("logic update local config, atoi failed, key:%s, value:%s", key, value)
	}
	config.EtcdConfig[infoType][keyType] = v
	//todo 如果是保存时间修改，是否需要立即设置对应的过期时间
	return nil
}

func deleteValue(key string, infoType chatSystem.EtcdInfoType) error {
	config.EtcdConfigMutex.Lock()
	defer config.EtcdConfigMutex.Unlock()

	index := strings.LastIndex(key, "/")
	if index == -1 || (index+1) >= len(key) {
		return fmt.Errorf("logic deleteValue key.LastIndex return error, key:%s", key)
	}
	keyType := key[index+1:]
	delete(config.EtcdConfig[infoType], keyType)
	return nil
}
