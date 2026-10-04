package config

const (
	Router_Login_Root           = "/login/v1/api/"
	Router_Login_AuthToken      = "authtoken"
	Router_Login_Kick           = "kick"
	Router_Login_Gag            = "gag"
	Router_Login_GateReg        = "gateregister"
	Router_Login_NotifyLogin    = "notifylogin"
	Router_Login_NotifyLogout   = "notifylogout"
	Router_Login_NotifyUserInfo = "notifyuserinfo"

	Router_auth_Root                 = "/auth/v1/api/"
	Router_auth_ban                  = "user/ban"
	Router_auth_is_ban               = "user/isban"
	Router_auth_gag                  = "user/gag"
	Router_auth_getshardnewusercount = "user/getshardnewusercount"

	Router_auth_ban_player    = "player/ban"
	Router_auth_is_ban_player = "player/isban"

	//路由查询Gm_uid
	Router_Query_Gm_Uid          = "/gm_uid/query "
	Router_InsertOrUpdate_Gm_Uid = "/gm_uid/insert_update"
	Router_Delete_Gm_Uid         = "/gm_uid/delete"

	//路由白名单操作
	Router_Query_White_List          = "/white_list/query"
	Router_InsertOrUpdate_White_List = "/white_list/insert_update"
	Router_Delete_White_List         = "/white_list/delete"

	//特殊活动领奖操作
	Router_Query_SpecialAward  = "/special_award/query"
	Router_Update_SpecialAward = "/special_award/update"

	//充值返钻
	Router_PayReward = "/pay_reward/opt"
)
