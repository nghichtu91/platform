package noticeimp

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/notice/config"

	"github.com/nghichtu91/platform/share/planx/arrayhelper"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/notice"
)

type NoticeInfo struct {
	Gid        string
	Version    string
	NoticeInfo notice.PublicRelease
}

func (nc *NoticeInfo) GetOuterNotice(gid, channelId, subChannelId, version, language, platform string) string {
	// PC包不需要提审, 所以也不需要判断渠道的提审状态
	if platform != consts.NoticePlatformWin {
		// 如果该渠道是提审状态， 返回新的notice地址
		if url, test := nc.getUrlIfChannelServer(channelId); test {
			return fmt.Sprintf(`{"URL":"%s"}`, url)
		}
	}
	return nc.buildOuterNotice(gid, channelId, subChannelId, version, language, platform)
}

// 返回公告跳转URL
func (nc *NoticeInfo) getUrlIfChannelServer(channelId string) (string, bool) {
	// 默认是过审状态
	if nc.NoticeInfo.ChannelServerInfo.ChannelsServer == nil {
		return "", false
	}
	info, ok := nc.NoticeInfo.ChannelServerInfo.ChannelsServer[channelId]
	if !ok {
		return "", false
	}
	if info.Status == notice.Channel_Pass {
		return "", false
	} else {
		return nc.NoticeInfo.VersionInfo.ChannelNoticeURL, true
	}
}

func (nc *NoticeInfo) buildOuterNotice(gid, channelId, subChannelId, version, language, platform string) string {
	publicNotice := &OuterNotice2Client{}
	nowTime := timeutil.Now().Unix()

	// 公告
	noticeList := make([]ReleaseNotice, 0)
	// hasForce := false // 是否存在强制更新的公告，如果有则为强更地址赋值，否则不赋值
	for _, releaseNotice := range nc.NoticeInfo.PublicInfo.Publics {
		// 时间筛选
		if nowTime >= releaseNotice.Begin && nowTime <= releaseNotice.End {
			// 渠道筛选
			// 国服需要用到渠道0，特判
			if (timeutil.IsAsiaShanghaiTZ() || channelId != "0") && !isChannelOk(releaseNotice.ChannelId, channelId) {
				// tilogs.L().Warnf("channel id invalid, %s", channelId)
				continue
			}
			if !isPlatformOk(releaseNotice.Platform, platform) {
				continue
			}
			noticeList = append(noticeList, FromNotice(releaseNotice, getShowOpenTime(releaseNotice.Kind, gid), language))

			if releaseNotice.Kind == notice.SystemNoticeKind_Forceupdate {
				// hasForce = true
			}

		} else {
			// tilogs.L().Warnf("notice time invalid, %v", nowTime)
		}
	}
	publicNotice.Publics = noticeList

	// 版本信息
	publicNotice.Version = nc.NoticeInfo.VersionInfo.Version
	publicNotice.Endpoint = nc.NoticeInfo.VersionInfo.Endpoint
	publicNotice.Bundle = buildBundleInfo(nc.NoticeInfo.VersionInfo.BundleAndroid, nc.NoticeInfo.VersionInfo.BundleIOS, nc.NoticeInfo.VersionInfo.BundlePC)
	publicNotice.PreHotfix = buildPrefixInfo(nc.NoticeInfo.VersionInfo.PreHotfixAndroid, nc.NoticeInfo.VersionInfo.PreHotfixIOS, nc.NoticeInfo.VersionInfo.PreHotfixPC)
	// 因为客户端的旧包解析bundle和preHotFix是用的正则，没办法从三个号拼接变为四个号拼接，因为要热更才能改代码，但数据变了没法解析热更号死锁了，所以这里新加一个字段(临时) 后面可能会删除
	// Todo by dubin 后面找客户端对这个怎么处理
	publicNotice.BundleNew = buildBundleInfoNew(nc.NoticeInfo.VersionInfo)
	publicNotice.PreHotfixNew = buildPrefixInfoNew(nc.NoticeInfo.VersionInfo)
	publicNotice.Whitelistpwd = nc.NoticeInfo.VersionInfo.Whitelistpwd
	publicNotice.BIOption = nc.NoticeInfo.VersionInfo.BIOption
	publicNotice.TokenTime = nc.NoticeInfo.VersionInfo.TokenTime
	publicNotice.TimeoutTime = nc.NoticeInfo.VersionInfo.TimeoutTime
	publicNotice.PrUrl = nc.NoticeInfo.VersionInfo.PrUrl
	publicNotice.PayUrls = nc.NoticeInfo.VersionInfo.PayUrls
	publicNotice.GZIPSize = nc.NoticeInfo.VersionInfo.GZIPSize
	publicNotice.ChatIp = nc.NoticeInfo.VersionInfo.ChatIp
	publicNotice.Time = timeutil.Now().Unix()
	publicNotice.ClientExpireTime = nc.NoticeInfo.VersionInfo.ClientExpireTime
	publicNotice.ABTestMode = nc.NoticeInfo.VersionInfo.ABTestMode
	_, offsetUTC := timeutil.Now().Zone()
	publicNotice.TimeZone = int64(offsetUTC / timeutil.HourSec)

	{
		// 登陆的客服开关
		err, state := buildCustomSwitchService(channelId, switchBuildParam{
			getValue:      config.GidCfg.GetTypeNoticeCustomerService,
			defaultSwitch: TypeNoticeCustomerService_Default_Open,
			errSwitch:     TypeNoticeCustomerService_Err,
		})
		if err != nil {
			return "{}"
		}

		publicNotice.NoticeCustomer = state
	}

	{
		// 登陆的Win端扫码开关
		err, state := buildCustomSwitchService(channelId, switchBuildParam{
			getValue:      config.GidCfg.GetTypeWinScanSwitchService,
			defaultSwitch: TypeWinScanSwitchService_Default_Close,
			errSwitch:     TypeWinScanSwitchService_Err,
		})
		if err != nil {
			return "{}"
		}
		publicNotice.WinScanSwitch = state
	}
	publicNotice.AndroidPrivacySwitch = config.GidCfg.GetAndroidPrivacy()
	publicNotice.IosPrivacySwitch = config.GidCfg.GetIosPrivacy()
	if planx.IsRunProd(config.RunMode) {
		publicNotice.ClientExpireTime = 0
	}

	//tilogs.L().Infof("ChatIp is : %s", nc.NoticeInfo.VersionInfo.ChatIp)
	tilogs.L().Debugf("publicNotice is : %+v", publicNotice)
	// 强更链接
	var updateURL string
	if platform == consts.NoticePlatformWin {
		updateURL = nc.NoticeInfo.VersionInfo.PCForceUrl
	} else if platform == consts.NoticePlatformMac {
		updateURL = nc.NoticeInfo.VersionInfo.MacForceUrl
	} else {
		updateURL = nc.NoticeInfo.ChannelServerInfo.GetForceUpdateUrl(channelId, subChannelId)
	}
	publicNotice.ForceUpdateURL = updateURL

	retJson, err := json.Marshal(publicNotice)
	if err != nil {
		tilogs.L().Errorf("marsha public notice error %v", err)
		return "{}"
	} else {
		return string(retJson)
	}
}

type switchBuildParam struct {
	getValue      func() string
	defaultSwitch int32
	errSwitch     int32
}

func buildCustomSwitchService(channelId string, param switchBuildParam) (error, int32) {
	if param.getValue == nil {
		return fmt.Errorf("invalid param"), param.errSwitch
	}
	ncsStr := param.getValue()
	sw := new(CustomerSwitches)
	if ncsStr == "" { // 使用默认值
		return nil, param.defaultSwitch
	}

	err := json.Unmarshal([]byte(ncsStr), sw)
	if err != nil {
		// 从旧版本升级，如果读取失败，直接返回新的
		tilogs.L().Errorf("failed to unmarshal from %s", ncsStr)
		return fmt.Errorf("failed to unmarshal from %s", ncsStr), param.errSwitch
	}

	for _, info := range sw.Switches {
		// 如果存在指定渠道的开关, 以渠道为准
		if strconv.Itoa(int(info.SwitchChannel)) == channelId {
			return nil, info.SwitchState
		}
	}

	// 否则就使用默认值
	return nil, param.defaultSwitch
}

func (nc *NoticeInfo) GetInnerNotice(sid, channelId, language, platform string) string {
	sidInt, err := strconv.Atoi(sid)
	if err != nil {
		tilogs.L().Errorf("sid error %v", err)
		return "{}"
	}
	nowTime := timeutil.Now().Unix()
	servers := make([]ReleaseNotice, 0)
	for _, innerNotice := range nc.NoticeInfo.PublicInfo.Inners {
		if nowTime >= innerNotice.Begin && nowTime <= innerNotice.End { // 时间范围内
			if isShardIdOk(innerNotice.Servers, sidInt) &&
				isChannelOk(innerNotice.ChannelId, channelId) &&
				isPlatformOk(innerNotice.Platform, platform) {
				servers = append(servers, FromNotice(innerNotice, 0, language))
			}
		}
	}
	retBytes, err := json.Marshal(InnerNotice2Client{PublicList: servers})
	if err != nil {
		tilogs.L().Errorf("marshal servers %v", err)
		return "{}"
	} else {
		return string(retBytes)
	}
}

func isPlatformOk(platform []string, p string) bool {
	return len(platform) == 0 || // platform没有All
		arrayhelper.ContainsString(platform, p)
}

func isChannelOk(chnannelList []string, ch string) bool {
	return len(chnannelList) == 0 ||
		arrayhelper.ContainsString(chnannelList, ch) ||
		arrayhelper.ContainsString(chnannelList, notice.ChannelId_All)
}

func isShardIdOk(shardList []int, sid int) bool {
	return len(shardList) == 0 ||
		arrayhelper.ContainsInt(shardList, sid) ||
		arrayhelper.ContainsInt(shardList, 0)
}

func isVersionOk(lowestClientVer string, version string) bool {
	return lowestClientVer == "" ||
		version >= lowestClientVer
}

// game_tools/game_tools/logic/game_version/add_game_version.go 中有相同的方法，修改需要一起修改
func buildBundleInfo(bundleAndroid int, bundleIOS int, bundlePC int) string {
	// Bundle 客户端需求，为0时候传空，即 ||
	bundleAndroidStr := ""
	bundleIOSStr := ""
	bundlePCStr := ""
	if bundleAndroid > 0 {
		bundleAndroidStr = strconv.Itoa(bundleAndroid)
	}

	if bundleIOS > 0 {
		bundleIOSStr = strconv.Itoa(bundleIOS)
	}

	if bundlePC > 0 {
		bundlePCStr = strconv.Itoa(bundlePC)
	}

	return fmt.Sprintf("%s|%s|%s", bundleAndroidStr, bundleIOSStr, bundlePCStr)
}

// 2022.4.14 客户端需求-增加下载号
func buildPrefixInfo(prefixAndriod int, prefixIOS int, prefixPC int) string {
	prefixAndriodStr := ""
	prefixIOSStr := ""
	prefixPCStr := ""
	if prefixAndriod > 0 {
		prefixAndriodStr = strconv.Itoa(prefixAndriod)
	}
	if prefixIOS > 0 {
		prefixIOSStr = strconv.Itoa(prefixIOS)
	}
	if prefixPC > 0 {
		prefixPCStr = strconv.Itoa(prefixPC)
	}

	return fmt.Sprintf("%s|%s|%s", prefixAndriodStr, prefixIOSStr, prefixPCStr)
}

// 加mac后的解析方法
func buildBundleInfoNew(version notice.VersionInfo) string {
	// Bundle 客户端需求，为0时候传空，即 ||
	bundleAndroidStr := ""
	bundleIOSStr := ""
	bundlePCStr := ""
	bundleMacStr := ""
	if version.BundleAndroid > 0 {
		bundleAndroidStr = strconv.Itoa(version.BundleAndroid)
	}

	if version.BundleIOS > 0 {
		bundleIOSStr = strconv.Itoa(version.BundleIOS)
	}

	if version.BundlePC > 0 {
		bundlePCStr = strconv.Itoa(version.BundlePC)
	}

	if version.BundleMac > 0 {
		bundleMacStr = strconv.Itoa(version.BundleMac)
	}

	return fmt.Sprintf("%s|%s|%s|%s", bundleAndroidStr, bundleIOSStr, bundlePCStr, bundleMacStr)
}

// 加mac后的解析方法
func buildPrefixInfoNew(version notice.VersionInfo) string {
	prefixAndriodStr := ""
	prefixIOSStr := ""
	prefixPCStr := ""
	prefixMacStr := ""
	if version.PreHotfixAndroid > 0 {
		prefixAndriodStr = strconv.Itoa(version.PreHotfixAndroid)
	}
	if version.PreHotfixIOS > 0 {
		prefixIOSStr = strconv.Itoa(version.PreHotfixIOS)
	}
	if version.PreHotfixPC > 0 {
		prefixPCStr = strconv.Itoa(version.PreHotfixPC)
	}
	if version.PreHotfixMac > 0 {
		prefixMacStr = strconv.Itoa(version.PreHotfixMac)
	}

	return fmt.Sprintf("%s|%s|%s|%s", prefixAndriodStr, prefixIOSStr, prefixPCStr, prefixMacStr)
}
