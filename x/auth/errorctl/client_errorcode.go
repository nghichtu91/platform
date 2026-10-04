package errorctl

import (
	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/auth/models"
)

type ErrorCodeForClient int

// 客户端对接使用的错误码，使得客户端能够正确处理错误跳转逻辑
// NOTE：请注意同事更新两个文档：
// - Auth&Login/Auth_ClientUI.md
// - http://wiki.taiyouxi.net/w/3k_engineer/client/interface/
// 并请和客户端沟通具体错误码的出现情况和客户端应该如何处理
// 帮助客户端合理友善的展示错误处理错误流程
const (
	ClientErrorNormal                   ErrorCodeForClient = 0   // Nothing
	ClientErrorUnknown                                     = 100 // 通用：不应该出现的情况
	ClientErrorFormatVerifyFailed                          = 101 // 通用： 请求的加密格式非法
	ClientErrorUsernameRegFormatIllegal                    = 103 // 用户名格式非法
	ClientErrorMaybeDBProblem                              = 102 // 通用： 数据库无法返回正确值
	ClientErrorHaveToUsePassword                           = 201 // 设备ID认证：玩家设备绑定了帐号密码，强制要求使用用户名密码登录
	ClientErrorUsernameNotFound                            = 202 // 用户名密码认证：用户名称未找到
	ClientErrorUserPasswordIncorrect                       = 203 // 用户名密码认证：用户密码不正确
	ClientErrorRegCodeProblem                              = 204 // 设备ID认证： 注册码验证错误
	ClientErrorRegCodeUsedProblem                          = 205 // 设备ID认证： 注册码已被使用过
	ClientErrorHeroSdkToken                                = 210 // SDK: 英雄sdk查token返回错误
	ClientErrorHeroSdkGetUser                              = 211 // SDK: 英雄sdk查user信息返回错误
	ClientErrorQuickSdkCheckUser                           = 220 // SDK: Quicksdk checkuser错误
	ClientErrorUsernameHasBeenUsed                         = 301 // 注册：用户名称已存在
	ClientErrorDeviceAlreadyBinded                         = 302 // 注册：该设备关联的存档已经绑定过用户名,无法再次绑定
	ClientErrorLoginAuthtokenNotReady                      = 401 // 登录网关：无法获取网关，因为认证数据无效
	ClientErrorGetGateNotExist                             = 402 // 登录网关：当前分服无可用网关
	ClientErrorUpdateVerParamErr                           = 501 // 客户端更新重定向，参数错误
	ClientErrorUpdateVerUrlNotFound                        = 502 // 客户端更新重定向，未找到url

	// auth服登录返回无需服务器记录的错误信息
	ClientErrorCodeNeedLog                = 100000 // 小于此数的 error code 需要服务器日志记录
	ClientErrorBanByGM                    = 100001 // 用户名密码认证：用户被禁止登陆
	ClientErrorGetGateMaintenance         = 100002 // 登录网关：服务器维护中
	ClientErrorGetGateServerIsFull        = 100003 // 服务器爆满，不允许注册新账号
	ClientErrorGetGateServerIsAbnormality = 100004 // 客户端不应该看到的服务器，看到了，点登录，拦截提示
	ClientErrorBanPlayerByGm              = 100005 // 您的账号已被封禁，封禁原因为：%s1。解封时间：%s2
)

type HttpErrorCode int

const (
	FailForClient HttpErrorCode = 400 // 因为客户端的原因，导致请求失败
	FailForServer               = 500 // 因为服务器的原因，导致请求失败
)

func CtrlErrorReturn(c *gin.Context, prefix string, err error, code HttpErrorCode, clientErrorCode ErrorCodeForClient) {
	errInfo := err.Error()
	switch config.GidCfg.RunMode {
	case "prod":
		// prod环境错误信息应该加密
		errInfo = secure.DefaultEncode.Encode64ForNet([]byte(errInfo))
	default:
		// do nothing
	}

	if IsErrorReturnLog(clientErrorCode) && err != models.ErrAccessSpeedExceedLimit {
		tilogs.L().Errorf("%s code %d err %s", prefix, clientErrorCode, err.Error())
	}

	c.JSON(int(code), struct {
		Result    string             `json:"result"`
		Error     string             `json:"error,omitempty"`
		ForClient ErrorCodeForClient `json:"forclient,omitempty"`
	}{
		Result:    "no",
		ForClient: clientErrorCode,
		Error:     errInfo,
	})
}

func CtrlBanReturn(c *gin.Context, prefix string, err error, code HttpErrorCode, clientErrorCode ErrorCodeForClient, banTime int64, banReason string) {
	tilogs.L().Errorf("Error In: %s, error: %s, clientErrorCode:%d, banTime:%d, banReason:%s",
		prefix, err.Error(), clientErrorCode, banTime, banReason)
	errInfo := err.Error()
	switch config.GidCfg.RunMode {
	case "prod":
		// prod环境错误信息应该加密
		errInfo = secure.DefaultEncode.Encode64ForNet([]byte(errInfo))
	default:
		// do nothing
	}

	c.JSON(int(code), struct {
		Result    string             `json:"result"`
		Error     string             `json:"error,omitempty"`
		BanReason string             `json:"banreason,omitempty"`
		BanTime   int64              `json:"bantime,omitempty"`
		ForClient ErrorCodeForClient `json:"forclient,omitempty"`
	}{
		Result:    "no",
		ForClient: clientErrorCode,
		Error:     errInfo,
		BanTime:   banTime,
		BanReason: banReason,
	})
}

func IsErrorReturnLog(errorCode ErrorCodeForClient) bool {
	return errorCode < ClientErrorCodeNeedLog
}
