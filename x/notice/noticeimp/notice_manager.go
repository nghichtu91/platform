package noticeimp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/x/common/notice"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/notice/config"
)

const MaintenStatus = 1

type NoticeManager struct {
	// 保存原始公告
	lock    sync.RWMutex
	Notices map[string]*NoticeInfo // key = {gid:version}

	driver       Driver
	loadRevision int64 // 用于记录加载的版本
}

func (n *NoticeManager) Init(mysqlUrl string) {
	n.driver = NewDriver(mysqlUrl)
	tilogs.L().Infof("[START]begin load notice")
	// 在加载数据库数据之前, 先记录一下当前的版本
	{
		cfg := config.Cfg
		key := consts.GetNoticeWatchUpdateKey(cfg.EtcdServer, cfg.Gid)
		_, rev, err := etcd.GetWithRev(key)
		if err != nil {
			tilogs.L().Errorf("[START]get revision failed %v", err)
			rev = -1 // 如果获取失败, 则设置为-1, 从服务器最新的版本开始watch
		}
		n.loadRevision = rev
	}
	n.LoadAndUpdate(config.Cfg.Gid, "")
	jsonLog, err := json.Marshal(n.Notices)
	if err != nil {
		panic(fmt.Sprint("[START] start failed", err))
	}
	tilogs.L().Infof("[START]end load notice %v revision %v", string(jsonLog), n.loadRevision)
}

func (n *NoticeManager) LoadAndUpdate(gid uint, version string) {
	var memoryList []*NoticeInfo
	memoryList = n.driver.Get(gid, version)
	noticeManager.UpdateMemoryNotice(memoryList)
	n.UpdateNoticeTime(memoryList)
	n.UpdateClientExpireTime(memoryList)
}

// 记录公告维护的时间,若没有公告处于维护状态，则时间记录为0
type NoticeTime struct {
	startTime    int64
	endTime      int64
	whitelistpwd string
}

// 更新etcd的中公告维护时间，通知到auth
func (n *NoticeManager) UpdateNoticeTime(noticeInfo []*NoticeInfo) {
	if len(noticeInfo) <= 0 {
		return
	}

	for _, info := range noticeInfo {
		etcdValue := notice.MaintenanceInfo{
			TimeInfos: make([]notice.MaintenanceTimeInfo, 0, 1),
		}

		for _, noticeInfo := range info.NoticeInfo.PublicInfo.Publics {
			if noticeInfo.Kind != MaintenStatus {
				continue
			}

			etcdValue.TimeInfos = append(etcdValue.TimeInfos, notice.MaintenanceTimeInfo{
				StartTime: noticeInfo.Begin,
				EndTime:   noticeInfo.End,
				Channels:  noticeInfo.ChannelId,
			})
		}

		etcdKey := fmt.Sprintf("%s/%s/%s/%s", config.Cfg.EtcdServer, consts.KeyEndPoint, info.Gid, info.Version)
		jsonBytes, err := json.Marshal(etcdValue)
		if err != nil {
			tilogs.L().Errorf("etcd.Put maintenanceInfo marshal error %v", err)
			continue
		}
		err = etcd.Put(etcdKey, string(jsonBytes))
		if err != nil {
			tilogs.L().Errorf("etcd.Put maintenanceInfo error %v", err)
		}
	}
}

// 更新客户端版本过期时间到etcd
func (n *NoticeManager) UpdateClientExpireTime(noticeInfo []*NoticeInfo) {
	// 获取ETCD中老的版本信息
	bundleVerkey := etcd.GetVersionWatchUpdateKey(config.Cfg.EtcdServer, fmt.Sprint(config.Cfg.Gid)) + etcd.GameVersion_Bundle

	// 获取ETCD中旧的Ver版本信息
	bundleVerInfo, getErr := etcd.GetBundleVerInfo(bundleVerkey)
	if getErr != nil {
		tilogs.L().Errorf("<UpdateClientExpireTime> get bundleVerInfo err : %v", getErr)
		return
	}

	change := false
	for _, notice := range noticeInfo {
		if info, has := bundleVerInfo[notice.Version]; !has || info.ClientExpireTime != notice.NoticeInfo.VersionInfo.ClientExpireTime {
			change = true
			bundleVerInfo[notice.Version] = etcd.BundleInfo{
				BundleVer:        info.BundleVer,
				ForceUpdate:      info.ForceUpdate,
				ClientExpireTime: notice.NoticeInfo.VersionInfo.ClientExpireTime,
			}
		}
	}

	if change {
		etcd.PutBundleVerInfo(bundleVerkey, bundleVerInfo)
	}
}

func (n *NoticeManager) UpdateMemoryNotice(newNoticeInfo []*NoticeInfo) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.Notices == nil {
		n.Notices = make(map[string]*NoticeInfo)
	}
	for _, notice := range newNoticeInfo {
		key := fmt.Sprintf("%s:%s", notice.Gid, notice.Version)
		n.Notices[key] = notice
	}
}

func getShowOpenTime(kind int, gid string) int64 {
	if kind != notice.SystemNoticeKind_MiniGame {
		return 0
	}
	key := fmt.Sprintf("%s/%s/defaults/showOpenTime", config.Cfg.EtcdDevOps, gid)
	timeStr, err := etcd.Get(key)
	if err != nil {
		tilogs.L().Errorf("<getShowOpenTime> etcd get key err, key=%v, err=%v", key, err)
		return 0
	}
	time, err := strconv.Atoi(timeStr)
	if err != nil {
		tilogs.L().Errorf("<getShowOpenTime> atoi err=%v, timeStr=%v", err, timeStr)
		return 0
	}
	return int64(time)
}

func (n *NoticeManager) GetOuterNotice(gid, version, channelId, subChannelId, language, platform string) string {
	key := fmt.Sprintf("%s:%s", gid, version)
	n.lock.RLock()
	v, ok := n.Notices[key]
	n.lock.RUnlock()
	if ok {
		return v.GetOuterNotice(gid, channelId, subChannelId, version, language, platform)
	} else {
		tilogs.L().Debugf("[GetNotice][OuterNotice] %s not found", key)
		return "{}"
	}
}

func (n *NoticeManager) GetInnerNotice(gid, version, sid, channelId, language, platform string) string {
	key := fmt.Sprintf("%s:%s", gid, version)
	n.lock.RLock()
	v, ok := n.Notices[key]
	n.lock.RUnlock()
	if ok {
		return v.GetInnerNotice(sid, channelId, language, platform)
	} else {
		return ""
	}
}
