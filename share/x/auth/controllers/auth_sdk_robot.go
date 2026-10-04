package controllers

import (
	"errors"

	"github.com/nghichtu91/platform/share/x/common/consts"
)

/*
	用于检查线上服务可用性的相关功能
*/

var (
	ErrInvalidRobotReq = errors.New("invalid robot request")
)

type robotSdk struct {
	flag string // 机器人可以修改标记
}

func (sdk *robotSdk) VerifyLogin(loginInfo *SdkLoginInfo) (SdkUserInterface, error) {
	// 因为这里接口对外，为了安全起见还是验证一下
	if loginInfo.tokenDecoded != loginInfo.uidDecoded+consts.RobotSuffix {
		return nil, ErrInvalidRobotReq
	}

	return &robotUser{uid: loginInfo.uidDecoded}, nil
}

func (sdk *robotSdk) VerifyPay() {}

func (sdk *robotSdk) Flag() string {
	return sdk.flag
}

type robotUser struct {
	uid string
}

func (ru *robotUser) UserId() string {
	return ru.uid
}

func (ru *robotUser) IsValid() bool {
	return true
}
