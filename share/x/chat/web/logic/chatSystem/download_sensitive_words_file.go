package chatSystem

import (
	"fmt"
	"github.com/gin-gonic/gin"
	gmConfig "github.com/nghichtu91/platform/share/x/chat/web/config"
	"github.com/nghichtu91/platform/share/x/chat/web/model/oss"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"
)

/**
创建权限管理分组
*/

type DownloadSensitiveWordsFile struct {
}

type DownloadSensitiveWordsFileResponse struct {
	FileName string
	Data     []byte
}

func (t *DownloadSensitiveWordsFileResponse) GetFileName() string {
	return t.FileName
}

func (t *DownloadSensitiveWordsFileResponse) GetData() []byte {
	return t.Data
}

func (req *DownloadSensitiveWordsFile) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	resp := &DownloadSensitiveWordsFileResponse{}
	path := fmt.Sprintf("chat/logic/illegal_words_%d_common_illegal.csv", gmConfig.Cfg.LocalConfig.Gid)
	if file, err := oss.GMGetCloudDB.Get(path); err != nil {
		return nil, err, fmt.Sprint(err)
	} else {
		resp.Data = file
		resp.FileName = fmt.Sprintf("illegal_words_%d_common_illegal.csv", gmConfig.Cfg.LocalConfig.Gid)
		return resp, nil, ""
	}
}
