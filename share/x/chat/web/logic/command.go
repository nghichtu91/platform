package logic

import (
	"fmt"
	"reflect"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/arrayhelper"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/chat/web/logic/chatSystem"
	"github.com/nghichtu91/platform/share/x/chat/web/logic/login"
	"github.com/nghichtu91/platform/share/x/chat/web/logic/shardId"
	"github.com/nghichtu91/platform/share/x/chat/web/logic/user"
	"github.com/nghichtu91/platform/share/x/chat/web/model/command"
)

// RegisterCommand TODO 设计
func RegisterCommand(g *gin.Engine) {
	post(g)
}

// TODO 支持get
func post(g *gin.Engine) {
	// login
	regHandler(g, command.GmCommandToken, &login.GetTokenRequest{})
	regHandler(g, command.GmCommandLogin, &login.LoginRequest{})
	regHandler(g, command.GmCommandQueryGid, &login.QueryGidRequest{})
	// user
	regHandler(g, command.GmCommandQueryUser, &user.QueryUserRequest{})
	regHandler(g, command.GmCommandUpdateUser, &user.UpdateUserRequest{})
	regHandler(g, command.GmCommandGetOneUser, &user.GetOneUserRequest{})
	regHandler(g, command.GmCommandDelUser, &user.DeleteUserRequest{})
	// power group
	regHandler(g, command.GmCommandCreatePower, &user.CreatePowerGroupRequest{})
	regHandler(g, command.GmCommandGetPower, &user.QueryPowerGroupRequest{})
	regHandler(g, command.GmCommandDelPower, &user.DeletePowerGroupRequest{})
	regHandler(g, command.GmCommandQueryRecord, &user.QueryRecordRequest{})
	// chatSystem
	regHandler(g, command.UploadSensitiveWordsFile, &chatSystem.UploadSensitiveWordsFile{})
	regHandler(g, command.SearchCharInfo, &chatSystem.SearchCharInfoRequest{})
	regHandler(g, command.GetShardIds, &shardId.GetShardRequest{})
	regHandler(g, command.QueryChannelEtcd, &chatSystem.QueryChannelEtcdRequest{})
	regHandler(g, command.EditChannelEtcd, &chatSystem.EditChannelEtcdRequest{})
	regHandler(g, command.DownloadSensitiveWordsFile, &chatSystem.DownloadSensitiveWordsFile{})
	regHandler(g, command.QueryChatStoreTime, &chatSystem.QueryChatStoreTimeRequest{})
	regHandler(g, command.EditChatStoreTime, &chatSystem.EditChatStoreTimeRequest{})

	// moreHandler
}

func regHandler(g *gin.Engine, reqKey string, reqType RequestInterface) {
	typeRegistry[reqKey] = reflect.TypeOf(reqType).Elem() // 向接口里注册数据，Handler里需要
	g.POST(reqKey, func(c *gin.Context) {
		if loginToken, err := preHandle(reqKey, c); err == nil {
			Handler(loginToken, reqKey, c)
		} else {
			tilogs.L().Errorf("pre handle %v", err)
			c.String(400, err.Error())
		}
	})
}

var ignoreHandleKeys = []string{
	command.GmCommandQueryGid,
	command.GmCommandToken,
	//command.GM_COMMAND_SET_PROFILE,
	// more ignoreCommand
}

// 1. 校验授权
// 2. 校验login_token
// 3. 检查权限
func preHandle(reqKey string, c *gin.Context) (string, error) {
	// 收集报告， 内部功能， 没有登录限制
	if arrayhelper.ContainsString(ignoreHandleKeys, reqKey) {
		return "", nil
	}

	// 1. 校验授权
	accessToken, err := c.Cookie("access_token")
	if err != nil {
		return "", err
	}
	if !login.HasAccessToken(accessToken) {
		return "", fmt.Errorf("未授权的请求")
	}

	if reqKey == command.GmCommandLogin {
		return "", nil
	}

	// 2. 校验login_token
	loginToken, err := c.Cookie("login_token")
	if err != nil {
		return "", err
	}
	session := sessions.Default(c)
	userInfo := session.Get(loginToken)

	if userInfo == nil {
		return "", fmt.Errorf("需要先登录")
	}

	// 3. 检查权限
	if !CheckPermission() {
		return "", fmt.Errorf("权限不足")
	}
	return loginToken, nil
}

func CheckPermission() bool {
	return true // TODO
}
