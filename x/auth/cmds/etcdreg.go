package cmds

import (
	"encoding/json"
	"fmt"

	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/etcd"
	dis "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
)

var etcdMgr *dis.ServiceMgr

func AuthEtcdReg(serverId, loginUrl, internalAddr, publicAddr string) error {
	etcdMgr = dis.NewSeriveMgr(
		dis.ServiceConfig{
			EtcdRoot: config.Cfg.CommonCfg.EtcdServer,
			SerId: dis.ServiceId{
				SerTyp: dis.ServiceType{
					Gid:    fmt.Sprintf("%d", config.Cfg.CommonCfg.Gid),
					SerTyp: etcd.Ser_Auth,
				},
				SerId: serverId,
			},
			HasPublicAddr:  true,
			HasPrivateAddr: true,
			HasExtraInfo:   true,
			CheckHealth:    true,
		}, &AuthEtcdServiceInfo{
			loginUrl:     loginUrl,
			internalAddr: internalAddr,
			publicAddr:   publicAddr,
		})
	return etcdMgr.Start()
}

func AuthEtcdStop() {
	if etcdMgr != nil {
		etcdMgr.Stop()
		tilogs.L().Infof("AuthEtcdStop")
	}
}

type AuthEtcdServiceInfo struct {
	loginUrl     string
	internalAddr string
	publicAddr   string
}

func (info *AuthEtcdServiceInfo) GetPublicAddr() string {
	return info.publicAddr
}
func (info *AuthEtcdServiceInfo) GetPrivateAddr() string {
	return info.internalAddr
}
func (info *AuthEtcdServiceInfo) GetExtraInfo() string {
	authUrl := fmt.Sprintf("http://%s%s", info.internalAddr,
		config.Router_auth_Root)
	bb, err := json.Marshal(consts.AuthExtraInfo{
		InternalAddr: info.internalAddr,
		PublicAddr:   info.publicAddr,
		LoginGateRegUrl: fmt.Sprintf("%s%s", info.loginUrl,
			config.Router_Login_GateReg),
		LoginLoginUrl: fmt.Sprintf("%s%s", info.loginUrl,
			config.Router_Login_NotifyLogin),
		LoginLogoutUrl: fmt.Sprintf("%s%s", info.loginUrl,
			config.Router_Login_NotifyLogout),
		LoginNotifyUser: fmt.Sprintf("%s%s", info.loginUrl,
			config.Router_Login_NotifyUserInfo),
		AuthBanUrl: fmt.Sprintf("%s%s", authUrl,
			config.Router_auth_ban),
		AuthIsBanUrl: fmt.Sprintf("%s%s", authUrl,
			config.Router_auth_is_ban),
		AuthGagUrl: fmt.Sprintf("%s%s", authUrl,
			config.Router_auth_gag),
		AuthKickUrl: fmt.Sprintf("%s%s", authUrl,
			config.Router_Login_Kick),
		GetShardNewUseCount: fmt.Sprintf("%s%s", authUrl,
			config.Router_auth_getshardnewusercount),
		QuerySdkIdUrl: fmt.Sprintf("http://%s%s", info.internalAddr,
			config.Router_Query_Gm_Uid),
		InsertOrUpdateSdkIdUrl: fmt.Sprintf("http://%s%s", info.internalAddr,
			config.Router_InsertOrUpdate_Gm_Uid),
		DeleteSdkIdUrl: fmt.Sprintf("http://%s%s", info.internalAddr,
			config.Router_Delete_Gm_Uid),
		QueryWhiteList: fmt.Sprintf("http://%s%s", info.internalAddr,
			config.Router_Query_White_List),
		DeleteWhiteList: fmt.Sprintf("http://%s%s", info.internalAddr,
			config.Router_Delete_White_List),
		InsertOrUpdateWhiteList: fmt.Sprintf("http://%s%s", info.internalAddr,
			config.Router_InsertOrUpdate_White_List),
		QueryUserSpecialAward: fmt.Sprintf("http://%s%s", info.internalAddr,
			config.Router_Query_SpecialAward),
		UpdateUserSpecialAward: fmt.Sprintf("http://%s%s", info.internalAddr,
			config.Router_Update_SpecialAward),
		PayRewardForUser: fmt.Sprintf("http://%s%s", info.internalAddr,
			config.Router_PayReward),

		AuthBanPlayerUrl: fmt.Sprintf("%s%s", authUrl,
			config.Router_auth_ban_player),
		AuthIsBanPlayerUrl: fmt.Sprintf("%s%s", authUrl,
			config.Router_auth_is_ban_player),
	})
	if err != nil {
		tilogs.L().Errorf("AuthEtcdServiceInfo GetExtraInfo err %s", err.Error())
		return ""
	}
	return string(bb)
}

func (info *AuthEtcdServiceInfo) GetTickExtraInfo() string {
	return ""
}
