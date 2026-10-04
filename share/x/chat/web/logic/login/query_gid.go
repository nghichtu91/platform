package login

import (
	"fmt"
	"github.com/nghichtu91/platform/share/x/chat/web/config"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"

	"github.com/gin-gonic/gin"
	//"vcs.taiyouxi.net/platform/x/gm_tools/config"
	//"vcs.taiyouxi.net/platform/x/gm_tools/model/server"
)

type QueryGidRequest struct {
}

type QueryGidResponse struct {
	Gid string `json:"gid"`
}

func (req *QueryGidRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	gid := config.Cfg.LocalConfig.Gid
	//isChannel := server.IsChannelServer(gid)
	//name := "正式"
	//if isChannel {
	//	name = "渠道"
	//}
	return &QueryGidResponse{Gid: fmt.Sprintf("%d", gid)}, nil, ""
	//return &QueryGidResponse{Gid: fmt.Sprintf("%d%s", gid, name)}, nil, ""
}
