package logic

import (
	"github.com/nghichtu91/platform/share/planx/tilogs/logiclog"
	"github.com/nghichtu91/platform/share/x/auth/config"
)

type loginInfo struct {
	Device  string
	Team    string
	IsReg   int // 首次为1，非首次为0
	CfgTeam string
}

func (lai *loginInfo) GetProperties() map[string]interface{} {
	ret := make(map[string]interface{}, 4)
	ret["Device"] = lai.Device
	ret["Team"] = lai.Team

	if lai.IsReg > 0 {
		ret["IsReg"] = 1
	} else {
		ret["IsReg"] = 0
	}

	ret["CfgTeam"] = lai.CfgTeam
	return ret
}

func (lai *loginInfo) NeedTA() bool {
	return false
}

type createInfo struct {
	IP string
}

func (c *createInfo) GetProperties() map[string]interface{} {
	return nil
}

func (c *createInfo) NeedTA() bool {
	return false
}

func LogCreate(uid, DeviceID, channelId, ip string) {
	logiclog.LogGameLogicError("createInfo", &logiclog.Common{
		Gid:      config.Cfg.CommonCfg.Gid,
		Channel:  channelId,
		Uid:      uid,
		DeviceId: DeviceID,
	}, &createInfo{
		IP: ip,
	}, "")
}

func LogLogin(uid, DeviceID, channelId, device string, isReg bool) {
	r := &loginInfo{
		Device: device,
	}
	if isReg {
		r.IsReg = 1
	}
	logiclog.LogGameLogicError("Login_Auth", &logiclog.Common{
		Gid:      config.Cfg.CommonCfg.Gid,
		Uid:      uid,
		Channel:  channelId,
		DeviceId: DeviceID,
	}, r, "")
}

func LogLoginTeam(uid, team, channelId, cfgTeam string, sharid uint) {
	r := &loginInfo{
		Team:    team,
		CfgTeam: cfgTeam,
	}
	logiclog.LogGameLogicError("Login_Auth", &logiclog.Common{
		Gid:     config.Cfg.CommonCfg.Gid,
		Sid:     uint(sharid),
		Channel: channelId,
		Uid:     uid,
	}, r, "")
}

type UserLogin struct {
	Appkey        string //英雄上传的gameID
	ClientOs      string //客户端类型
	NTSDKJson     string //客户端传来的NTSDK信息
	Platform      string //平台
	ServerId      string //区服表示ID，自定义
	BaseChannelId string //渠道号
	AppChannelId  string //次级渠道号
	DeviceId1     string //设备号
	UserId        string //账号ID
	OpenId        string //渠道账号ID
	TransactionId string //事件关联ID
	PlayerId      int64
	AccountID     string
	IP            string
	AdsJson       string
	UserBalance   string
}

func (c *UserLogin) GetProperties() map[string]interface{} {
	return nil
}

func (c *UserLogin) NeedTA() bool {
	return false
}
