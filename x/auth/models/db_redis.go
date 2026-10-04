// +build authredis

package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/nghichtu91/platform/share/planx/dynamodb"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"

	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/servers/db"
)

func (d *DBByRedis) Init(config DBConfig) error {
	return nil
}

// GetDeviceInfo
// 返回数据库级错误，或者XErrDBNotFound
func (d *DBByRedis) GetDeviceInfo(deviceID string) (*DeviceUserInfo, error) {
	_db := authRedisPool.Get()
	defer _db.Close()
	reply, err := _db.Do(allmetrics.GetAuthDBStatPrefix("DeviceInfo", "GET"), "GET", deviceID)
	if err != nil {
		return nil, err
	}

	if reply != nil {
		deviceinfo, err := redis.Bytes(reply, err)
		if err != nil {
			tilogs.L().Errorf(err.Error())
			return nil, err
		}
		var di DeviceUserInfo
		if err := json.Unmarshal(deviceinfo, &di); err != nil {
			tilogs.L().Errorf(err.Error())
			return nil, err
		}
		//已经存在 直接返回
		return &di, nil
	}

	return nil, XErrDBNotFound
}

func (d *DBByRedis) UpdateDeviceInfo(deviceID string, info *DeviceUserInfo) error {
	_db := authRedisPool.Get()
	defer _db.Close()

	// 当需要用用户名登陆时，需要加入Name信息
	now_t := time.Now().Unix()
	value := *info
	value.LastTime = now_t

	deviceinfo, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	_, err = _db.Do(allmetrics.GetAuthDBStatPrefix("DeviceInfo", "SET"), "SET", deviceID, deviceinfo)
	return err
}

func (d *DBByRedis) SetDeviceInfo(deviceID, name string, uid db.UserID) (*DeviceUserInfo, error) {
	_db := authRedisPool.Get()
	defer _db.Close()
	now_t := time.Now().Unix()
	// 当需要用用户名登陆时，需要加入Name信息
	value := DeviceUserInfo{
		UserId:     uid,
		Display:    "anonymous",
		Name:       name,
		LastTime:   now_t,
		CreateTime: now_t,
	}

	deviceinfo, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	_, err = _db.Do(allmetrics.GetAuthDBStatPrefix("DeviceInfo", "SET"), "SET", deviceID, deviceinfo)
	return &value, err
}

func (d *DBByRedis) IsNameExist(name string) (int, error) {
	_db := authRedisPool.Get()
	defer _db.Close()
	userNameKey := fmt.Sprintf("un:%s", name)
	return redis.Int(_db.Do(allmetrics.GetAuthDBStatPrefix("un", "EXISTS"), "EXISTS", userNameKey))
}

//返回逻辑错误：XErrAuthUsernameNotFound
func (d *DBByRedis) GetUnKey(name string) (db.UserID, error) {
	_db := authRedisPool.Get()
	defer _db.Close()

	unkey := fmt.Sprintf("un:%s", name)
	uid, err := redis.String(_db.Do(allmetrics.GetAuthDBStatPrefix("un", "GET"), "GET", unkey))
	if err != nil {
		return db.InvalidUserID, XErrAuthUsernameNotFound
	}
	return db.UserIDFromStringOrNil(uid), nil
}

func (d *DBByRedis) SetUnKey(name string, uid db.UserID) error {
	_db := authRedisPool.Get()
	defer _db.Close()

	unkey := fmt.Sprintf("un:%s", name)
	_, err := _db.Do(allmetrics.GetAuthDBStatPrefix("un", "SET"), "SET", unkey, uid)
	return err
}

func (d *DBByRedis) GetUnInfo(uid db.UserID) (string, string, int64, int64, string, error) {
	_db := authRedisPool.Get()
	defer _db.Close()

	var v struct {
		Name      string `redis:"name"`
		Passwd    string `redis:"pwd"`
		Bantime   string `redis:"bantime"`
		Gagtime   string `redis:"gagtime"`
		BanReason string `redis:"banreason"`
	}

	uidkey := makeAuthUidKey(uid)
	reply, err := redis.Values(_db.Do(allmetrics.GetAuthDBStatPrefix("AuthUid", "HMGET"), "HMGET", uidkey, "name", "pwd", "bantime", "gagtime", "banreason"))
	_, err = redis.Scan(reply, &v.Name, &v.Passwd, &v.Bantime, &v.Gagtime)
	if err != nil {
		return "", "", 0, 0, err
	}
	return v.Name, v.Passwd, v.Bantime, v.Gagtime, nil
}

func (d *DBByRedis) UpdateUnInfo(uid string, deviceID, authToken string) error {
	_db := authRedisPool.Get()
	defer _db.Close()
	now_t := time.Now().Unix()
	uidkey := fmt.Sprintf("uid:%s", uid)
	args := redis.Args{}.Add(uidkey)
	args = args.Add("authtoken").Add(authToken)
	args = args.Add("lasttime").Add(now_t)

	if deviceID != "" {
		args = args.Add("device").Add(deviceID)
	}

	_, err := redis.String(_db.Do(allmetrics.GetAuthDBStatPrefix("AuthUid", "HMSET"), "HMSET", args...))
	return err
}

func (d *DBByRedis) UpdateBanUn(uid db.UserID, time_to_ban int64, reason string) error {
	_db := authRedisPool.Get()
	defer _db.Close()
	now_t := time.Now().Unix()
	uidkey := makeAuthUidKey(uid)
	args := redis.Args{}.Add(uidkey)
	args = args.Add("bantime").Add(now_t + time_to_ban)
	args = args.Add("banreason").Add(reason)
	_, err := redis.String(_db.Do(allmetrics.GetAuthDBStatPrefix("AuthUid", "HMSET"), "HMSET", args...))
	return err
}

func (d *DBByRedis) UpdateGagUn(uid db.UserID, time_to_gag int64) error {
	_db := authRedisPool.Get()
	defer _db.Close()
	now_t := time.Now().Unix()
	uidkey := makeAuthUidKey(uid)
	args := redis.Args{}.Add(uidkey)
	args = args.Add("gagtime").Add(now_t + time_to_gag)

	_, err := redis.String(_db.Do(allmetrics.GetAuthDBStatPrefix("AuthUid", "HMSET"), "HMSET", args...))
	return err
}

func (d *DBByRedis) SetUnInfo(uid db.UserID,
	name, deviceID, passwd, email, authToken string) error {
	_db := authRedisPool.Get()
	defer _db.Close()
	dbpasswd := fmt.Sprintf("%x", secure.DefaultEncode.PasswordForDB(passwd))

	now_t := time.Now().Unix()

	value := &struct {
		Name       string `json:"name" redis:"name"`
		Device     string `json:"device,omitempty" redis:"device"`
		Passwd     string `json:"pwd,omitempty" redis:"pwd"`
		Email      string `json:"email,omitempty" redis:"email"`
		AuthToken  string `json:"authtoken,omitempty" redis:"authtoken"`
		LastTime   int64  `json:"lasttime" redis:"lasttime"`
		CreateTime int64  `json:"createtime" redis:"createtime"`
		BanTime    int64  `json:"bantime" redis:"bantime"`
		GagTime    int64  `json:"gagtime" redis:"gagtime"`
	}{
		Name:       name,
		Device:     deviceID,
		Email:      email,
		Passwd:     dbpasswd,
		AuthToken:  authToken,
		LastTime:   now_t,
		CreateTime: now_t,
		BanTime:    0,
		GagTime:    0,
	}
	uidkey := makeAuthUidKey(uid)
	_, err := redis.String(_db.Do(allmetrics.GetAuthDBStatPrefix("AuthUid", "HMSET"), "HMSET", redis.Args{}.Add(uidkey).AddFlat(value)...))
	return err
}

// 以下两个接口，由于此表是key-range的，所以redis不支持服务器列表显示是否有玩家功能
func (d *DBByRedis) GetUserShardInfo(uid string) ([]AuthUserShardInfo, error) {
	return []dynamodb.AuthUserShardInfo{}, nil
}

func (d *DBByRedis) SetUserShardInfo(uid, gidSidStr string, hasRole bool) error {
	return nil
}

func (d *DBByRedis) AddWhiteList(ip, note string) error {
	_db := authRedisPool.Get()
	defer _db.Close()

	_, err := _db.Do(allmetrics.GetAuthDBStatPrefix("WhiteListInfo", "HSET"), "HSET", "whitelist", ip, note)
	if err != nil {
		return err
	}
	return nil
}

func (d *DBByRedis) CheckWhiteList(ip string) (bool, error) {
	_db := authRedisPool.Get()
	defer _db.Close()

	return redis.Bool(_db.Do(allmetrics.GetAuthDBStatPrefix("WhiteListInfo", "HEXISTS"), "HEXISTS", "whitelist", ip))
}

//func (d *DBByRedis) IncrDeviceTotal() (db.UserID, error) {
//_db := authRedisPool.Get()
//defer _db.Close()
//uid, err := redis.Int64(_db.Do("INCR", "device_total"))
//return db.UserID(uid), err
//}
