package notice

// 公告的种类
const (
	SystemNoticeKind_Public      = 0 //通常
	SystemNoticeKind_Maintaince  = 1 //维护
	SystemNoticeKind_Forceupdate = 2 //强制更新
	SystemNoticeKind_MiniGame    = 3 //迷你小游戏
	SystemNoticeKind_Stop        = 4 //阻挡
	SystemNoticeKind_JapanUpdate = 5 //日服更新预告
)

const (
	SystemNoticeType_None   = 0 //无
	SystemNoticeType_Verion = 1 //版本
	SystemNoticeType_Active = 2 //活动
	SystemNoticeType_Notice = 3 //公告
	SystemNoticeType_New    = 4 //最新
)

// 公告的状态
const (
	SystemNoticeStatus_OK      = 0 //生效中
	SystemNoticeStatus_Expired = 1 //已过期
	SystemNoticeStatus_Future  = 2 //将来生效
)

// 公告位置 废弃掉
// 只有Inner类型的才能选择服务器
const (
	SystemNoticeLocation_Both  = 0
	SystemNoticeLocation_Inner = 1
	SystemNoticeLocation_Outer = 2
)

// 渠道服（渠道服）信息
const (
	Channel_Pass    = 0 // 默认审核通过
	Channel_Testing = 1 // 提审中
)

const ChannelId_All = "-1"

type SystemNoticeInfo struct {
	ID             int      `json:"id"`
	Kind           int      `json:"kind"`
	Priority       int      `json:"priority"`
	GType          int      `json:"gtype"`
	StartTime      int      `json:"start_time"`
	EndTime        int      `json:"end_time"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	ReleaseVersion string   `json:"release_version"`
	ReleaseShardId int      `json:"release_shard_id"`
	Status         int      `json:"status"`
	LocationType   int      `json:"locationType"`
	CornerMark     int      `json:"cornerMark"`
	ChannelID      []string `json:"channelId"`
	ShowOpenTime   int64    `json:"show_open_time"`
	ChatIp         string   `json:"chatIp"` // 聊天服ip
}

// 发布公告
type ReleaseNotice struct {
	Index        string   `json:"Index"` // ID
	Title        string   `json:"Title"`
	Type         int      `json:"Type"`
	Priority     int      `json:"Priority"`
	SendTime     int64    `json:"SendTime"`
	Body         string   `json:"Body"`
	Begin        int64    `json:"Begin"`
	End          int64    `json:"End"`
	LocationType int      `json:"LocationType"`
	Servers      []int    `json:"Servers"`
	Kind         int      `json:"Kind"` // @SystemNoticeKind_Public
	ChannelId    []string `json:"ChannelId"`
	Platform     []string `json:"Platform"`
	CornerMark   int      `json:"cornerMark"` // 1:版本 2：公告
	TimeZone     int64    `json:"time_zone"`  // 时区 废弃
}

type NoticeInfo struct {
	Publics []ReleaseNotice `json:"Publics"`
	Inners  []ReleaseNotice `json:"Inners"`
}

type VersionInfo struct {
	Version          string            `json:"Version"`
	Endpoint         map[string]string `json:"Endpoint"`
	BundleAndroid    int               `json:"BundleAndroid"`
	BundleIOS        int               `json:"BundleIOS"`
	BundlePC         int               `json:"BundlePC"`
	BundleMac        int               `json:"BundleMac"`
	PreHotfixAndroid int               `json:"PreHotfixAndroid"`
	PreHotfixIOS     int               `json:"PreHotfixIOS"`
	PreHotfixPC      int               `json:"PreHotfixPC"`
	PreHotfixMac     int               `json:"PreHotfixMac"`
	PCForceUrl       string            `json:"PCForceUrl"`
	MacForceUrl      string            `json:"MacForceUrl"`
	Whitelistpwd     string            `json:"Whitelistpwd"`
	BIOption         string            `json:"BIOption"`
	TokenTime        int               `json:"TokenTime"`
	TimeoutTime      int               `json:"TimeoutTime"`
	PrUrl            string            `json:"PrUrl"`
	PayUrls          map[string]string `json:"PayUrls"`
	GZIPSize         int64             `json:"gzip_size"`
	ChannelNoticeURL string            `json:"test_notice_url"` // 提审服的notice url
	LowestClientVer  string            `json:"LowestClientVersion"`
	ForceUpdate      bool              `json:"force_update"`
	ChatIp           string            `json:"chatIp"`
	ClientExpireTime int64             `json:"clientExpireTime"`
	ABTestMode       int64             `json:"ABTestMode"`
}

/*
维护公告时间及白名单信息,用于notice服通知auth服
每个	Gid+Version对应一个
每个里可能有多个维护公告,所以时间有多组
*/
type MaintenanceInfo struct {
	TimeInfos []MaintenanceTimeInfo
}
type MaintenanceTimeInfo struct {
	StartTime int64
	EndTime   int64
	Channels  []string
}

// 发布的公告内容
type PublicRelease struct {
	PublicInfo        NoticeInfo     `json:"public_info"`         // 具体公告内容
	VersionInfo       VersionInfo    `json:"version_info"`        // 版本内容
	ChannelServerInfo ChannelsServer `json:"channel_server_info"` // 渠道服信息
}

// 提审服的信息
type ChannelServerInfo struct {
	Status            int               `json:"status"`               // 状态 0 审核通过， 1 未过审
	ForceUpdateUrl    string            `json:"force_update_url"`     // 主渠道强更地址
	AppForceUpdateUrl map[string]string `json:"app_force_update_url"` // 子渠道强更地址 {子渠道ID, url}
}

type ChannelsServer struct {
	ChannelsServer map[string]ChannelServerInfo // <channelId, >
}

// 三种情况
// 1.子渠道id为空，则主渠道强更地址
// 2.子渠道id不为空，且有对应的强更地址
// 3.子渠道id不为空，但没有对应的强更地址
func (ch *ChannelsServer) GetForceUpdateUrl(channelId, subChannelId string) string {
	if ch.ChannelsServer == nil {
		return ""
	}
	if chMap, ok := ch.ChannelsServer[channelId]; ok {
		if chMap.AppForceUpdateUrl == nil || chMap.AppForceUpdateUrl[subChannelId] == "" {
			return chMap.ForceUpdateUrl
		} else {
			return chMap.AppForceUpdateUrl[subChannelId]
		}
	} else {
		return ""
	}
}
