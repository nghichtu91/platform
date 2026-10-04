package controllers

import (
	"github.com/nghichtu91/platform/share/x/auth/models"
)

func GagUser(uid, gid string, time_to_ban int64) error {
	// 两方面 第一要设置禁言的标签 第二要通知在线的账号禁言
	err := models.SetGagUnInfo(uid, time_to_ban)
	if err != nil {
		return err
	}

	return nil

	// 废弃此方法，改为推送到gamex
	// 再通知login把在线的
	//return notifyLoginServerSendGagInfo(uid, gid, time_to_ban)
}
