package chatSystem

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/etcd"
	gmConfig "github.com/nghichtu91/platform/share/x/chat/web/config"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"reflect"
)

type EditChatStoreTimeRequest struct {
	CurrentChannel     string `form:"CurrentChannel"`
	GuildChannel       string `form:"GuildChannel"`
	SystemChannel      string `form:"SystemChannel"`
	TeamChannel        string `form:"TeamChannel"`
	WorldChannel       string `form:"WorldChannel"`
	ZhaoMuChannel      string `form:"ZhaoMuChannel"`
	GuildWarChannel    string `form:"GuildWarChannel"`
	GuildPartyChannel  string `form:"GuildPartyChannel"`
	CrossServerChannel string `form:"CrossServerChannel"`
	WarFieldChannel    string `form:"WarFieldChannel"`
	ActivityChannel    string `form:"ActivityChannel"`
	CountryChannel     string `form:"CountryChannel"`
}

type EditChatStoreTimeResponse struct {
}

func (req *EditChatStoreTimeRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	resp := &EditChannelEtcdResponse{}
	t := reflect.TypeOf(req).Elem()
	v := reflect.ValueOf(req).Elem()
	// 根据前端传递的参数动态生成filter
	for k := 0; k < t.NumField(); k++ {

		key := fmt.Sprintf("%s/%d/%s/%s",
			gmConfig.Cfg.LocalConfig.ServerEtcdRoot,
			gmConfig.Cfg.LocalConfig.Gid,
			consts.KeyChatStoreTimePath,
			t.Field(k).Tag.Get("form"))
		value := v.Field(k).String()

		if err := etcd.Put(key, value); err != nil {
			return nil, err, fmt.Sprint(err)
		}
	}
	return resp, nil, ""
}
