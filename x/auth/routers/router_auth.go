package routers

import (
	"github.com/gin-gonic/gin"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/limit"
	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/auth/controllers"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

func statsCCU(c *gin.Context) {
	allmetrics.AddAuthValidReqCount()
}

func RegAuth(g *gin.Engine) {
	auth_controller := controllers.AuthController{}

	auth_public := g.Group("/auth/v1/")
	auth_public.Use(limit.CheckIdentity(consts.Spec_Header, consts.Spec_Header_Content), limit.RateLimit, statsCCU)

	if planx.IsCheatEnable(config.GidCfg.CheatEnable) {
		// unity3d登录, 返回一下uid
		auth_public.GET("user/login", auth_controller.LoginAsUser())
		// unity3d注册并登录, 返回一下uid
		auth_public.GET("user/reg/:id", auth_controller.RegisterAndLogin())
		// 已有账号，用uid登录
		auth_public.GET("user/login/:uid", auth_controller.LoginByUid())
	}

	// 手机sdk登录
	auth_publicV2 := g.Group("/auth/v2/")
	auth_publicV2.Use(limit.CheckIdentity(consts.Spec_Header, consts.Spec_Header_Content), limit.RateLimit, statsCCU)
	auth_publicV2.POST("sdk/login", auth_controller.LoginWithSdk())

	auth_publicV2.POST("nt/:id", auth_controller.QueryNTID)
	// 修改密码
	auth_publicV2.POST("user/resetPwd", auth_controller.ResetPasswordByCheat())

	// 下面是测试接口
	if config.GidCfg.RunMode == planx.RunMode_PrefTest {
		g.GET("/auth/testsentry", auth_controller.TestSentry())
	}

	auth_test := g.Group("/test")
	auth_test.Use(limit.RateLimit)
	auth_test.GET("echoip", auth_controller.EchoIp())

}

func RegLogin(g *gin.Engine) {
	loginController := controllers.LoginController{}

	login_public := g.Group("/login/v1/")
	login_public.Use(limit.CheckIdentity(consts.Spec_Header, consts.Spec_Header_Content), limit.RateLimit, statsCCU)

	if config.GidCfg.RunMode != "prod" {
		// DebugKick是调试用API，协助客户端测试踢人操作
		login_public.GET("debug/kick", loginController.DebugKick())
	}

	// 进入gate连入gamex
	login_public.GET("getgate", loginController.GetGate())

	login_public_v2 := g.Group("/login/v2/")
	login_public_v2.Use(limit.CheckIdentity(consts.Spec_Header, consts.Spec_Header_Content), limit.RateLimit, statsCCU)
	// 获取服务器列表
	login_public_v2.GET("shards/:gid", loginController.FetchShardsInfoV2())
}
