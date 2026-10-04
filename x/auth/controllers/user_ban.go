package controllers

import (
	"errors"
	"fmt"
	"sync/atomic"
	"unsafe"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/x/auth/config"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	internalConfig "github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/auth/models"

	"github.com/astaxie/beego/httplib"
)

func BanUser(uid, gid string, time_to_ban int64, reason string) error {
	// 先封掉账号
	err := models.SetBanUnInfo(uid, time_to_ban, reason)
	if err != nil {
		return err
	}

	// 再通知login把在线的全踢掉
	return notifyLoginServerKick(uid, gid, "IDS_ERROR_NETWORK_90000", time_to_ban)
}

func notifyLoginServerKick(uid, gid, reason string, banTime int64) error {
	loginUrl := *(*string)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&config.LoginUrl))))
	URL := fmt.Sprintf("%s%s?id=%s&gid=%s&banTime=%d&reason=%s&byGM=yes",
		loginUrl, internalConfig.Router_Login_Kick, uid, gid, banTime, reason)
	var rst struct {
		Result string `json:"result"`
	}
	req := httplib.Get(URL)
	err := req.ToJSON(&rst)
	if err != nil {
		tilogs.L().Errorf("notifyLoginServerKick failed %s", err.Error())
		return err
	}

	rsp, _ := req.Response()
	defer rsp.Body.Close()

	if rst.Result != "ok" {
		return errors.New(rst.Result)
	}

	return nil
}

//封禁角色
func BanPlayer(uid, acid, gid string, banTime int64, reason string) error {
	// 先封掉账号下该角色
	err := models.SetBanPlayer(uid, acid, banTime, reason)
	if err != nil {
		return err
	}

	if banTime > 0 {
		// 再通知login把在线的全踢掉
		time := banTime - timeutil.Now().Unix()
		return notifyLoginServerKick(uid, gid, "IDS_System_Role_Banned", time) //IDS_ERROR_NETWORK_90000
	}
	return nil
}
