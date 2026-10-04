package auth

import (
	"encoding/json"
	"fmt"

	"github.com/astaxie/beego/httplib"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/auth/controllers"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/common/service_proto/common"
)

// RegRobotSDK 以机器人sdk类型登录auth正式环境
func RegRobotSDK(uri, uid string) (authToken string, err error) {
	authToken, err = RegInnerCommonSDK(uri, uid, consts.RobotChannel, consts.RobotDevice, "0", uid+consts.RobotSuffix)
	return
}

func RegInnerCommonSDK(uri, uid, channelId, device, typ string, token string) (authToken string, err error) {
	req := common.NewReq(uri+prodAuthPath, httplib.Post).
		Param("token", common.Enc(token)).
		Param("channelId", common.Enc(channelId)).
		Param("device", common.Enc(device)).
		Param("uid", common.Enc(uid)).
		Param("typ", common.Enc(typ))

	resp := new(controllers.SdkLoginRet)
	respByte, err := req.Bytes()
	if err != nil {
		return
	}

	err = json.Unmarshal(respByte, resp)
	if err != nil {
		tilogs.L().Errorf("unmarshal resp bytes failed, resp %s, err %s", string(respByte), err.Error())
		return
	}

	if resp.Result != "ok" {
		err = fmt.Errorf(resp.Err)
		return
	}

	authToken = resp.AuthToken
	return
}
