package models

import (
	"context"
	"errors"

	"github.com/nghichtu91/platform/share/x/auth/config_data"

	"github.com/nghichtu91/platform/share/planx/servers/db"
)

type DBInterface interface {
	Init(config DBConfig) error
	GetDeviceInfo(deviceID string, checkGM bool) (*DeviceUserInfo, error, bool)
	UpdateDeviceInfo(deviceID string, info *DeviceUserInfo) error
	SetDeviceInfo(deviceID, name string, uid db.UserID, channelID, device string) (*DeviceUserInfo, error)
	IsNameExist(name string) (int, error)
	// Deprecated
	GetUnKey(name string, checkGM bool) (db.UserID, error, bool)
	GetUnKeyV2(ctx context.Context, name string, checkGM bool) (db.UserID, error, bool)
	SetUnKey(name string, uid db.UserID) error
	// Deprecated
	GetUnInfo(uid db.UserID) (*UserInfo, error)
	GetUnInfoV2(ctx context.Context, uid db.UserID) (*UserInfo, error)
	// Deprecated
	UpdateUnInfo(uid db.UserID, deviceID, authToken string) error
	UpdateUnInfoV2(ctx context.Context, uid db.UserID, deviceID, authToken string) error
	UpdateRecallID(uid db.UserID, recallID string) error
	UpdateBanUn(uid string, time_to_ban int64, reason string) error
	UpdateGagUn(uid string, time_to_gag int64) error
	SetUnInfo(uid db.UserID, name, deviceID, passwd, email, authToken string) error
	SetUnInfoPass(uid db.UserID, name, deviceID, passwd, email, authToken string) error

	UpdateBanPlayer(uid string, acid string, endTime int64, reason string) error
	GetPlayerBanInfo(uid string, acid string) *BanPlayerInfo

	// IncrDeviceTotal() (db.UserID, error)

	// Deprecated
	// GetAuthToken(authToken string) (userID db.UserID, sdkDeviceId string, err error)
	// Deprecated
	// SetAuthToken(authToken string, userID db.UserID, time_out int64, sdkDeviceId string) error

	GetUserShardInfo(uid string) ([]AuthUserShardInfo, error)
	SetUserShardInfo(uid, gidSidStr string, roleName, roleLevel, mainHero, lastLoginTime, mainlineLevel, headIcon, playerId string) error

	GetUserHistoryShard(uid string) ([]string, error)
	SetUserHistoryShard(uid, shard string) error

	SetLastLoginShard(uid string, gidsid string) error
	GetLastLoginShard(uid string) (string, error)

	SetPayFeedBackIfNot(uid string) (error, bool)
	SetRecallPlayerInfo(recallId, uid string, recallnum, gid int64) error
	MarketGetByHashM(recallId string) (bool, error)
	QueryRecallIdByAcid(accountId string) (map[string]interface{}, error)

	AddWhiteList(ip, note string) error
	CheckWhiteList(ip string) (bool, error)
	// 白名单操作
	InsertOrUpdateWhiteList(infos *WhiteListInfo) ([]*WhiteListInfo, error)
	DeleteWhiteList(content string, typ uint) ([]*WhiteListInfo, error)
	QueryAllWhiteList() ([]*WhiteListInfo, error)
	QueryWhiteList(typ uint, param string) ([]*WhiteListInfo, error)

	GetSdkDeviceIdByUid(uid string) (string, error)
	InsertOrUpdateSdkIdToUid(infos *SdkIdToUidInfos) ([]*SdkIdToUidInfos, error)
	DeleteSdkIdToUid(unmberId string, shard uint) ([]*SdkIdToUidInfos, error)
	QuerySdkIdToUid() ([]*SdkIdToUidInfos, error)

	QueryUserSpecialAward(uid db.UserID) (string, error)
	UpdateUserSpecialAward(uid db.UserID, awardStr string) error

	RewardForUser(rewardType int, userID db.UserID, optType string) (map[int]*config_data.RewardInfo, int32)
}

// XXX 曾经这里有Redis的使用接口，目前只使用DynamoDB
var db_interface DBInterface = &DBByDynamoDB{}

var (
	XErrDBNotFound = errors.New("DB Not Found")
)
