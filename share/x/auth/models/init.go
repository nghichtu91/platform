package models

import (
	"fmt"

	"github.com/nghichtu91/platform/share/x/auth/config"

	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/nghichtu91/platform/share/planx/redispool"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	cconfig "github.com/nghichtu91/platform/share/x/common/config"
)

const AUTHTOKEN_TIMEOUT = 24 * 60 * 60 // min

type DBConfig struct {
	DynamoRegion          string
	DynamoAccessKeyID     string
	DynamoSecretAccessKey string
	DynamoSessionToken    string

	MongoDBUrl  string
	MongoDBName string
}

type DBByRedis struct {
}

var (
	loginRedisPool redispool.IPool
	authRedisPool  redispool.IPool
	db_config      DBConfig
)

func InitDb(cfg *config.CommonConfig, gidConfig *cconfig.GidConfig) error {
	db_config.DynamoRegion = gidConfig.AwsRegion
	db_config.DynamoAccessKeyID = gidConfig.AwsAccessKey
	db_config.DynamoSecretAccessKey = gidConfig.AwsSecretKey
	db_config.MongoDBUrl = gidConfig.MongoUrl
	db_config.MongoDBName = cfg.MongoAuthDbName
	switch gidConfig.DBDriver {
	case consts.Driver_Mongo:
		tilogs.L().Infof("Auth start with mongodb %v", db_config)
		db_interface = &DBByMongoDBV2{}
	case consts.Driver_Dynamo:
		fallthrough
	default:
		db_interface = &DBByDynamoDB{}
		InitAuthRedis(cfg)
		tilogs.L().Infof("Auth start with DynamoDB and redis %v", db_config)
	}
	tilogs.L().Infof("Auth start with %v", db_config)
	return db_interface.Init(db_config)
}

func InitLoginRedis(cfg *config.CommonConfig) {
	loginRedisPool = newRedisPool("loginredis",
		cfg.LoginRedisAddr,
		cfg.LoginRedisDbPwd,
		cfg.LoginRedisDb,
		cfg.LoginRedisAddr, config.GidCfg)
}

func InitAuthRedis(cfg *config.CommonConfig) {
	authRedisPool = newRedisPool("authredis",
		cfg.AuthRedisAddr,
		cfg.AuthRedisDbPwd,
		cfg.AuthRedisDb,
		cfg.AuthRedisAddr, config.GidCfg)
}

func makeAuthTokenKey(authToken string) string {
	return fmt.Sprintf("at:%s", authToken)
}

func makeShardUserCountKey(gid uint) string {
	return fmt.Sprintf("shardusercount:%d", gid)
}

// GetRedisLimitConn 获取用于访问限制的redis conn
func GetRedisLimitConn() redispool.RedisPoolConn {
	return loginRedisPool.Get()
}
