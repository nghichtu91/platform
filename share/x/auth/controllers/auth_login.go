package controllers

import (
	"context"

	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/auth/models"
	"github.com/nghichtu91/platform/share/x/gatex/gate"
)

func notifyLoginServer(ctx context.Context, authToken string, userID db.UserID, sdkDeviceId string) bool {
	span, _ := opentracing.StartSpanFromContext(ctx, "notifyLoginServer")
	defer span.Finish()

	err := models.LoginRegAuthToken(authToken, userID, sdkDeviceId)
	if err != nil {
		tilogs.L().Errorf("notifyLoginServer LoginRegAuthToken err %s", err.Error())
		return false
	}
	return true
}

// 防止同一个帐号多重login, 通知上一个登录的设备退出
func avoidMultipleLogin(gid uint, uid db.UserID, reason string, banTime int64, byGM bool, kickErrCode int, source string) (status string) {
	tilogs.L().Debugf("avoidMultipleLogin before GetLastLoginStatus gid = %v uid = %v", gid, uid)
	OldAccountID, OldLoginToken, OldRpc, ok := models.GetLastLoginStatus(gid, uid)
	tilogs.L().Debugf("avoidMultipleLogin after GetLastLoginStatus OldRpc = %v ok = %v", OldRpc, ok)

	if ok { // 如果这个账户已有登录信息，尝试通知退出
		var oldConn *rpc.Client
		var reply int
		var err error
		tilogs.L().Debugf("avoidMultipleLogin Dial OldRpc=%v", OldRpc)
		if oldConn, err = jsonrpc.Dial("tcp", OldRpc); err == nil {
			defer oldConn.Close()
			tilogs.L().Debugf("avoidMultipleLogin Dial GateServer.KickItOffline OldLoginToken = %v "+
				"OldAccountID = %v source=%v", OldLoginToken, OldAccountID, source)
			err = oldConn.Call("GateServer.KickItOffline",
				&gate.KickOfflineParam{
					LoginToken:  OldLoginToken,
					AccountID:   OldAccountID,
					Reason:      reason,
					KickErrCode: kickErrCode,

					AfterDuration:   1,
					NoLoginDuration: int(banTime),
					ByGM:            byGM,
					Source:          source,
				},
				&reply)
			tilogs.L().Infof("avoidMultipleLogin Dial GateServer.KickItOffline success，reply=%d OldLoginToken=%v OldAccountID=%v",
				reply, OldLoginToken, OldAccountID)

			if reply == 1 { // RPC通知成功，并已通知相关客户端下线, 请等一段时间后重试
				return "RETURN"
			} else {
				return "Gate told login server, not found that player"
			}
		} else {
			// 相关服务器上RPC通知失败，则允许继续登录
			tilogs.L().Warnf("[Login.getGate.call] kick out last login failed. rpc conn", err.Error())
			models.DeleteLoginStatus(gid, uid)
			return "Gate not exist"
		}
	}

	return "CONTINUE"
}
