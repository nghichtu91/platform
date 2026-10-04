package logic_log

import (
	googleproto "github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/auth/hero_log/common_log"

	sls "github.com/aliyun/aliyun-log-go-sdk"
)

func LogUserLoginBdcLog(logPlayerInfo common_log.LogPlayerInfo, adsJson string) {
	tilogs.L().Debugf("start send UserLoginBdcLog ")
	common_log.Log(logPlayerInfo, common_log.User_Login_event_id, []*sls.LogContent{
		{
			Key:   googleproto.String(common_log.User_Login_user_balance),
			Value: googleproto.String("{}"),
		},
		{
			Key:   googleproto.String(common_log.User_Login_ads_json),
			Value: googleproto.String(adsJson),
		},
	})
}
