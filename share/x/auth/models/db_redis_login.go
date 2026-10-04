package models

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

const (
	acNumberBase = "ac_number_base"
	acAtoi       = "ac_atoi"
	acIota       = "ac_iota"

	tokenUserID      = "user_id"
	tokenCreateAt    = "create_at"
	tokenSDKDeviceID = "sdk_device_id"

	tokenAccessLogTTL = 1 // token访问记录过期时间，1秒

	tokenTTL = 24 * 60 * 60 // token过期时间，1天
)

var (
	ErrAccessSpeedExceedLimit = errors.New("token access speed exceed limit")
	ErrEmptyToken             = errors.New("invalid empty token")
)

// default use dynamoDB
func GetNumberIDForPlayer(acID string) (int64, error) {
	conn := loginRedisPool.Get()
	if conn.IsNil() {
		return 0, fmt.Errorf("nil redis connection")
	}
	defer conn.Close()

	id, err := redis.Int64(conn.Do(allmetrics.GetAuthDBStatPrefix("NumberID", "HGET"), "HGET", acAtoi, acID))
	if err != nil && err != redis.ErrNil {
		return 0, err
	}
	if id == 0 {
		// register redis
		id, err = redis.Int64(conn.Do(allmetrics.GetAuthDBStatPrefix("NumberID", "INCR"), "INCR", acNumberBase))
		if err != nil {
			return 0, err
		}
		conn.Send("MULTI")
		err = conn.Send("HSET", acAtoi, acID, id)
		if err != nil {
			return 0, err
		}
		err = conn.Send("HSET", acIota, id, acID)
		if err != nil {
			return 0, err
		}
		_, err := conn.DoCmdBuffer(allmetrics.GetAuthDBStatPrefix("NumberID", "cmdbuf"), true)
		if err != nil {
			return 0, err
		}
	}
	tilogs.L().Debugf("NumberID is %d", id)
	return id, nil
}

// GetAcidFromNumberID 通过numberID反查对应的acid
func GetAcidFromNumberID(numberID int) (string, error) {
	conn := loginRedisPool.Get()
	if conn.IsNil() {
		return "", fmt.Errorf("nil redis connection")
	}
	defer conn.Close()

	return redis.String(conn.Do(allmetrics.GetAuthDBStatPrefix("AcID", "HGET"), "HGET", acIota, strconv.Itoa(numberID)))
}

func InitNumberIDInRedis() error {
	conn := loginRedisPool.Get()
	if conn.IsNil() {
		return fmt.Errorf("nil redis connection")
	}
	defer conn.Close()

	exist, err := redis.Int(conn.Do(allmetrics.GetAuthDBStatPrefix("NumberID", "EXISTS"), "EXISTS", acNumberBase))
	if err != nil {
		return err
	}
	if exist == 1 {
		return nil
	}
	// register redis
	_, err = conn.Do(allmetrics.GetAuthDBStatPrefix("NumberID", "SET"), "SET", acNumberBase, 0)
	if err != nil {
		return err
	}

	return nil
}

func SetRecallInfo(recallId, accountId string, recallNum, gid int64) error {
	return db_interface.SetRecallPlayerInfo(recallId, accountId, recallNum, gid)
}

func GetRecallID(accountId string) (string, error) {
	info, err := db_interface.QueryRecallIdByAcid(accountId)
	if err != nil || len(info) <= 0 {
		return "", err
	}
	recallid, ok := info["recallid"].(string)
	if !ok {
		return "", err
	}
	return recallid, nil
}

func IsRecallIdIn(recallId string) (bool, error) {
	info, err := db_interface.MarketGetByHashM(recallId)
	return info, err
}

func SetRecallID(uid db.UserID, recallID string) error {
	return db_interface.UpdateRecallID(uid, recallID)
}

func AddShardNewUser(sid string) error {
	key := makeShardUserCountKey(config.Cfg.CommonCfg.Gid)
	conn := loginRedisPool.Get()
	if conn.IsNil() {
		return fmt.Errorf("nil redis connection")
	}
	defer conn.Close()

	_, err := redis.Int(conn.Do(allmetrics.GetAuthDBStatPrefix(key, "HINCRBY"), "HINCRBY", key, sid, 1))
	if err != nil {
		return err
	}

	return nil
}

func GetShardNewUserCount(sid string) (int, error) {
	key := makeShardUserCountKey(config.Cfg.CommonCfg.Gid)
	conn := loginRedisPool.Get()
	if conn.IsNil() {
		return 0, fmt.Errorf("nil redis connection")
	}
	defer conn.Close()
	n, err := redis.Int(conn.Do(allmetrics.GetAuthDBStatPrefix(key, "HGET"), "HGET", key, sid))
	if err != nil {
		return 0, err
	}
	return n, nil
}

// GetAuthToken 获取AuthToken
// 同一个前缀访问token有限制
func GetAuthToken(authToken, prefix string) (db.UserID, string, error) {
	if authToken == "" {
		return db.InvalidUserID, "", ErrEmptyToken
	}
	conn := loginRedisPool.Get()
	if conn.IsNil() {
		return db.InvalidUserID, "", redis.ErrPoolExhausted
	}
	defer conn.Close()

	// 预期的token攻击可能为无效token攻击或有效token攻击
	// 有效token攻击会在下面被access limit限制
	// 如果成功获取token后记录access limit，每次无效token攻击会让redis进行一次HGET操作
	// 如果每次请求记录access limit，每次无效token请求会进行一次SETEX操作，并可能引起缓存雪崩
	// 综合考虑后采用成功获取token后记录的方案

	// 检查是否存在对应token的access limit
	has, err := redis.Bool(conn.Do(allmetrics.GetAuthDBStatPrefix(prefix+":authToken", "EXISTS"), "EXISTS", prefix+":"+authToken))
	if err != nil {
		return db.InvalidUserID, "", err
	}
	if has {
		return db.InvalidUserID, "", ErrAccessSpeedExceedLimit
	}

	v, err := redis.StringMap(conn.Do(allmetrics.GetAuthDBStatPrefix("authToken", "HGETALL"), "HGETALL", authToken))
	switch err {
	case redis.ErrNil:
		err = XErrLoginAuthtokenNotFound
	case nil:
		// 获取token后记录access limit
		conn.Do(allmetrics.GetAuthDBStatPrefix(prefix+":authToken", "SETEX"), "SETEX", prefix+":"+authToken, tokenAccessLogTTL, prefix)
		return db.UserIDFromStringOrNil(v[tokenUserID]), v[tokenSDKDeviceID], nil
	}

	return db.InvalidUserID, "", err
}

// SetAuthToken 设置AuthToken
func SetAuthToken(authToken string, userID db.UserID, sdkDeviceId string) error {
	conn := loginRedisPool.Get()
	if conn.IsNil() {
		return redis.ErrPoolExhausted
	}
	defer conn.Close()

	conn.Send("HMSET", authToken,
		tokenUserID, userID.String(),
		tokenCreateAt, time.Now(),
		tokenSDKDeviceID, sdkDeviceId,
	)
	conn.Send("EXPIRE", authToken, tokenTTL)

	_, err := conn.DoCmdBuffer(allmetrics.GetAuthDBStatPrefix("authToken", "DoCmdBuffer"), false)

	return err
}
