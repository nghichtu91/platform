package routers

import (
	"fmt"

	"github.com/nghichtu91/platform/share/x/auth/controllers"

	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/x/auth/config"
)

func RegUserMgr(g *gin.Engine) {
	userMgrController := controllers.UserMgrController{}
	login_api := g.Group(config.Router_Login_Root)

	// gate->login
	login_api.POST(config.Router_Login_NotifyLogin, userMgrController.NotifyUserLogin())
	login_api.POST(config.Router_Login_NotifyLogout, userMgrController.NotifyUserLogout())
	// game->login
	login_api.POST(config.Router_Login_NotifyUserInfo, userMgrController.NotifyUserInfo())
	login_api.GET(config.Router_Login_Kick, userMgrController.NotifyKickUser())
	login_api.GET(config.Router_Login_Gag, userMgrController.NotifyGagUser())

	auth_api := g.Group(config.Router_auth_Root)
	auth_api.GET(fmt.Sprintf("%s/:id", config.Router_auth_ban), userMgrController.BanUser())
	auth_api.GET(fmt.Sprintf("%s/:id", config.Router_auth_is_ban), userMgrController.IsBan())
	auth_api.GET(fmt.Sprintf("%s/:id", config.Router_auth_gag), userMgrController.GagUser())                               // 禁言，应该没用了
	auth_api.GET(fmt.Sprintf("%s/:id", config.Router_auth_getshardnewusercount), userMgrController.GetShardNewUserCount()) // 获取shard newuser的数量

	auth_api.GET(fmt.Sprintf("%s/:id", config.Router_auth_ban_player), userMgrController.BanPlayer())
	auth_api.GET(fmt.Sprintf("%s/:id", config.Router_auth_is_ban_player), userMgrController.IsBanPlayer())

	g.GET(config.Router_Query_Gm_Uid, userMgrController.QueryGmUid())
	g.DELETE(config.Router_Delete_Gm_Uid, userMgrController.DeleteGmUid())
	g.POST(config.Router_InsertOrUpdate_Gm_Uid, userMgrController.InsertOrUpdateGmUid())

	//白名单操作
	g.GET(config.Router_Query_White_List, userMgrController.QueryWhiteList())
	g.POST(config.Router_InsertOrUpdate_White_List, userMgrController.InsertOrUpdateWhiteList())
	g.DELETE(config.Router_Delete_White_List, userMgrController.DeleteWhiteList())

	//特殊活动奖励操作
	g.GET(config.Router_Query_SpecialAward, userMgrController.QueryUserSpecialAward())
	g.POST(config.Router_Update_SpecialAward, userMgrController.UpdateUserSpecialAward())

	//充值返钻
	g.POST(config.Router_PayReward, userMgrController.PayRewardForUser())
}
