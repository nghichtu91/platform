package chatSystem

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/etcd"
	gmConfig "github.com/nghichtu91/platform/share/x/chat/web/config"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

type QueryChannelEtcdRequest struct {
}

type QueryChannelEtcdResponse struct {
	ResData map[string]string `json:"data"`
}

type QueryChatStoreTimeRequest struct {
}

type QueryChatStoreTimeResponse struct {
	ResData map[string]string `json:"data"`
}

func (req *QueryChannelEtcdRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	resp := &QueryChannelEtcdResponse{}
	key := fmt.Sprintf("%s/%d/chat_config_path", gmConfig.Cfg.LocalConfig.ServerEtcdRoot, gmConfig.Cfg.LocalConfig.Gid)
	etcdRes, _, err := etcd.GetSubRecursiveWithRev(key)
	if err != nil {
		return nil, err, fmt.Sprint(err)
	}
	resp.ResData = etcdRes
	return resp, nil, ""
}

func (req *QueryChatStoreTimeRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	resp := &QueryChatStoreTimeResponse{}
	key := fmt.Sprintf("%s/%d/%s", gmConfig.Cfg.LocalConfig.ServerEtcdRoot, gmConfig.Cfg.LocalConfig.Gid, consts.KeyChatStoreTimePath)
	etcdRes, _, err := etcd.GetSubRecursiveWithRev(key)
	if err != nil {
		return nil, err, fmt.Sprint(err)
	}
	resp.ResData = etcdRes
	return resp, nil, ""
}
