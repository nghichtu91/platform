package models

import "github.com/nghichtu91/platform/share/planx/servers/db"

const ()

// LoginRegAuthToken
func LoginRegAuthToken(authToken string, userID db.UserID, sdkDeviceId string) error {
	// var time_out int64 = AUTHTOKEN_TIMEOUT
	// return db_interface.SetAuthToken(authToken, userID, time_out, sdkDeviceId)
	// JWS2-19885 将token处理改为使用redis
	return SetAuthToken(authToken, userID, sdkDeviceId)
}

func LoginVerifyAuthToken(authToken, prefix string) (db.UserID, string, error) {
	// return db_interface.GetAuthToken(authToken)
	// JWS2-19885 将token处理改为使用redis
	return GetAuthToken(authToken, prefix)
}

func GetSdkDeviceIdByUid(uid string) (string, error) {
	return db_interface.GetSdkDeviceIdByUid(uid)
}
