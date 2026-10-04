package models

import (
	"fmt"

	"github.com/nghichtu91/platform/share/x/common/allmetrics"

	"github.com/gomodule/redigo/redis"

	"github.com/nghichtu91/platform/share/planx/servers/db"
)

func (d *DBByRedis) GetAuthToken(authToken string) (db.UserID, error) {
	_db := loginRedisPool.Get()
	defer _db.Close()
	key := makeAuthTokenKey(authToken)
	r, err := redis.String(_db.Do(allmetrics.GetAuthDBStatPrefix("AuthToken", "GET"), "GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return db.InvalidUserID, XErrLoginAuthtokenNotFound
		} else {
			return db.InvalidUserID, err
		}
	}
	return db.UserIDFromStringOrNil(r), nil
}

func (d *DBByRedis) SetAuthToken(authToken string, userID db.UserID, time_out int64) error {
	_db := loginRedisPool.Get()
	defer _db.Close()
	key := makeAuthTokenKey(authToken)
	r, err := redis.String(_db.Do(allmetrics.GetAuthDBStatPrefix("AuthToken", "SETEX"), "SETEX", key, time_out, userID))
	if err != nil {
		return err
	}

	if r != "OK" {
		return fmt.Errorf("LoginRegAuthToken return is not OK in DB")
	}
	return nil
}
