package login

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/x/chat/web/model/errs"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"
)

type GetTokenRequest struct {
	ClientId     string `json:"client_id" form:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret"`
	GrantType    string `json:"grant_type" form:"grant_type"`
}

type GetTokenResponse struct {
	AccessToken string `json:"access_token"`
}

const GM_WEB_KEY = "A0z5RWqyxFXEhFtr"
const GM_WEB_CLIENT_ID = "gmtool.taiyouxi.cn"
const GM_WEB_CLIENT_SECRET = "st4mtjULIuh2ks6r"

var lock sync.RWMutex
var AccessTokenMap = make(map[string]bool, 16)

func HasAccessToken(token string) bool {
	lock.RLock()
	defer lock.RUnlock()
	return AccessTokenMap[token]
}

func AddAccessToken(token string) {
	lock.Lock()
	defer lock.Unlock()
	AccessTokenMap[token] = true
}

func (req *GetTokenRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	var token = ""
	if req.ClientId == GM_WEB_CLIENT_ID && req.ClientSecret == GM_WEB_CLIENT_SECRET {
		ret := md5.Sum([]byte(fmt.Sprintf("%s%s%d%s", req.ClientId, req.ClientSecret, time.Now().Unix(), GM_WEB_KEY)))
		token = fmt.Sprintf("%x", ret)
		AddAccessToken(token)
	} else {
		return nil, errs.LogicError, errs.AccessToken
	}

	return &GetTokenResponse{
		AccessToken: token,
	}, nil, ""
}
