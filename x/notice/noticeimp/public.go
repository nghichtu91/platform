package noticeimp

import (
	"github.com/gin-gonic/gin"

	"github.com/nghichtu91/platform/share/planx/ginhelper"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type NoticePublicController struct {
	ginhelper.GinController
}

func (pc *NoticePublicController) GetNoticeHandler(c *gin.Context) {
	//pc.GetString(c, "ip")

	gid := pc.GetString(c, "gid")
	version := pc.GetString(c, "version")
	shardId := pc.GetString(c, "shardId")
	channelId := pc.GetString(c, "channelId")
	subChannelId := pc.GetString(c, "subChannelId") // 主要用于获取强更地址
	language := pc.GetString(c, "language")
	platform := pc.GetString(c, "platform")
	tilogs.L().Debugf("[GetNotice][Request][Notice]gid=%s&version=%s&shardId=%s&channelId=%s&subChannelId=%s&language=%s&platform=%s",
		gid, version, shardId, channelId, subChannelId, language, platform)

	var retNotice string
	if shardId != "" {
		tilogs.L().Debugf("Get InnerNotice ")
		retNotice = GetInnerNotice(gid, version, shardId, channelId, language, platform)
	} else {
		tilogs.L().Debugf("Get OuterNotice ")
		retNotice = GetOuterNotice(gid, version, channelId, subChannelId, language, platform)
	}
	c.String(200, retNotice)
	tilogs.L().Debugf("[GetNotice][Response][Notice]%s", retNotice)
}
