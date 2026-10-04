package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nghichtu91/platform/share/x/auth/config_data"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/gomodule/redigo/redis"

	cfg "github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"

	"github.com/nghichtu91/platform/share/planx/dynamodb"
	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/servers/db"
)

type DBByDynamoDB struct {
	db *AuthDynamoDB
}

func (d *DBByDynamoDB) UpdateRecallID(uid db.UserID, recallID string) error {
	uidkey := makeAuthUidKey(uid)
	data := make(map[string]interface{}, 4)

	data["recall_id"] = recallID
	return d.db.UpdateByHash(cfg.Cfg.CommonCfg.AuthUserInfoTable, uidkey, data)
}

func (s *DBByDynamoDB) MarketGetByHashM(hash string) (bool, error) {
	data, err := s.db.GetByHashM("recallplayer", hash)
	if len(data) <= 0 || err != nil {
		return false, err
	}
	return true, err
}

func (s *DBByDynamoDB) QueryRecallIdByAcid(accountId string) (map[string]interface{}, error) {
	return s.db.QueryByAccountId("recallplayer", "user_id-index", accountId)
}
func (d *DBByDynamoDB) Init(config DBConfig) error {
	tilogs.L().Debugf("DBByDynamoDB init")
	d.db = &AuthDynamoDB{DynamoDB: &dynamodb.DynamoDB{}}
	d.db.Connect(
		config.DynamoRegion,
		config.DynamoAccessKeyID,
		config.DynamoSecretAccessKey,
		config.DynamoSessionToken)

	// DynamoDB表操作之间需要等待一段时间
	// TODO 这里初始化DynamoDB的操作应该单独一个小工具

	err := d.db.InitTable()
	if err != nil {
		tilogs.L().Errorf("DBByDynamoDB Init err %s", err.Error())
		return err
	}

	//time.Sleep(1 * time.Second)
	d.db.CreateHashTable(cfg.Cfg.CommonCfg.AuthDeviceTable, "Id", "S")
	//time.Sleep(1 * time.Second)
	d.db.CreateHashTable(cfg.Cfg.CommonCfg.AuthNameTable, "Name", "S")
	//time.Sleep(1 * time.Second)
	d.db.CreateHashTable(cfg.Cfg.CommonCfg.AuthUserInfoTable, "UId", "S")

	d.db.CreateHashTable(cfg.Cfg.CommonCfg.AuthGmTable, "device", "S")
	//time.Sleep(1 * time.Second)
	//d.db.CreateHashTable(cfg.Cfg.Dynamo_NameAuthToken, "Token", "S")
	//time.Sleep(5 * time.Second)

	err = d.db.InitTable()
	if err != nil {
		tilogs.L().Errorf("DBByDynamoDB Init err %s", err.Error())
		return err
	}
	return nil
}

func getFromAnys2String(typ, name string, data map[string]interface{}) (string, bool) {
	data_any, name_ok := data[name]
	if !name_ok {
		tilogs.L().Warnf("getFromAnys2String %s Err by no %s", typ, name)
		return "", false
	}

	data_str, ok := data_any.(string)
	if !ok {
		tilogs.L().Warnf("getFromAnys2String %s Err by %s no string", typ, name)
		return "", false
	}

	return data_str, true
}

func getFromAnys2int64(typ, name string, data map[string]interface{}) (int64, bool) {
	data_any, name_ok := data[name]
	if !name_ok {
		tilogs.L().Warnf("getFromAnys2int64 %s Err by no %s", typ, name)
		return 0, false
	}

	data_int64, ok := data_any.(int64)
	if !ok {
		tilogs.L().Warnf("getFromAnys2int64 %s Err by %s no string", typ, name)
		return 0, false
	}

	return data_int64, true
}

// GetDeviceInfo
// 返回数据库级错误，或者XErrDBNotFound
func (d *DBByDynamoDB) GetDeviceInfo(deviceID string, checkGM bool) (
	info *DeviceUserInfo, err error, isGM bool) {
	di := &DeviceUserInfo{}
	// 首先去gm表查
	if checkGM {
		data_uid, err := d.db.GetByHash(cfg.Cfg.CommonCfg.AuthGmTable, deviceID)
		if err == nil {
			uids, ok := data_uid.(string)
			if ok {
				di.UserId = db.UserIDFromStringOrNil(uids)
				tilogs.L().Warnf("GM login by %s %s", deviceID, di.UserId.String())
				return di, nil, true
			}
		}
	}

	data, err := d.db.GetByHashM(cfg.Cfg.CommonCfg.AuthDeviceTable, deviceID)
	if err != nil {
		return nil, err, false
	}

	ok := true
	user_id, uok := getFromAnys2String("device", "user_id", data)
	ok, di.UserId = ok && uok, db.UserIDFromStringOrNil(user_id)

	dn, uok := getFromAnys2String("device", "dn", data)
	ok, di.Display = ok && uok, dn

	lasttime, uok := getFromAnys2int64("device", "lasttime", data)
	ok, di.LastTime = ok && uok, lasttime

	createtime, uok := getFromAnys2int64("device", "createtime", data)
	ok, di.CreateTime = ok && uok, createtime

	tilogs.L().Debugf("GetDeviceInfo %v %v", ok, *di)

	if ok {
		// 可能这个没有
		di.Name, _ = getFromAnys2String("device", "name", data)
		di.ChannelId, _ = getFromAnys2String("device", "channelid", data)

		return di, nil, false
	} else {
		return nil, XErrDBNotFound, false
	}

}

func (d *DBByDynamoDB) UpdateDeviceInfo(deviceID string, info *DeviceUserInfo) error {
	now_t := time.Now().Unix()
	data := map[string]interface{}{
		"lasttime": now_t,
	}
	return d.db.UpdateByHash(cfg.Cfg.CommonCfg.AuthDeviceTable, deviceID, data)
}

func (d *DBByDynamoDB) SetDeviceInfo(deviceID, name string, uid db.UserID,
	channelID, device string) (*DeviceUserInfo, error) {
	// 当需要用用户名登陆时，需要加入Name信息
	now_t := time.Now().Unix()
	data := map[string]interface{}{
		"user_id":    uid.String(),
		"dn":         "anonymous",
		"lasttime":   now_t,
		"createtime": now_t,
	}
	if name != "" {
		data["name"] = name
	}
	if channelID != "" {
		data["channelid"] = channelID
	}
	if device != "" {
		data["device"] = device
	}
	value := &DeviceUserInfo{
		UserId:     uid,
		Display:    "anonymous",
		Name:       name,
		LastTime:   now_t,
		CreateTime: now_t,
		ChannelId:  channelID,
	}
	err := d.db.SetByHashM(cfg.Cfg.CommonCfg.AuthDeviceTable, deviceID, data)
	return value, err
}

func (d *DBByDynamoDB) SetRecallPlayerInfo(recallId, accountId string, recallNum, gid int64) error {
	data := map[string]interface{}{
		"user_id":     accountId,
		"recall_user": "user:",
		"gid":         gid,
		"recall_num":  recallNum,
	}
	return d.db.SetByHashM_IfNoExist("recallplayer", recallId, data)
}

func (d *DBByDynamoDB) IsNameExist(name string) (int, error) {
	userNameKey := fmt.Sprintf("un:%s", name)
	data, err := d.db.GetByHash(cfg.Cfg.CommonCfg.AuthNameTable, userNameKey)
	if err != nil {
		return -1, err
	}
	if data == nil {
		return 0, nil
	} else {
		return 1, nil
	}
}

//返回逻辑错误：XErrAuthUsernameNotFound
func (d *DBByDynamoDB) GetUnKey(name string, checkGM bool) (
	user db.UserID, error error, isGM bool) {
	// 首先去gm表查
	if checkGM {
		data, err := d.db.GetByHash(cfg.Cfg.CommonCfg.AuthGmTable, name)
		if err == nil {
			uids, ok := data.(string)
			if ok {
				return db.UserIDFromStringOrNil(uids), nil, true
			}
		}
	}

	userNameKey := fmt.Sprintf("un:%s", name)
	data, err := d.db.GetByHash(cfg.Cfg.CommonCfg.AuthNameTable, userNameKey)
	if err != nil {
		return db.InvalidUserID, err, false
	}
	uids, ok := data.(string)
	if !ok {
		return db.InvalidUserID, XErrAuthUsernameNotFound, false
	}
	return db.UserIDFromStringOrNil(uids), nil, false
}

func (d *DBByDynamoDB) GetUnKeyV2(ctx context.Context, name string, checkGM bool) (
	user db.UserID, error error, isGM bool) {
	return d.GetUnKey(name, checkGM)
}

func (d *DBByDynamoDB) SetUnKey(name string, uid db.UserID) error {
	userNameKey := fmt.Sprintf("un:%s", name)
	// 注意db.UserID和int64在type比较时是不一样的
	return d.db.SetByHash(cfg.Cfg.CommonCfg.AuthNameTable, userNameKey, uid.String())
}

func (d *DBByDynamoDB) GetUnInfo(uid db.UserID) (*UserInfo, error) {
	uidkey := makeAuthUidKey(uid)
	res, err := d.db.GetByHashM(cfg.Cfg.CommonCfg.AuthUserInfoTable, uidkey)
	if err != nil {
		return nil, err
	}
	name_any, name_ok := res["name"]
	pass_any, pass_ok := res["pwd"]

	name := ""
	pass := ""
	if name_ok {
		name, _ = name_any.(string)
	}
	if pass_ok {
		pass, _ = pass_any.(string)
	}
	reason := ""
	reason_any, ok := res["banreason"]
	if ok {
		reason, _ = reason_any.(string)
	}
	bantime, _ := getFromAnys2int64("GetUnInfo", "bantime", res)
	gagtime, _ := getFromAnys2int64("GetUnInfo", "gagtime", res)

	deviceID := ""
	deviceID_any, ok := res["device"]
	if ok {
		deviceID, _ = deviceID_any.(string)
	}

	recallID := ""
	recallID_any, ok := res["recall_id"]
	if ok {
		recallID, _ = recallID_any.(string)
	}

	ret := UserInfo{
		Name:     name,
		PassWord: pass,
		BanTime:  bantime,
		GagTime:  gagtime,
		Reason:   reason,
		DeviceID: deviceID,
		RecallID: recallID,
	}
	return &ret, nil
}

func (d *DBByDynamoDB) GetUnInfoV2(ctx context.Context, uid db.UserID) (*UserInfo, error) {
	return d.GetUnInfo(uid)
}

func (d *DBByDynamoDB) UpdateUnInfo(uid db.UserID, deviceID, authToken string) error {
	uidkey := makeAuthUidKey(uid)
	data := make(map[string]interface{}, 4)

	data["authtoken"] = authToken
	if deviceID != "" {
		data["device"] = deviceID
	}
	data["lasttime"] = time.Now().Unix()
	return d.db.UpdateByHash(cfg.Cfg.CommonCfg.AuthUserInfoTable, uidkey, data)
}

func (d *DBByDynamoDB) UpdateUnInfoV2(ctx context.Context, uid db.UserID, deviceID, authToken string) error {
	return d.UpdateUnInfo(uid, deviceID, authToken)
}

func (d *DBByDynamoDB) UpdateBanUn(uid string, time_to_ban int64, reason string) error {
	uidkey := fmt.Sprintf("uid:%s", uid)
	data := make(map[string]interface{}, 4)

	data["bantime"] = time.Now().Unix() + time_to_ban
	data["banreason"] = reason
	return d.db.UpdateByHash(cfg.Cfg.CommonCfg.AuthUserInfoTable, uidkey, data)
}

func (d *DBByDynamoDB) UpdateGagUn(uid string, time_to_gag int64) error {
	uidkey := fmt.Sprintf("uid:%s", uid)
	data := make(map[string]interface{}, 4)

	data["gagtime"] = time.Now().Unix() + time_to_gag
	return d.db.UpdateByHash(cfg.Cfg.CommonCfg.AuthUserInfoTable, uidkey, data)
}

func (d *DBByDynamoDB) SetLastLoginShard(uid string, gidsid string) error {
	uidkey := fmt.Sprintf("uid:%s", uid)
	data := make(map[string]interface{}, 4)

	data["gid_sid"] = gidsid
	return d.db.UpdateByHash(cfg.Cfg.CommonCfg.AuthUserInfoTable, uidkey, data)
}

func (d *DBByDynamoDB) SetPayFeedBackIfNot(uid string) (error, bool) {
	uidkey := fmt.Sprintf("uid:%s", uid)
	m, err := d.db.GetByHashM(cfg.Cfg.CommonCfg.AuthUserInfoTable, uidkey)
	if err != nil {
		return err, false
	}
	pfb, ok := getFromAnys2String("payfeedback", "payfeedback", m)
	if !ok || pfb != "got" {
		if err := d.db.UpdateByHash(cfg.Cfg.CommonCfg.AuthUserInfoTable, uidkey, map[string]interface{}{
			"payfeedback": "got",
		}); err != nil {
			return err, false
		}
		return nil, true
	}
	return nil, false
}

func (d *DBByDynamoDB) GetLastLoginShard(uid string) (string, error) {
	uidkey := fmt.Sprintf("uid:%s", uid)

	res, err := d.db.GetByHashM(cfg.Cfg.CommonCfg.AuthUserInfoTable, uidkey)
	if err != nil {
		return "", err
	}
	sidgid_any, sidgid_ok := res["gid_sid"]
	if !(sidgid_ok) {
		return "", nil
	}
	sidgid, sidgid_ok := sidgid_any.(string)
	if sidgid_ok {
		return sidgid, nil
	} else {
		return "", errors.New("UserInfo sidgid data typ Err")
	}
}

func (d *DBByDynamoDB) SetUnInfo(uid db.UserID,
	name, deviceID, passwd, email, authToken string) error {
	dbpasswd := fmt.Sprintf("%x", secure.DefaultEncode.PasswordForDB(passwd))
	return d.SetUnInfoPass(uid, name, deviceID, dbpasswd, email, authToken)
}

func (d *DBByDynamoDB) SetUnInfoPass(uid db.UserID,
	name, deviceID, dbpasswd, email, authToken string) error {
	now_t := time.Now().Unix()
	data := map[string]interface{}{
		"device":     deviceID,
		"pwd":        dbpasswd,
		"lasttime":   now_t,
		"createtime": now_t,
		"bantime":    0,
		"gagtime":    0,
	}
	if name != "" {
		data["name"] = name
	}
	if email != "" {
		data["email"] = email
	}
	uidkey := makeAuthUidKey(uid)
	return d.db.SetByHashM(cfg.Cfg.CommonCfg.AuthUserInfoTable, uidkey, data)
}

func (d *DBByDynamoDB) GetAuthToken(authToken string) (db.UserID, string, error) {
	_db := authRedisPool.Get()
	defer _db.Close()
	key := makeAuthTokenKey(authToken)
	r, err := redis.String(_db.Do(allmetrics.GetAuthDBStatPrefix("AuthToken", "GET"), "GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return db.InvalidUserID, "", XErrLoginAuthtokenNotFound
		} else {
			return db.InvalidUserID, "", err
		}
	}
	return db.UserIDFromStringOrNil(r), "", nil
}

func (d *DBByDynamoDB) SetAuthToken(authToken string, userID db.UserID, time_out int64, sdkDeviceId string) error {
	_db := authRedisPool.Get()
	defer _db.Close()
	key := makeAuthTokenKey(authToken)
	r, err := redis.String(_db.Do(allmetrics.GetAuthDBStatPrefix("AuthToken", "SET"), "SETEX", key, time_out, userID))
	if err != nil {
		return err
	}

	if r != "OK" {
		return fmt.Errorf("LoginRegAuthToken return is not OK in DB")
	}
	return nil
}

func (d *DBByDynamoDB) GetUserShardInfo(uid string) ([]AuthUserShardInfo, error) {
	info, err := d.db.QueryUserShardInfo(cfg.Cfg.CommonCfg.AuthUserShardInfoTable, uid)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (d *DBByDynamoDB) SetUserShardInfo(uid, gidSidStr string, roleName, roleLevel, mainHero, lastLoginTime, mainlineLevel, headIcon, playerId string) error {
	return d.db.SetUserShardInfo(cfg.Cfg.CommonCfg.AuthUserShardInfoTable, uid, gidSidStr, roleLevel, lastLoginTime)
}

func (mdb *DBByDynamoDB) GetUserHistoryShard(uid string) ([]string, error) {
	return nil, nil
}

func (mdb *DBByDynamoDB) SetUserHistoryShard(uid, shard string) error {
	return nil
}

func (d *DBByDynamoDB) GetRoleInfoBelongToShards(uid string) ([]string, error) {
	return nil, nil
}

func (d *DBByDynamoDB) SetRoleInfoBelongToShards(uid, shard string) error {
	return nil
}

func (d *DBByDynamoDB) GetDB() *AuthDynamoDB {
	return d.db
}

func (d *DBByDynamoDB) AddWhiteList(ip, note string) error {
	return nil
}
func (d *DBByDynamoDB) CheckWhiteList(ip string) (bool, error) {
	return false, nil
}

func (d *DBByDynamoDB) GetSdkDeviceIdByUid(uid string) (string, error) {
	return "", nil
}

func (d *DBByDynamoDB) InsertOrUpdateSdkIdToUid(infos *SdkIdToUidInfos) ([]*SdkIdToUidInfos, error) {

	return nil, nil
}

func (d *DBByDynamoDB) DeleteSdkIdToUid(numberId string, shard uint) ([]*SdkIdToUidInfos, error) {

	return nil, nil
}

func (d *DBByDynamoDB) QuerySdkIdToUid() ([]*SdkIdToUidInfos, error) {

	return nil, nil
}

func (d *DBByDynamoDB) InsertOrUpdateWhiteList(infos *WhiteListInfo) ([]*WhiteListInfo, error) {
	return nil, nil

}
func (d *DBByDynamoDB) DeleteWhiteList(content string, typ uint) ([]*WhiteListInfo, error) {
	return nil, nil

}
func (d *DBByDynamoDB) QueryAllWhiteList() ([]*WhiteListInfo, error) {
	return nil, nil
}

func (d *DBByDynamoDB) QueryWhiteList(typ uint, param string) ([]*WhiteListInfo, error) {
	return nil, nil
}

func (d *DBByDynamoDB) QueryUserSpecialAward(uid db.UserID) (string, error) {
	return "", nil
}
func (d *DBByDynamoDB) UpdateUserSpecialAward(uid db.UserID, awardStr string) error {
	return nil
}

func (d *DBByDynamoDB) RewardForUser(rewardType int, userID db.UserID, optType string) (map[int]*config_data.RewardInfo, int32) {
	return nil, 0
}

//更新封禁角色信息
func (d *DBByDynamoDB) UpdateBanPlayer(uid string, acid string, banTime int64, reason string) error {
	return nil
}

func (d *DBByDynamoDB) GetPlayerBanInfo(uid string, acid string) *BanPlayerInfo {
	return nil
}
