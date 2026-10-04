package ntsdk

import (
	"strconv"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/timeutil"
)

const release = "RELEASE"

var (
	SdkNtCfg SdkNtConfig
)

type SdkNtConfig struct {
	CommonCfg   SdkNtCommonCfg `toml:"SdkNtCommonConfig"`   // 国内通用配置
	HWCommonCfg SdkNtCommonCfg `toml:"SdkNtHWCommonConfig"` // 海外通用配置
	SdkCfg      SdkNtAppConfig `toml:"SdkNtConfig"`         // 国内服务器配置
	HMTCfg      SdkNtAppConfig `toml:"SdkNtHMTConfig"`      // 港澳台东南亚安卓服务器配置
	JPCfg       SdkNtAppConfig `toml:"SdkNtJPConfig"`       // 日本服务器配置

	CNGameRequestCfg     SdkGameRequestAddr `toml:"CNGameRequestCfg"`     // 国内游戏请求地址
	AbroadGameRequestCfg SdkGameRequestAddr `toml:"AbroadGameRequestCfg"` // 海外游戏请求地址
	VNCfg                SdkNtAppConfig     `toml:"SdkNtVNConfig"`        // 越南服务器配置
}

type SdkNtCommonCfg struct {
	FormalLoginIpAddress       string `toml:"formalLoginIpAddress"`
	SandBoxLoginIpAddress      string `toml:"sandBoxLoginIpAddress"`
	FormalGameGidIpAddress     string `toml:"formalGameGidIpAddress"`
	SandBoxGameGidIpAddress    string `toml:"sandBoxGameGidIpAddress"`
	FormalGameSharedIpAddress  string `toml:"formalGameSharedIpAddress"`
	SandBoxGameSharedIpAddress string `toml:"sandBoxGameSharedIpAddress"`
	ContentType                string `toml:"contentType"`
}

type SdkNtAppConfig struct {
	AppId     string `toml:"appId"`
	SecretKey string `toml:"secretKey"`
}

type SdkGameRequestAddr struct {
	NTGameRequestAddr   string `toml:"nt_game_request_addr"`    // NTGame请求地址
	NTGameRequestQAAddr string `toml:"nt_game_request_qa_addr"` // NTGameQA环境请求地址
}

func GetNtAppConfig() SdkNtAppConfig {
	switch {
	case timeutil.IsAsiaShanghaiTZ(): // 国内
		return SdkNtCfg.SdkCfg
	case timeutil.IsJPTZ():
		return SdkNtCfg.JPCfg
	case timeutil.IsVN():
		return SdkNtCfg.VNCfg
	default: // 海外
		return SdkNtCfg.HMTCfg
	}
}

func GetNtSdkAppIdForGamePush() []string {
	return []string{GetNtAppConfig().AppId}
}

func GetNtSdkAppId() string {
	return GetNtAppConfig().AppId
}

func GetNtSdkAppIdInt() int {
	id, _ := strconv.Atoi(GetNtSdkAppId())
	return id
}

func GetContentType() string {
	if timeutil.IsAsiaShanghaiTZ() { // 国内
		return SdkNtCfg.CommonCfg.ContentType
	} else { // 海外
		return SdkNtCfg.HWCommonCfg.ContentType
	}
}

func GetLoginUrl(environment string) string {
	if timeutil.IsAsiaShanghaiTZ() { // 国内
		if environment == release { // 正式环境
			return SdkNtCfg.CommonCfg.FormalLoginIpAddress
		}
		return SdkNtCfg.CommonCfg.SandBoxLoginIpAddress
	} else { // 海外
		if environment == release { // 正式环境
			return SdkNtCfg.HWCommonCfg.FormalLoginIpAddress
		}
		return SdkNtCfg.HWCommonCfg.SandBoxLoginIpAddress
	}
}

func GetSecretKeyForGamePush() []string {
	return []string{GetNtAppConfig().SecretKey}
}

func GetSecretKey() string {
	return GetNtAppConfig().SecretKey
}

func GetSharedIpAddress(runMode string) string {
	if timeutil.IsAsiaShanghaiTZ() { // 国内
		if planx.IsRunProd(runMode) { // 正式环境
			return SdkNtCfg.CommonCfg.FormalGameSharedIpAddress
		}
		return SdkNtCfg.CommonCfg.SandBoxGameSharedIpAddress
	} else { // 海外
		if planx.IsRunProd(runMode) { // 正式环境
			return SdkNtCfg.HWCommonCfg.FormalGameSharedIpAddress
		}
		return SdkNtCfg.HWCommonCfg.SandBoxGameSharedIpAddress
	}
}

func GetGidIpAddress(runMode string) string {
	if timeutil.IsAsiaShanghaiTZ() { // 国内
		if planx.IsRunProd(runMode) { // 正式环境
			return SdkNtCfg.CommonCfg.FormalGameGidIpAddress
		}
		return SdkNtCfg.CommonCfg.SandBoxGameGidIpAddress
	} else { // 海外
		if planx.IsRunProd(runMode) { // 正式环境
			return SdkNtCfg.HWCommonCfg.FormalGameGidIpAddress
		}
		return SdkNtCfg.HWCommonCfg.SandBoxGameGidIpAddress
	}
}

func GetSercetKeyByAppId(appId string) string {
	switch appId {
	case SdkNtCfg.SdkCfg.AppId:
		return SdkNtCfg.SdkCfg.SecretKey
	case SdkNtCfg.HMTCfg.AppId:
		return SdkNtCfg.HMTCfg.SecretKey
	case SdkNtCfg.JPCfg.AppId:
		return SdkNtCfg.JPCfg.SecretKey
	case SdkNtCfg.VNCfg.AppId:
		return SdkNtCfg.VNCfg.SecretKey
	default:
		return ""
	}
}

// GetGameRequestAddr 获取游戏请求地址
func GetGameRequestAddr(runMode string) string {
	if planx.IsRunProd(runMode) {
		if timeutil.IsAsiaShanghaiTZ() { // 国内
			return SdkNtCfg.CNGameRequestCfg.NTGameRequestAddr
		} else { // 海外
			return SdkNtCfg.AbroadGameRequestCfg.NTGameRequestAddr
		}
	} else {
		if timeutil.IsAsiaShanghaiTZ() { // 国内
			return SdkNtCfg.CNGameRequestCfg.NTGameRequestQAAddr
		} else { // 海外
			return SdkNtCfg.AbroadGameRequestCfg.NTGameRequestQAAddr
		}
	}
}
