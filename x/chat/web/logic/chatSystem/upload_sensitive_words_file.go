package chatSystem

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"

	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	gmConfig "github.com/nghichtu91/platform/share/x/chat/web/config"
	"github.com/nghichtu91/platform/share/x/chat/web/model/oss"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"
)

/**
创建权限管理分组
*/

type UploadSensitiveWordsFile struct {
}

type UploadSensitiveWordsFileResponse struct {
}

func (req *UploadSensitiveWordsFile) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	// 将文件上传至oss
	resp := &UploadSensitiveWordsFileResponse{}
	file, fileMd5, err := parseFile(c)
	if err != nil {
		return nil, err, fmt.Sprint(err)
	}
	path := fmt.Sprintf("chat/logic/illegal_words_%d_common_illegal.csv", gmConfig.Cfg.LocalConfig.Gid)
	tilogs.L().Warnf("fail to parse file %v, %v", path)
	if err := oss.GMGetCloudDB.Put(path, file); err != nil {
		return nil, err, fmt.Sprint(err)
	}
	// 修改etcd
	key := fmt.Sprintf("%s/%d/chat_illegal_words_hash_path", gmConfig.Cfg.LocalConfig.ServerEtcdRoot, gmConfig.Cfg.LocalConfig.Gid)
	if _, err := etcd.GetEtcd().Put(context.Background(), key, fileMd5); err != nil {
		return nil, err, fmt.Sprint(err)
	}
	return resp, nil, ""
}

func parseFile(c *gin.Context) ([]byte, string, error) {
	file, _, err := c.Request.FormFile("sensitiveWordsFile")
	if err != nil {
		return nil, "", err
	}
	bytes, err := ioutil.ReadAll(file)
	b := md5.Sum(bytes)
	Md5Str := hex.EncodeToString(b[:])

	if err != nil {
		return nil, "", err
	}
	return bytes, Md5Str, nil
}
