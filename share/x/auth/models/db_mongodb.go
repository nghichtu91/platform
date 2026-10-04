package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/auth/config_data"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// MongoAuth 通用的mongo Device表结构
type MongoAuth struct {
	DeviceID        string                   `bson:"id,omitempty"`      // 主键 + 索引
	UserID          string                   `bson:"user_id,omitempty"` // 索引2
	DisplayName     string                   `bson:"dn,omitempty"`
	LastGidSid      string                   `bson:"gid_sid,omitempty"`
	ChannelID       string                   `bson:"channelid,omitempty"`
	Device          string                   `bson:"device,omitempty"`
	LastAuthTime    int64                    `bson:"lasttime,omitempty"`
	CreateTime      int64                    `bson:"createtime,omitempty"`
	BanTime         int64                    `bson:"bantime,omitempty"`
	BanReason       string                   `bson:"banreason,omitempty"`
	GagTime         int64                    `bson:"gagtime,omitempty"`
	NameAuth        string                   `bson:"name,omitempty"` // 索引3
	NameAuthPwd     string                   `bson:"pwd,omitempty"`
	AuthToken       string                   `bson:"authtoken,omitempty"`
	Payfeeback      string                   `bson:"payfeedback,omitempty"`
	SpecialAward    string                   `bson:"special_award,omitempty"`     // 特殊活动已领奖列表
	PayReward       bool                     `bson:"pay_reward,omitempty"`        // 是否已领取过充值返钻
	LoginRankReward bool                     `bson:"login_rank_reward,omitempty"` // 是否已领取过登录排名返利。
	LoginRewardTime int64                    `bson:"login_reward_time,omitempty"` // 登录排名返利奖励领取时间
	BanPlayerInfos  map[string]BanPlayerInfo `bson:"ban_player_infos,omitempty"`  // 封禁的角色列表
	Version         int                      `bson:"version,omitempty"`           // 版本号，每次写加1，确保存档原子性
}

// 角色封禁信息
type BanPlayerInfo struct {
	UserID     string `bson:"user_id"`      // uid
	Acid       string `bson:"acid"`         // acid
	BanEndTime int64  `bson:"ban_end_time"` // 封禁的结束时间
	BanResult  string `bson:"ban_result"`   // 封禁的原因
}

func (ma *MongoAuth) GetDeviceUserInfo() DeviceUserInfo {
	return DeviceUserInfo{
		UserId:     db.UserIDFromStringOrNil(ma.UserID),
		Display:    ma.DisplayName,
		LastTime:   ma.LastAuthTime,
		CreateTime: ma.CreateTime,
		Name:       ma.NameAuth,
		ChannelId:  ma.ChannelID,
	}
}

func (ma *MongoAuth) GetDeviceUserInfoPtr() *DeviceUserInfo {
	return &DeviceUserInfo{
		UserId:     db.UserIDFromStringOrNil(ma.UserID),
		Display:    ma.DisplayName,
		LastTime:   ma.LastAuthTime,
		CreateTime: ma.CreateTime,
		Name:       ma.NameAuth,
		ChannelId:  ma.ChannelID,
	}
}

func (ma *MongoAuth) GetUserInfo() *UserInfo {
	return &UserInfo{
		Name:     ma.NameAuth,
		PassWord: ma.NameAuthPwd,
		BanTime:  ma.BanTime,
		GagTime:  ma.GagTime,
		Reason:   ma.BanReason,
		DeviceID: ma.DeviceID,
	}
}

type MongoUserShardInfo struct {
	UserID        string `bson:"user_id,omitempty"`
	GidSid        string `bson:"gid_sid,omitempty"`
	RoleName      string `bson:"role_name,omitempty"`
	RoleLevel     string `bson:"role_level,omitempty"`
	MainHero      string `bson:"main_hero,omitempty"`
	LastLoginTime string `bson:"lastLoginTime,omitempty"`
	HistoryShard  string `bson:"history_shard,omitempty"`
	MainlineLevel string `bson:"mainline_level,omitempty"`
}

type MongoUserShardInfoV2 struct {
	UserID string `bson:"user_id,omitempty"` // 主键 + 索引
	AuthUserShardInfo
}

type MongoTokens struct {
	UserID      string    `bson:"user_id,omitempty"` // 主键 + 索引
	AuthToken   string    `bson:"authtoken,omitempty"`
	CreateAt    time.Time `bson:"create_at,omitempty"`
	SdkDeviceId string    `bson:"sdk_device_id,omitempty"`
}

type SdkIdToUidInfos struct {
	SdkId                   string `bson:"sdk_id,omitempty"`
	Uid                     string `bson:"uid,omitempty"`
	CurrentPlayerNumber     string `bson:"current_player_number,omitempty"`
	CurrentPlayerNumberDesc string `bson:"current_player_number_desc,omitempty"`
	TargetPlayerNumber      string `bson:"target_player_number,omitempty"`
	TargetPlayerNumberDesc  string `bson:"target_player_number_desc,omitempty"`
	ShardId                 uint   `bson:"sid,omitempty"`
}

type WhiteListInfo struct {
	Typ        uint   `bson:"typ,omitempty"`
	Content    string `bson:"content,omitempty"`
	ModifyTime int64  `bson:"modify_time,omitempty"`
	SdkId      string `bson:"sdk_id,omitempty"`
}

type MongoHistoryShard struct {
	UserId  string `bson:"user_id,omitempty"`
	ShardId string `bson:"shard_id,omitempty"`
}

const (
	MongoCollectionDevices        = "Devices"
	MongoCollectionTokens         = "UserInfos"
	MongoCollectionUserShardInfos = "UserShardInfos"
	MongoCollectionSdkIdToUid     = "SdkIdToUid"
	MongoCollectionWhiteList      = "WhiteList"
	MongoCollectionHistoryShard   = "HistoryShard"
)

// DBByMongoDB mgo实现的auth db
// Deprecated
// 由于mgo的性能问题，替换为mongo官方库
type DBByMongoDB struct {
	*mgo.Session
	DBName string
}

func (mdb *DBByMongoDB) QueryRecallIdByAcid(accountId string) (map[string]interface{}, error) {
	return nil, nil
}
func (mdb *DBByMongoDB) UpdateRecallID(uid db.UserID, recallID string) error {
	return nil
}

func (mdb *DBByMongoDB) SetRecallPlayerInfo(recallId, uid string, recallNum, gid int64) error {
	return nil
}

func (mdb *DBByMongoDB) MarketGetByHashM(recallId string) (bool, error) {
	return false, nil
}

func (mdb *DBByMongoDB) GetDB() *mgo.Database {
	return mdb.DB(mdb.DBName)
}

func (mdb *DBByMongoDB) GetMongo() *DBByMongoDB {
	return &DBByMongoDB{
		Session: mdb.Copy(),
		DBName:  mdb.DBName,
	}
}

// GetMongoClone 使用Clone方式获取session，复用之前的socket，比Copy省资源
// 但是如果原session出现问题，Clone出的session也不可用
func (mdb *DBByMongoDB) GetMongoClone() *DBByMongoDB {
	return &DBByMongoDB{
		Session: mdb.Clone(),
		DBName:  mdb.DBName,
	}
}

func (mdb *DBByMongoDB) Init(config DBConfig) error {
	tilogs.L().Infof("DBByMongoDB Init %v", config)
	session, err := mgo.DialWithTimeout(config.MongoDBUrl, 5*time.Second)
	if err != nil {
		return err
	}
	mdb.Session = session
	mdb.DBName = config.MongoDBName

	session.SetMode(mgo.Strong, true)
	index := mgo.Index{
		Key:        []string{"id"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	c := mdb.GetDB().C(MongoCollectionDevices)
	if err := c.EnsureIndex(index); err != nil {
		return err
	}

	index2 := mgo.Index{
		Key:        []string{"user_id"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	c2 := mdb.GetDB().C(MongoCollectionDevices)
	if err := c2.EnsureIndex(index2); err != nil {
		return err
	}

	// Tokens
	{
		index := mgo.Index{
			Key:         []string{"create_at"},
			ExpireAfter: time.Second * AUTHTOKEN_TIMEOUT,
		}
		ct := mdb.GetDB().C(MongoCollectionTokens)
		if err := ct.EnsureIndex(index); err != nil {
			return err
		}

		index2 := mgo.Index{
			Key:        []string{"user_id"},
			Unique:     true,
			DropDups:   true,
			Background: true,
			Sparse:     true,
		}
		ct2 := mdb.GetDB().C(MongoCollectionTokens)
		if err := ct2.EnsureIndex(index2); err != nil {
			return err
		}
	}

	indexName := mgo.Index{
		Key:        []string{"name"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	cName := mdb.GetDB().C(MongoCollectionDevices)
	if err := cName.EnsureIndex(indexName); err != nil {
		return err
	}

	index3 := mgo.Index{
		Key:        []string{"user_id"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	c3 := mdb.GetDB().C(MongoCollectionUserShardInfos)
	if err := c3.EnsureIndex(index3); err != nil {
		return err
	}

	index4 := mgo.Index{
		Key:        []string{"user_id", "shard_id"},
		Background: true,
		Sparse:     true,
	}
	c4 := mdb.GetDB().C(MongoCollectionHistoryShard)
	if err := c4.EnsureIndex(index4); err != nil {
		return err
	}
	return nil
}

func (mdb *DBByMongoDB) UpdateDeviceCollection(selector interface{}, update interface{}) error {

	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)

	query := c.Find(selector)
	if n, err := query.Count(); err != nil {
		return err
	} else if n != 1 {
		return fmt.Errorf("DBByMongoDB.UpdateDeviceCollection Find%v, %d account.", selector, n)
	}
	var ma MongoAuth
	if err := query.One(&ma); err != nil {
		return err
	}
	if err := c.Update(selector, update); err != nil {
		return err
	}

	return nil

}

// GetDeviceInfo 根据DeviceID查找是否存在已注册信息
func (mdb *DBByMongoDB) GetDeviceInfoClone(deviceID string, checkGM bool) (*DeviceUserInfo, error, bool) {
	s := mdb.GetMongoClone()
	defer s.Close()

	// 对checkGM做判断
	var gmUid string
	if checkGM {
		c := s.GetDB().C(MongoCollectionSdkIdToUid)
		var sdkIdToUidInfo SdkIdToUidInfos
		err := c.Find(bson.D{{Name: "sdk_id", Value: deviceID}}).One(&sdkIdToUidInfo)
		if err == nil {
			gmUid = sdkIdToUidInfo.Uid
		}
	}

	c := s.GetDB().C(MongoCollectionDevices)
	var ma MongoAuth
	err := c.Find(bson.D{{Name: "id", Value: deviceID}}).One(&ma)
	switch err {
	case mgo.ErrNotFound:
		// tilogs.L().Debugf("DBByMongoDB.GetDeviceInfo not found devicdid")
		return nil, XErrDBNotFound, false
	case nil:
		if gmUid != "" {
			tilogs.L().Errorf("成功替换uid, old uid is %s,new uid is %s", ma.UserID, gmUid)
			ma.UserID = gmUid
			di := ma.GetDeviceUserInfo()
			return &di, nil, true
		}
		di := ma.GetDeviceUserInfo()
		// tilogs.L().Debugf("DBByMongoDB.GetDeviceInfo %v", di)
		return &di, nil, false
	}

	return nil, nil, false
}

// GetDeviceInfo 根据DeviceID查找是否存在已注册信息
func (mdb *DBByMongoDB) GetDeviceInfo(deviceID string, checkGM bool) (*DeviceUserInfo, error, bool) {
	s := mdb.GetMongo()
	defer s.Close()

	// 对checkGM做判断
	var gmUid string
	if checkGM {
		c := s.GetDB().C(MongoCollectionSdkIdToUid)
		var sdkIdToUidInfo SdkIdToUidInfos
		err := c.Find(bson.D{{Name: "sdk_id", Value: deviceID}}).One(&sdkIdToUidInfo)
		if err == nil {
			gmUid = sdkIdToUidInfo.Uid
		}
	}

	c := s.GetDB().C(MongoCollectionDevices)
	var ma MongoAuth
	err := c.Find(bson.D{{Name: "id", Value: deviceID}}).One(&ma)
	switch err {
	case mgo.ErrNotFound:
		// tilogs.L().Debugf("DBByMongoDB.GetDeviceInfo not found devicdid")
		return nil, XErrDBNotFound, false
	case nil:
		if gmUid != "" {
			tilogs.L().Errorf("成功替换uid, old uid is %s,new uid is %s", ma.UserID, gmUid)
			ma.UserID = gmUid
			di := ma.GetDeviceUserInfo()
			return &di, nil, true
		}
		di := ma.GetDeviceUserInfo()
		// tilogs.L().Debugf("DBByMongoDB.GetDeviceInfo %v", di)
		return &di, nil, false
	}

	return nil, nil, false
}

// UpdateDeviceInfo 来更新之后的访问时间
func (mdb *DBByMongoDB) UpdateDeviceInfo(deviceID string, info *DeviceUserInfo) error {
	now_t := time.Now().Unix()
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)
	_, err := c.Upsert(bson.D{{Name: "id", Value: deviceID}}, bson.M{"$set": bson.D{{Name: "lasttime", Value: now_t}}})

	return err
}

// SetDeviceInfo
func (mdb *DBByMongoDB) SetDeviceInfo(deviceID, name string, uid db.UserID,
	channelID, device string) (*DeviceUserInfo, error) {
	now_t := time.Now().Unix()
	ma := MongoAuth{
		UserID:       uid.String(),
		DisplayName:  "anonymous",
		LastAuthTime: now_t,
		CreateTime:   now_t,
		NameAuth:     name,
		ChannelID:    channelID,
		Device:       device,
	}

	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)
	_, err := c.Upsert(bson.D{{Name: "id", Value: deviceID}}, bson.M{"$set": ma})
	switch err {
	case nil:
		dui := ma.GetDeviceUserInfo()
		tilogs.L().Debugf("DBByMongoDB.SetDeviceInfo %v", ma.GetDeviceUserInfo())
		return &dui, nil
	}
	return nil, nil
}

// IsNameExist 判断玩家是否使用了用户名模式
func (mdb *DBByMongoDB) IsNameExist(name string) (int, error) {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)
	n, err := c.Find(bson.D{{Name: "name", Value: name}}).Count()
	if err != nil {
		return -1, nil
	}
	switch err {
	case mgo.ErrNotFound:
		return 0, nil
	case nil:
		// do nothing
	default:
		return -1, err
	}

	if n == 0 {
		return 0, nil
	}

	return 1, nil
}

// 返回逻辑错误：XErrAuthUsernameNotFound todo 返回true，绕过正常检查
func (mdb *DBByMongoDB) GetUnKey(name string, checkGM bool) (db.UserID, error, bool) {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)
	var ma MongoAuth
	err := c.Find(bson.D{{Name: "name", Value: name}}).One(&ma)
	switch err {
	case mgo.ErrNotFound:
		return db.InvalidUserID, XErrAuthUsernameNotFound, false
	case nil:
		// 对checkGM做判断
		if checkGM {
			c := s.GetDB().C(MongoCollectionSdkIdToUid)
			var sdkIdToUidInfo SdkIdToUidInfos
			err := c.Find(bson.D{{Name: "sdk_id", Value: ma.DeviceID}}).One(&sdkIdToUidInfo)
			if err == nil {
				tilogs.L().Infof("成功替换uid, old uid is %s,new uid is %s", ma.UserID, sdkIdToUidInfo.Uid)
				return db.UserIDFromStringOrNil(sdkIdToUidInfo.Uid), nil, true
			}
		}
		return db.UserIDFromStringOrNil(ma.UserID), nil, false
	}
	return db.InvalidUserID, nil, false
}

func (mdb *DBByMongoDB) GetUnKeyV2(ctx context.Context, name string, checkGM bool) (db.UserID, error, bool) {
	return mdb.GetUnKey(name, checkGM)
}

func (mdb *DBByMongoDB) SetUnKey(name string, uid db.UserID) error {
	return mdb.UpdateDeviceCollection(bson.D{{Name: "user_id", Value: uid.String()}},
		bson.M{"$set": bson.D{{Name: "name", Value: name}}})
}

// GetUnInfo
// return value: name, password, bantime, gagtime, error
func (mdb *DBByMongoDB) GetUnInfo(uid db.UserID) (*UserInfo, error) {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)
	var ma MongoAuth
	err := c.Find(bson.D{{Name: "user_id", Value: uid.String()}}).One(&ma)
	if err != nil {
		return nil, err
	}
	ret := UserInfo{
		Name:     ma.NameAuth,
		PassWord: ma.NameAuthPwd,
		BanTime:  ma.BanTime,
		GagTime:  ma.GagTime,
		Reason:   ma.BanReason,
		DeviceID: ma.DeviceID,
	}
	return &ret, nil
}

func (mdb *DBByMongoDB) GetUnInfoV2(ctx context.Context, uid db.UserID) (*UserInfo, error) {
	return mdb.GetUnInfo(uid)
}

// UpdateUnInfo 目前用在更新AuthToken
// XXX by YZH UserInfoTable
func (mdb *DBByMongoDB) UpdateUnInfo(uid db.UserID, deviceID, authToken string) error {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)
	if deviceID != "" {
		c.Upsert(bson.D{{Name: "id", Value: deviceID}},
			bson.M{"$set": MongoAuth{
				AuthToken: authToken,
			}})
	} else if uid.IsValid() {
		c.Upsert(bson.D{{Name: "id", Value: deviceID}},
			bson.M{"$set": MongoAuth{
				AuthToken: authToken,
			}})
	}
	return nil
}

func (mdb *DBByMongoDB) UpdateUnInfoV2(ctx context.Context, uid db.UserID, deviceID, authToken string) error {
	return mdb.UpdateUnInfo(uid, deviceID, authToken)
}

func (mdb *DBByMongoDB) UpdateBanUn(uid string, time_to_ban int64, reason string) error {
	bantime := time.Now().Unix() + time_to_ban
	return mdb.UpdateDeviceCollection(bson.D{{Name: "user_id", Value: uid}},
		bson.M{"$set": MongoAuth{BanTime: bantime, BanReason: reason}})
}

func (mdb *DBByMongoDB) UpdateGagUn(uid string, time_to_gag int64) error {
	gagtime := time.Now().Unix() + time_to_gag
	return mdb.UpdateDeviceCollection(bson.D{{Name: "user_id", Value: uid}},
		bson.M{"$set": MongoAuth{GagTime: gagtime}})

}

func (mdb *DBByMongoDB) SetUnInfo(uid db.UserID, name, deviceID, passwd, email, authToken string) error {
	dbpasswd := fmt.Sprintf("%x", secure.DefaultEncode.PasswordForDB(passwd))
	return mdb.SetUnInfoPass(uid, name, deviceID, dbpasswd, email, authToken)
}

// SetUnInfoPass GMTools中，用来乾坤大挪移。用来把一个accountid映射到一个有用户名密码的帐号下。方便查找问题。
func (mdb *DBByMongoDB) SetUnInfoPass(uid db.UserID, name, deviceID, dbpasswd, email, authToken string) error {
	now_t := time.Now().Unix()
	// FIXME by YZH UpdateDeviceCollection 使用了双索引,两个独立索引,能起到加速索引作用吗?
	// FIXME by YZH 这里应该是insert而不应该是Update

	return mdb.UpdateDeviceCollection(bson.D{{Name: "id", Value: deviceID}, {Name: "user_id", Value: uid.String()}},
		bson.M{"$set": MongoAuth{
			NameAuth:     name,
			NameAuthPwd:  dbpasswd,
			LastAuthTime: now_t,
			// ModifyTime: //now_t, 因为MongoDB使用一个Collection就能够处理原来的多个表,因此这个创建时间不应该更新了。
			BanTime: 0,
			GagTime: 0,
		}})
}

func (mdb *DBByMongoDB) GetAuthToken(authToken string) (db.UserID, string, error) {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionTokens)
	var mt MongoTokens
	err := c.Find(MongoTokens{AuthToken: authToken}).One(&mt)
	switch err {
	case mgo.ErrNotFound:
		return db.InvalidUserID, "", XErrLoginAuthtokenNotFound
	case nil:
		// do nothing
	default:
		return db.InvalidUserID, "", err
	}
	return db.UserIDFromStringOrNil(mt.UserID), mt.SdkDeviceId, nil
}

func (mdb *DBByMongoDB) SetAuthToken(authToken string, userID db.UserID, time_out int64, sdkDeviceId string) error {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionTokens)
	if _, err := c.Upsert(MongoTokens{UserID: userID.String()}, MongoTokens{
		UserID:      userID.String(),
		AuthToken:   authToken,
		CreateAt:    time.Now(),
		SdkDeviceId: sdkDeviceId,
	}); err != nil {
		return err
	}
	return nil
}

func (mdb *DBByMongoDB) GetUserShardInfo(uid string) ([]AuthUserShardInfo, error) {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionUserShardInfos)
	var result []AuthUserShardInfo
	err := c.Find(MongoUserShardInfo{UserID: uid}).All(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (mdb *DBByMongoDB) SetUserShardInfo(uid, gidSidStr string, roleName, roleLevel, mainHero, lastLoginTime, mainlineLevel, headIcon, playerId string) error {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionUserShardInfos)
	_, err := c.Upsert(MongoUserShardInfo{UserID: uid},
		bson.M{"$set": MongoUserShardInfo{
			UserID:        uid,
			GidSid:        gidSidStr,
			RoleName:      roleName,
			RoleLevel:     roleLevel,
			MainHero:      mainHero,
			LastLoginTime: lastLoginTime,
			MainlineLevel: mainlineLevel,
		}})
	return err
}

func (mdb *DBByMongoDB) GetUserHistoryShard(uid string) ([]string, error) {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionHistoryShard)
	var result []MongoHistoryShard
	err := c.Find(bson.M{"user_id": uid}).All(&result)
	//	err := c.Find(MongoHistoryShard{UserID: uid}).All(&result)
	if err != nil {
		return nil, err
	}

	historyShard := make([]string, 0, len(result))
	for _, res := range result {
		historyShard = append(historyShard, res.ShardId)
	}

	return historyShard, nil
}

func (mdb *DBByMongoDB) SetUserHistoryShard(uid, shard string) error {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionHistoryShard)
	_, err := c.Upsert(MongoHistoryShard{UserId: uid, ShardId: shard},
		bson.M{"$set": MongoHistoryShard{
			UserId:  uid,
			ShardId: shard,
		}})
	return err
}

func (mdb *DBByMongoDB) SetLastLoginShard(uid string, gidsid string) error {
	return mdb.UpdateDeviceCollection(bson.D{{Name: "user_id", Value: uid}},
		bson.M{"$set": bson.D{{Name: "gid_sid", Value: gidsid}}})

}

func (mdb *DBByMongoDB) GetLastLoginShard(uid string) (string, error) {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)

	query := c.Find(bson.D{{Name: "user_id", Value: uid}})
	if n, err := query.Count(); err != nil {
		return "", err
	} else if n != 1 {
		return "", errors.New("DBByMongoDB.GetLastLoginShard Find more than one account.")
	}
	var ma MongoAuth
	if err := query.One(&ma); err != nil {
		return "", err
	}

	if ma.LastGidSid == "" {
		return "", nil
	}

	return ma.LastGidSid, nil
}

func (mdb *DBByMongoDB) SetPayFeedBackIfNot(uid string) (error, bool) {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)

	query := c.Find(bson.D{{Name: "user_id", Value: uid}})
	if n, err := query.Count(); err != nil {
		return err, false
	} else if n != 1 {
		return errors.New("DBByMongoDB.GetLastLoginShard Find more than one account."), false
	}
	var ma MongoAuth
	if err := query.One(&ma); err != nil {
		return err, false
	}

	if ma.Payfeeback != "got" {
		if err := c.Update(bson.D{{Name: "user_id", Value: uid}},
			bson.M{"$set": bson.D{{Name: "payfeedback", Value: "got"}}}); err != nil {
			return err, false
		}
		return nil, true
	}
	return nil, false
}

func (d *DBByMongoDB) AddWhiteList(ip, note string) error {
	return nil
}
func (d *DBByMongoDB) CheckWhiteList(ip string) (bool, error) {
	return false, nil
}

func (mdb *DBByMongoDB) GetSdkDeviceIdByUid(uid string) (string, error) {
	s := mdb.GetMongo()
	defer s.Close()
	var mt MongoTokens
	c := s.GetDB().C(MongoCollectionTokens)
	err := c.Find(bson.M{"user_id": uid}).One(&mt)
	if err != nil {
		tilogs.L().Errorf("get SdkDeviceId from UserInfo fail,err is %v", err)
	}
	return mt.SdkDeviceId, nil
}

func (mdb *DBByMongoDB) InsertOrUpdateSdkIdToUid(infos *SdkIdToUidInfos) ([]*SdkIdToUidInfos, error) {

	s := mdb.GetMongo()
	defer s.Close()
	var mt []*SdkIdToUidInfos
	c := s.GetDB().C(MongoCollectionSdkIdToUid)

	_, err := c.Upsert(bson.M{"current_player_number": infos.CurrentPlayerNumber},
		bson.M{"$set": bson.M{"sdk_id": &infos.SdkId,
			"uid":                   &infos.Uid,
			"target_player_number":  &infos.TargetPlayerNumber,
			"current_player_number": &infos.CurrentPlayerNumber}})
	/*c.Remove(bson.M{"current_player_number": infos.CurrentPlayerNumber})
	err := c.Insert(&infos)*/
	if err != nil {
		tilogs.L().Errorf("Insert fail , sdkIdToUidInfos value : %v", infos)
	}
	c.Find(nil).All(&mt)
	tilogs.L().Debugf("FIND SUCCESS")
	return mt, nil
}

func (mdb *DBByMongoDB) DeleteSdkIdToUid(numberId string, shard uint) ([]*SdkIdToUidInfos, error) {
	s := mdb.GetMongo()
	defer s.Close()
	var mt []*SdkIdToUidInfos
	c := s.GetDB().C(MongoCollectionSdkIdToUid)
	c.Remove(bson.M{"current_player_number": numberId})

	c.Find(nil).All(&mt)
	return mt, nil
}

func (mdb *DBByMongoDB) QuerySdkIdToUid() ([]*SdkIdToUidInfos, error) {

	s := mdb.GetMongo()
	defer s.Close()
	var mt []*SdkIdToUidInfos
	c := s.GetDB().C(MongoCollectionSdkIdToUid)
	c.Find(nil).All(&mt)
	return mt, nil
}

func (mdb *DBByMongoDB) InsertOrUpdateWhiteList(info *WhiteListInfo) ([]*WhiteListInfo, error) {
	s := mdb.GetMongo()
	defer s.Close()
	var mt []*WhiteListInfo
	c := s.GetDB().C(MongoCollectionWhiteList)
	c.Remove(bson.M{"typ": info.Typ, "content": info.Content})
	err := c.Insert(&info)
	if err != nil {

	}
	c.Find(nil).All(&mt)
	tilogs.L().Debugf("FIND SUCCESS")
	return mt, nil

}
func (mdb *DBByMongoDB) DeleteWhiteList(content string, typ uint) ([]*WhiteListInfo, error) {
	s := mdb.GetMongo()
	defer s.Close()
	var mt []*WhiteListInfo
	c := s.GetDB().C(MongoCollectionWhiteList)
	c.Remove(bson.M{"typ": typ, "content": content})

	c.Find(nil).All(&mt)
	return mt, nil
}

func (mdb *DBByMongoDB) QueryAllWhiteList() ([]*WhiteListInfo, error) {
	s := mdb.GetMongo()
	defer s.Close()
	var mt []*WhiteListInfo
	c := s.GetDB().C(MongoCollectionWhiteList)
	c.Find(nil).All(&mt)
	return mt, nil
}

func (mdb *DBByMongoDB) QueryWhiteList(typ uint, param string) ([]*WhiteListInfo, error) {
	s := mdb.GetMongo()
	defer s.Close()
	var mt []*WhiteListInfo
	var err error
	c := s.GetDB().C(MongoCollectionWhiteList)
	if typ == 3 {
		err = c.Find(bson.M{"typ": typ, "sdk_id": param}).All(&mt)
	} else {
		err = c.Find(bson.M{"typ": typ, "content": param}).All(&mt)
	}

	return mt, err
}

// QueryUserSpecialAward 获取该账号下已领取过的特殊活动的奖励信息
func (mdb *DBByMongoDB) QueryUserSpecialAward(uid db.UserID) (string, error) {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)
	var ma MongoAuth
	err := c.Find(bson.D{{Name: "user_id", Value: uid.String()}}).One(&ma)
	if err != nil {
		return "", err
	}
	return ma.SpecialAward, nil
}

// UpdateUserSpecialAward 更新该账号下已领取过的特殊活动的奖励信息
func (mdb *DBByMongoDB) UpdateUserSpecialAward(uid db.UserID, awardStr string) error {
	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)
	var ma MongoAuth
	err := c.Find(bson.D{{Name: "user_id", Value: uid.String()}}).One(&ma)
	if err != nil {
		return err
	}
	_, err = c.Upsert(bson.D{{Name: "id", Value: ma.DeviceID}}, bson.M{"$set": bson.D{{Name: "special_award", Value: awardStr}}})
	return err
}

// 充值返钻
func (mdb *DBByMongoDB) PayRewardForUser(uid db.UserID, opt string) (int32, bool, int32) {
	const (
		PayRewardCode_OK    = 1 // 成功
		PayRewardCode_ERROR = 2 // 异常
		PayRewardCode_Award = 3 // 奖励已领取，不能重复领取
	)

	s := mdb.GetMongo()
	defer s.Close()
	c := s.GetDB().C(MongoCollectionDevices)
	var ma MongoAuth
	err := c.Find(bson.D{{Name: "user_id", Value: uid.String()}}).One(&ma)
	if err != nil {
		tilogs.L().Errorf("RewardForUser,mongoDB Find err,uid:%s", uid.String())
		return 0, false, PayRewardCode_ERROR
	}
	// 获取充值列表，判断uid是否在返钻的列表中
	payRewardCfg := config_data.GetPayRewardConfig(uid.String())
	if payRewardCfg == nil {
		tilogs.L().Infof("RewardForUser,has no award,uid:%s", uid.String())
		return 0, false, PayRewardCode_OK
	}
	if opt == "reward" {
		// 检测返钻奖励是否已领取过
		if ma.PayReward {
			tilogs.L().Infof("RewardForUser,already award,uid:%s", uid.String())
			return payRewardCfg.MoneyScore, ma.PayReward, PayRewardCode_Award
		}
		_, err = c.Upsert(bson.D{{Name: "id", Value: ma.DeviceID}}, bson.M{"$set": bson.D{{Name: "pay_reward", Value: ma.PayReward}}})
		if err != nil {
			tilogs.L().Errorf("RewardForUser,mongoDB Upsert err,uid:%s", uid.String())
			return payRewardCfg.MoneyScore, ma.PayReward, PayRewardCode_ERROR
		}
		ma.PayReward = true
	}
	return payRewardCfg.MoneyScore, ma.PayReward, PayRewardCode_OK
}
