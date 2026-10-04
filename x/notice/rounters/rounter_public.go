package rounters

import (
	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/limit"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/notice/noticeimp"
)

//统计公告的请求，调用一次增加1
func statsReq(c *gin.Context) {
	allmetrics.AddNoticeReqCount()
}

//注册公告
func RegPublic(g *gin.Engine) {
	controller := noticeimp.NoticePublicController{}

	auth_public := g.Group("/notice/v1/")
	auth_public.Use(
		//添加中间件，验证身份，限制频率，统计访问次数
		limit.CheckIdentity(consts.Spec_Header, consts.Spec_Header_Content),
		limit.RateLimit, statsReq)

	// 获取公告
	auth_public.GET("getnotice", controller.GetNoticeHandler)
}
