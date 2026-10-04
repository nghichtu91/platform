package msg

import (
	"github.com/nghichtu91/platform/share/x/common/consts"
)

type KickOfflineParam struct {
	LoginToken      string
	AccountID       string
	Reason          string
	KickErrCode     int
	AfterDuration   int
	NoLoginDuration int
}

type ClientHandShakeInfo struct {
	LoginToken               string `json:"-"` // 不进入json
	GZipLimitS               string `json:"-"` // 不进入json
	ChannelId                string `json:"ch_id"`
	BDCId                    string `json:"bdc_id"` // 废弃
	Version                  string `json:"ver"`
	DeviceType               string `json:"device_type"`
	DeviceSystem             string `json:"device_system"`
	UpdateVer                string `json:"update_ver"`
	IP                       string `json:"ip"`
	IDFA                     string `json:"idfa"`
	DeviceMemory             string `json:"device_memory"`
	AddressFlag              string `json:"address_flag"` // 联运渠道广告标识
	DeviceId                 string `json:"dev_id"`
	AppChannelId             string `json:"app_channel_id"` // 客户端命名：projectID
	SdkChannelUid            string `json:"sdk_channel_uid"`
	BdcCommonInfo            string `json:"bdc_common_info"` // 客户端命名：ServerBean
	Language                 string `json:"language"`
	ClientBuildTimeAndHotVer string `json:"clientBuildTimeAndHotVer"` // 客户端命名：buildInfo
	BDCDeviceId              string `json:"bdc_device_id"`
	SubChannelID             string `json:"sub_channel_id"`  // 子渠道号，仅国服bdc使用，海外会传与channelID一样的数字
	ClientPlatform           string `json:"client_platform"` // 客户端平台
	DistinctID               string `json:"distinct_id"`     // 海外为了打通Appfly，使用中台生成的唯一ID替换deviceID
	ClientType               string `json:"-"`               // 客户端类型，用于区分是正式服还是测试服，不进入json

	SdkId              string `json:"sdk_dev_id"`
	AccountNumberId    int64  `json:"ac_number_id"`
	RechargeMoney      int32  `json:"recharge_money"`     // 充值返利-累充积分
	RechargeReturn     bool   `json:"recharge_return"`    // 充值返利-是否已领取够累充返利的奖励
	LoginDay           int32  `json:"loginDay"`           // 登陆排名返利-登陆天数。
	Fair1v1Dan         int32  `json:"fair1v1Dan"`         // 登陆排名返利-1v1段位。
	Fair3v3Dan         int32  `json:"fair3v3Dan"`         // 登陆排名返利-3v3段位。
	LoginAndRankReward bool   `json:"loginAndRankReward"` // 登陆排名返利-是否已领。
	ClaimTime          int64  `json:"claimTime"`          // 登陆排名返利-领取时间。
	IsSuper            bool   `json:"isSuper"`            // 是否是超级用户（白名单）
}

func RobotClientHandshakeInfo(token, limit string) ClientHandShakeInfo {
	return ClientHandShakeInfo{
		LoginToken:               token,
		GZipLimitS:               limit,
		ChannelId:                consts.RobotChannel,
		BDCId:                    consts.RobotChannel,
		Version:                  consts.RobotChannel,
		DeviceType:               consts.RobotChannel,
		DeviceSystem:             consts.RobotChannel,
		UpdateVer:                consts.RobotChannel,
		IP:                       consts.RobotChannel,
		IDFA:                     consts.RobotChannel,
		DeviceMemory:             consts.RobotChannel,
		AddressFlag:              consts.RobotChannel,
		DeviceId:                 consts.RobotChannel,
		AppChannelId:             consts.RobotChannel,
		SdkChannelUid:            consts.RobotChannel,
		BdcCommonInfo:            consts.RobotChannel,
		Language:                 consts.RobotChannel,
		ClientBuildTimeAndHotVer: consts.RobotChannel,
		BDCDeviceId:              consts.RobotChannel,
		SubChannelID:             consts.RobotChannel,
		ClientPlatform:           consts.RobotChannel,
		DistinctID:               consts.RobotChannel,
		ClientType:               consts.ClientTypeDebug,
	}
}

// HandShakeResp 向客户端发送的握手回复
type HandShakeResp struct {
	Result   string // "fail", "ok", "no"
	EncAcID  string // 编码后的acid
	SrvTS    string // 服务器时间
	Ver      string // 版本信息
	EncNumID string // 编码后的数字ID

	AccountID string
	NumID     int
}
