package noticeimp

import (
	"encoding/json"
	"fmt"

	"github.com/nghichtu91/platform/share/planx/signalhandler"

	"github.com/nghichtu91/platform/share/planx/multilingual"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/common/consts"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	CConfig "github.com/nghichtu91/platform/share/x/common/config"
	"github.com/nghichtu91/platform/share/x/common/notice"
	"github.com/nghichtu91/platform/share/x/notice/config"
)

type ReleaseNotice struct {
	Index        string `json:"Index"` // ID
	Title        string `json:"Title"`
	Type         int    `json:"Type"`
	Priority     int    `json:"Priority"`
	SendTime     int64  `json:"SendTime"`
	Body         string `json:"Body"`
	Begin        int64  `json:"Begin"`
	End          int64  `json:"End"`
	LocationType int    `json:"LocationType"`
	Kind         int    `json:"Kind"`
	ShowOpenTime int64  `json:"ShowOpenTime"`
	CornerMark   int    `json:"CornerMark"`
	TimeZone     int64  `json:"time_zone"` // 废弃，客户端改用OuterNotice2Client下面的TimeZone字段
}

type NoticeBody struct {
	Title   string `json:"Title"`
	Content string `json:"Content"`
}

func FromNotice(notice notice.ReleaseNotice, showOpenTime int64, language string) ReleaseNotice {
	bodyMul := &NoticeBody{
		Title:   multilingual.UnMarshalByLanguage(language, notice.Title),
		Content: multilingual.UnMarshalByLanguage(language, notice.Body),
	}
	body, _ := json.Marshal(bodyMul)

	return ReleaseNotice{
		Index:        notice.Index,
		Title:        bodyMul.Title,
		Type:         notice.Type,
		Priority:     notice.Priority,
		SendTime:     notice.SendTime,
		Body:         string(body),
		Begin:        notice.Begin,
		End:          notice.End,
		LocationType: notice.LocationType,
		Kind:         notice.Kind,
		ShowOpenTime: showOpenTime,
		CornerMark:   notice.CornerMark,
		TimeZone:     notice.TimeZone,
	}
}

type OuterNotice2Client struct {
	Publics              []ReleaseNotice   `json:"Publics"`
	Version              string            `json:"Version"`
	Endpoint             map[string]string `json:"Endpoint"`
	Bundle               string            `json:"Bundle"`
	PreHotfix            string            `json:"PreHotfix"`
	Whitelistpwd         string            `json:"Whitelistpwd"`
	BIOption             string            `json:"BIOption"`
	TokenTime            int               `json:"TokenTime"`
	TimeoutTime          int               `json:"TimeoutTime"`
	PrUrl                string            `json:"PrUrl"`
	PayUrls              map[string]string `json:"PayUrls"`
	GZIPSize             int64             `json:"gzip_size"`
	ForceUpdateURL       string            `json:"ForceUpdateURL"`
	ChatIp               string            `json:"chatIp"`
	Time                 int64             `json:"ServerTime"`
	ClientExpireTime     int64             `json:"clientExpireTime"`
	ABTestMode           int64             `json:"ABTestMode"`
	TimeZone             int64             `json:"time_zone"`      // 时区
	NoticeCustomer       int32             `json:"NoticeCustomer"` // 公告客服开关
	WinScanSwitch        int32             `json:"WinScanSwitch"`
	AndroidPrivacySwitch string            `json:"AndroidPrivacySwitch"`
	IosPrivacySwitch     string            `json:"IosPrivacySwitch"`
	BundleNew            string            `json:"BundleNew"`
	PreHotfixNew         string            `json:"PreHotfixNew"`
}

type CustomerSwitches struct {
	Switches []SwitchInfo `json:"switches,omitempty"`
}

type SwitchInfo struct {
	// 开关类型，和Debug枚举以及玩法枚举一致
	SwitchType int32 ` json:"SwitchType,omitempty"`
	// 开关状态，见 enum.proto SwitchState
	SwitchState int32 `json:"SwitchState,omitempty"`
	// 渠道
	SwitchChannel int32 `json:"SwitchChannel,omitempty"`
}

const (
	TypeNoticeCustomerService_Default_Open = 0
	TypeNoticeCustomerService_Err          = -1

	TypeWinScanSwitchService_Default_Close = 2
	TypeWinScanSwitchService_Err           = -1
)

type InnerNotice2Client struct {
	PublicList []ReleaseNotice `json:"Publics"`
}

var (
	noticeManager = NoticeManager{}
	quitChan      = make(chan struct{}, 1)
)

func InitNotice(w *util.WaitGroupWrapper) (success bool) {
	cfg := config.Cfg
	gidCfg := CConfig.InitLoadGidConfig(cfg.EtcdDevOps, cfg.Gid)
	if gidCfg == nil {
		return false
	}
	config.GidCfg = gidCfg
	if w != nil {
		config.GidCfg.StartWatch(w, cfg.EtcdDevOps, cfg.Gid)
		signalhandler.SignalKillFunc(func() { config.GidCfg.StopConfigBattleCheck() })
	}

	noticeManager.Init(config.GidCfg.MysqlUrl)
	startWatch(w, noticeManager.loadRevision)
	return true
}

func CloseNotice() {
	close(quitChan)
}

// 监听etcd变化，更新公告内容
// 采用监听的方式（取代URL通知）是为了集群部署
func startWatch(w *util.WaitGroupWrapper, rev int64) {
	cfg := config.Cfg
	watchKey := consts.GetNoticeWatchUpdateKey(cfg.EtcdServer, cfg.Gid)

	etcd.WatchWithRevNoPrevRetry(watchKey, rev, false, quitChan, w, func(resp clientv3.WatchResponse) {
		for _, event := range resp.Events {
			if event.Type == clientv3.EventTypePut {
				versions := getUpdateVersion(event.Kv.Value)
				tilogs.L().Infof("receive notice update gid=%d, version=%v", cfg.Gid, versions)
				for _, v := range versions {
					onReceiveUpdateNotify(cfg.Gid, v)
				}
			}
		}
	})
}

func getUpdateVersion(value []byte) []string {
	if len(value) == 0 {
		return nil
	}
	version := make([]string, 0)
	err := json.Unmarshal(value, &version)
	if err != nil {
		tilogs.L().Errorf("unmarshal update version error %v", err)
		return nil
	}
	return version
}

func onReceiveUpdateNotify(gid uint, version string) {
	tilogs.L().Infof("[UPDATE]begin update notice")
	noticeManager.LoadAndUpdate(gid, version)
	jsonLog, err := json.Marshal(noticeManager.Notices)
	if err != nil {
		tilogs.L().Errorf(fmt.Sprint("[UPDATE] update notice failed", err))
		return
	}
	tilogs.L().Infof("[UPDATE]end load notice %v", string(jsonLog))
}

func GetOuterNotice(gid, version string, channelId string, subChannelId, language, platform string) string {
	return noticeManager.GetOuterNotice(gid, version, channelId, subChannelId, language, platform)
}

func GetInnerNotice(gid, version, sid string, channelId, language, platform string) string {
	return noticeManager.GetInnerNotice(gid, version, sid, channelId, language, platform)
}
