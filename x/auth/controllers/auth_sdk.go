package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nghichtu91/platform/share/planx/tilogs/logiclog"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/x/auth/hero_log/common_log"

	"github.com/nghichtu91/platform/share/planx/ginhelper"

	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/nghichtu91/platform/share/planx/etcd"

	uuid "github.com/satori/go.uuid"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/gin-gonic/gin"

	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/auth/errorctl"
	"github.com/nghichtu91/platform/share/x/auth/hero_log/logic_log"
	"github.com/nghichtu91/platform/share/x/auth/logic"
	"github.com/nghichtu91/platform/share/x/auth/models"
)

// AuthController about object
type AuthController struct {
	ginhelper.GinController
}

/*
SdkInterface 定义了一组SDK行为
*/
type SdkInterface interface {
	/*
		VerifyLogin 用来验证登录， 需要返回必要的用户信息
	*/
	VerifyLogin(loginInfo *SdkLoginInfo) (SdkUserInterface, error)

	/*
		VerifyLogin is used for verifying pay
	*/
	VerifyPay()

	/*
		flag of sdk, for example: hero.com
	*/
	Flag() string
}

/*
SdkUserInterface 定义了一组SDK用户的必要信息
*/
type SdkUserInterface interface {
	UserId() string
	IsValid() bool
}

type SdkLoginInfo struct {
	Token         string `form:"token"`
	ChannelId     string `form:"channelId"`
	AppChannelId  string `form:"app_channelId"`
	Device        string `form:"device"`
	Uid           string `form:"uid"`
	PlatformType  string `form:"typ"` // ANDROID || IOS
	DeviceId      string `form:"device_Id"`
	IP            string `form:"ip"`
	AddressFlag   string `form:"address_flag"`
	SdkChannelUid string `form:"sdkChannelUID"`
	BdcCommonInfo string `form:"bdc_common_info"`
	SdkDeviceid   string `form:"sdk_deviceid"`
	Environment   string `form:"environment"`

	tokenDecoded         string
	channelIdDecoded     string
	appChannelIdDecoded  string
	deviceDecoded        string
	platformTypeDecoded  string
	uidDecoded           string
	deviceIdDecoded      string
	ipDecoded            string
	addressFlagDecoded   string
	sdkChannelUidDecoded string
	sdkDeviceIdDecoded   string
	environmentDecoded   string
}

func (s *SdkLoginInfo) Decode() error {
	tokenBytes, err := secure.DefaultEncode.Decode64FromNet(s.Token)
	if err != nil {
		return err
	}
	s.tokenDecoded = string(tokenBytes)

	channelBytes, err := secure.DefaultEncode.Decode64FromNet(s.ChannelId)
	if err != nil {
		return err
	}
	s.channelIdDecoded = string(channelBytes)

	if s.Device != "" {
		deviceBytes, err := secure.DefaultEncode.Decode64FromNet(s.Device)
		if err != nil {
			return err
		}
		s.deviceDecoded = string(deviceBytes)
	}

	platformBytes, err := secure.DefaultEncode.Decode64FromNet(s.PlatformType)
	if err != nil {
		return err
	}
	s.platformTypeDecoded = string(platformBytes)

	uidBytes, err := secure.DefaultEncode.Decode64FromNet(s.Uid)
	if err != nil {
		return err
	}
	s.uidDecoded = string(uidBytes)

	deviceIdBytes, err := secure.DefaultEncode.Decode64FromNet(s.DeviceId)
	if err != nil {
		return err
	}
	s.deviceIdDecoded = string(deviceIdBytes)

	appChannelIdBytes, err := secure.DefaultEncode.Decode64FromNet(s.AppChannelId)
	if err != nil {
		return err
	}
	s.appChannelIdDecoded = string(appChannelIdBytes)

	ipIdBytes, err := secure.DefaultEncode.Decode64FromNet(s.IP)
	if err != nil {
		return err
	}
	s.ipDecoded = string(ipIdBytes)

	add, err := secure.DefaultEncode.Decode64FromNet(s.AddressFlag)
	if err != nil {
		return err
	}
	s.addressFlagDecoded = string(add)

	sdkChannelUid, err := secure.DefaultEncode.Decode64FromNet(s.SdkChannelUid)
	if err != nil {
		return err
	}
	s.sdkChannelUidDecoded = string(sdkChannelUid)

	bdcCommonInfo, err := secure.DefaultEncode.Decode64FromNet(s.BdcCommonInfo)
	if err != nil {
		return err
	}
	s.BdcCommonInfo = string(bdcCommonInfo)

	sdkDeviceid, err := secure.DefaultEncode.Decode64FromNet(s.SdkDeviceid)
	if err != nil {
		return err
	}
	s.sdkDeviceIdDecoded = string(sdkDeviceid)

	environment, err := secure.DefaultEncode.Decode64FromNet(s.Environment)
	if err != nil {
		return err
	}
	s.environmentDecoded = string(environment)
	return nil
}

func (s *SdkLoginInfo) Check() bool {
	return s.Token != "" && s.ChannelId != ""
}

type SdkLoginRet struct {
	Result       string                `json:"result"`
	AuthToken    string                `json:"authtoken,omitempty"`
	DisplayName  string                `json:"display,omitempty"`
	ShardHasRole []models.ShardHasRole `json:"shardrole"`
	LastShard    string                `json:"lastshard"`
	GagTime      int64                 `json:"gag_time"` // 禁言截止时间 0为不禁言
	NewReg       bool                  `json:"new_reg"`  // 是否新注册的账号
	Err          string                `json:"error,omitempty"`
}

func (dic *AuthController) LoginWithSdk() gin.HandlerFunc {
	return func(c *gin.Context) {
		tilogs.L().Debugf("rev LoginWithSdk")
		loginInfo := &SdkLoginInfo{}
		if err := c.Bind(loginInfo); err != nil {
			errorctl.CtrlErrorReturn(c, "[Auth.SDK.LOGIN]", err,
				errorctl.FailForClient, errorctl.ClientErrorFormatVerifyFailed)
			return
		}

		if !loginInfo.Check() {
			// 无法校验的请求一概放弃
			errorctl.CtrlErrorReturn(c, "[Auth.SDK.LOGIN]",
				fmt.Errorf("token and channel should be required"),
				errorctl.FailForClient, errorctl.ClientErrorFormatVerifyFailed,
			)
			return
		}

		if err := loginInfo.Decode(); err != nil {
			errorctl.CtrlErrorReturn(c, "[Auth.SDK.LOGIN]",
				err, errorctl.FailForClient, errorctl.ClientErrorFormatVerifyFailed,
			)
			return
		}
		tilogs.L().Debugf("got login info %+v", loginInfo)

		sdk := newSdk(loginInfo)
		if sdk == nil {
			errorctl.CtrlErrorReturn(c, "[Auth.SDK.LOGIN]",
				fmt.Errorf("bad channelId:%s", loginInfo.channelIdDecoded),
				errorctl.FailForServer, errorctl.ClientErrorFormatVerifyFailed,
			)
			return
		}

		userInfo, err := sdk.VerifyLogin(loginInfo)
		if err != nil {
			errorctl.CtrlErrorReturn(c, "[Auth.SDK.LOGIN]",
				err, errorctl.FailForClient, errorctl.ClientErrorHeroSdkGetUser,
			)
			return
		}

		tilogs.L().Debugf("[login]verify login, got user info %v", userInfo)
		if userInfo == nil || !userInfo.IsValid() {
			errorctl.CtrlErrorReturn(c, "[Auth.SDK.LOGIN]",
				fmt.Errorf("get user error"), errorctl.FailForServer, errorctl.ClientErrorQuickSdkCheckUser,
			)
			return
		}

		res, authToken, display, uid, gagTime, newReg := login(c, sdk.Flag(), userInfo.UserId(),
			loginInfo.channelIdDecoded, loginInfo.deviceDecoded, loginInfo.sdkChannelUidDecoded, loginInfo)
		if !res {
			// login 函数会填充失败原因
			tilogs.L().Debugf("login failed with result false")
			return
		}

		ret := &SdkLoginRet{}
		ret.ShardHasRole, ret.LastShard = _getUserShardInfo(uid)
		ret.Result = "ok"
		ret.AuthToken = authToken
		ret.DisplayName = display
		ret.GagTime = gagTime
		ret.NewReg = newReg
		tilogs.L().Debugf("sdk login ret %+v", loginInfo)
		c.JSON(200, ret)
	}
}

// sdkChannelUid 参数暂时仅用于英雄bdc日志里的 userId 记录
func login(c *gin.Context, sdkFlag, sdkUid, channelId, device, sdkChannelUid string, sdkLoginInfo *SdkLoginInfo) (
	res bool,
	authToken string,
	display,
	uidstr string,
	gagTime int64,
	newReg bool) {

	channelId = abroadChannelAccountCross(channelId)
	sdkDeviceID := fmt.Sprintf("%s@%s@%s", sdkUid, channelId, sdkFlag)
	di, err, isGM := models.TryToCheckDeviceInfo(sdkDeviceID)
	newReg = false
	if err == models.XErrUserNotExist {
		// 可以创建新帐号
		di, err = models.AddDevice(sdkDeviceID, "", channelId, device)
		uid := db.InvalidUserID
		if di != nil {
			uid = di.UserId
		}
		if err != nil {
			errorctl.CtrlErrorReturn(c, "[Auth.Device]",
				fmt.Errorf("AddDevice error: uid(%d), error(%s)", uid, err),
				errorctl.FailForServer, errorctl.ClientErrorMaybeDBProblem,
			)
			return false, "", "", "", 0, newReg
		}
		newReg = true
		// bi
		logic.LogCreate(uid.String(), device, channelId, "")
	} else if err != nil {
		errorctl.CtrlErrorReturn(c, "[Auth.Device]",
			err,
			errorctl.FailForServer, errorctl.ClientErrorMaybeDBProblem,
		)
		return false, "", "", "", 0, newReg
	}

	uid := di.UserId

	// bi
	logic.LogLogin(uid.String(), sdkDeviceID, channelId, device, newReg)

	var clientOs string
	if sdkLoginInfo.platformTypeDecoded == ANDROID {
		clientOs = "ANDROID"
	} else {
		clientOs = "IOS"
	}

	bdcCommonInfo := common_log.LogCommonUploadInfo{}
	if sdkLoginInfo.BdcCommonInfo != "" {
		if err := json.Unmarshal([]byte(sdkLoginInfo.BdcCommonInfo), &bdcCommonInfo); err != nil {
			tilogs.L().Errorf("json.Unmarshal sdkLoginInfo.BDCCommonInfo %s err %s", sdkLoginInfo.BdcCommonInfo, err.Error())
		}
	}

	logic_log.LogUserLoginBdcLog(common_log.LogPlayerInfo{
		ClientOs:         clientOs,
		BaseChannelId:    sdkLoginInfo.channelIdDecoded,
		BaseAppChannelId: sdkLoginInfo.appChannelIdDecoded,
		BaseDeviceId:     sdkLoginInfo.sdkDeviceIdDecoded,
		UserId:           sdkChannelUid,
		OpenId:           sdkChannelUid,
		RoleId:           "",
		RoleKey:          "",
		UserIp:           sdkLoginInfo.ipDecoded,
		TransactionId:    "0",
		BdcDeviceId:      bdcCommonInfo.BdcDeviceId,
		DeviceIdType:     bdcCommonInfo.DeviceIdType,
		DeviceKey:        bdcCommonInfo.DeviceKey,
		DeviceModel:      bdcCommonInfo.DeviceModel,
		IME:              bdcCommonInfo.IME,
		GAid:             bdcCommonInfo.GAid,
		IdFa:             bdcCommonInfo.IdFa,
		IdFv:             bdcCommonInfo.IdFv,
		AndroidId:        bdcCommonInfo.AndroidId,
		OaId:             bdcCommonInfo.OaId,
		SdkVersion:       bdcCommonInfo.SdkVersion,
		BdcClientOs:      bdcCommonInfo.BdcClientOs,
		OldDeviceId:      bdcCommonInfo.OldDeviceId,
	}, sdkLoginInfo.addressFlagDecoded)
	logiclog.LogGameLogicError("UserLogin", &logiclog.Common{
		Gid:     config.Cfg.CommonCfg.Gid,
		Channel: channelId,
		Uid:     uid.String(),
	}, &logic.UserLogin{
		Appkey:        common_log.Cfg.AppKey,
		ClientOs:      clientOs,
		NTSDKJson:     sdkLoginInfo.BdcCommonInfo,
		Platform:      common_log.Cfg.Platform,
		ServerId:      common_log.Cfg.ServerId,
		BaseChannelId: sdkLoginInfo.channelIdDecoded,
		AppChannelId:  sdkLoginInfo.appChannelIdDecoded,
		DeviceId1:     sdkLoginInfo.sdkDeviceIdDecoded,
		UserId:        sdkChannelUid,
		OpenId:        sdkChannelUid,
		TransactionId: "0",
		PlayerId:      0,
		AccountID:     uid.String(),
		IP:            sdkLoginInfo.ipDecoded,
		AdsJson:       sdkLoginInfo.addressFlagDecoded,
		UserBalance:   "{}",
	}, "")

	if !isGM {
		err, ban_time, ban_reason, gag_time := models.UserBanInfo(uid)
		if err != nil {
			errorctl.CtrlBanReturn(c, "[Auth.Device]",
				err, errorctl.FailForServer,
				errorctl.ClientErrorBanByGM, ban_time, ban_reason)
			return false, "", "", "", 0, newReg
		}
		// 下面逻辑err == nil
		authToken, err = models.AuthToken(context.Background(), uid, sdkDeviceID)
		gagTime = gag_time
		if err != nil {
			errorctl.CtrlErrorReturn(c, "[Auth.Device]",
				err,
				errorctl.FailForServer, errorctl.ClientErrorMaybeDBProblem,
			)
			return false, "", "", "", 0, newReg
		}
	} else {
		authToken = uuid.NewV4().String()
	}

	// Auth token 发送到登录校验服务器，并设置有效时间
	notifyLoginServer(context.Background(), authToken, uid, sdkDeviceID)
	return true, authToken, di.Display, uid.String(), gagTime, newReg
}

const (
	ANDROID = "0"
)

func _getUserShardInfo(uid string) (userShard []models.ShardHasRole, lastShard string) {
	_userShard, err := models.GetUserShardHasRole(context.Background(), uid)
	if err != nil {
		tilogs.L().Warnf("[quick] login GetUserShard err: %v", err)
	} else {
		userShard = _userShard
	}

	_lastShard, err := models.GetUserLastShard(context.Background(), uid)
	if err != nil {
		tilogs.L().Warnf("[quick] login GetUserLastShard err: %v", err)
	} else {
		lastShard = _lastShard
	}
	return
}

func GetServerTeam(accountId []byte, gid uint, sid uint) (string, string) {
	teamAB_parent := fmt.Sprintf("%s/%d/%d/%s", config.Cfg.CommonCfg.EtcdServer, gid, sid, consts.KeyTeamAB)
	teamAB, err := etcd.Get(teamAB_parent)
	if err != nil {
		return "a", teamAB
	}
	switch teamAB {
	case "A":
		return "a", teamAB
	case "B":
		return "b", teamAB
	case "AB":
		lastLetterByte := accountId[len(accountId)-1]
		if isOdd(int(lastLetterByte)) {
			return "a", teamAB
		} else {
			return "b", teamAB
		}
	}
	tilogs.L().Debugf("teamAB_parent %s team %s", teamAB_parent, "team is wrong")
	return "", ""

}
func isOdd(num int) bool {
	return num%2 != 0
}

func newSdk(loginInfo *SdkLoginInfo) SdkInterface {
	if loginInfo.channelIdDecoded == consts.RobotChannel {
		return &robotSdk{flag: consts.RobotSuffix[1:]}
	} else if strings.Contains(loginInfo.channelIdDecoded, "@"+consts.RobotChannel) { // 内部QA测试号，channelId参数：channelId+@+robot
		loginInfo.channelIdDecoded = strings.TrimSuffix(loginInfo.channelIdDecoded, "@"+consts.RobotChannel)
		return &robotSdk{flag: "taihe.com"}
	} else {
		return &NtSdk{}
	}
}

func abroadChannelAccountCross(channel string) string {
	if timeutil.IsAsiaShanghaiTZ() { // 国内
		// 由于国服二测时没有支持账号互通，并且国服二测还有ios账号，公测想要支持账号互通并且充值返利也对，所以公测ios也用Android的渠道号生成deviceid
		if channel == consts.CNHeroAndroid || channel == consts.CNHeroIOS {
			return consts.CNHeroAndroid
		}
	} else if timeutil.IsVN() {
		return "crossvn"
	} else { // 海外
		if channel == consts.AbroadOfficeWebChannel || channel == consts.AbroadGoogleChannel ||
			channel == consts.AbroadIOSChannel || channel == consts.AbroadTapTapChannel ||
			channel == consts.AbroadPCChannel || channel == consts.AbroadMacChannel {
			return "cross"
		}
	}
	return channel
}
