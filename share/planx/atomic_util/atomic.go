package atomic_util

import (
	"errors"

	"github.com/nghichtu91/platform/share/planx/timongo"
	"github.com/nghichtu91/platform/share/planx/timysql"
	"github.com/nghichtu91/platform/share/planx/tiredis"
)

// IAtomicDB 可持久化的数据库
type IAtomicDB interface {
	// Incr 原子性的返回自增的int值
	// 如果对应key之前不存在，新建并返回1
	Incr(key string) (int64, error)

	// Close 关闭对应的db链接
	Close() error
}

var (
	ErrCfgNil         = errors.New("atomic db config is nil")
	ErrNotSupportType = errors.New("config type not support")
)

const (
	MongoDBNameDefault = "DBGlobal"
	MySQLDBNameDefault = "jws2_global"
	TableNameDefault   = "Misc"
	PriKeyNameDefault  = "UIDType"
)

const (
	_ = iota
	TypeEtcd
	TypeMongo
	TypeMySQL
	TypeRedis
)

// AtomicDBCfg 通用的初始化设置
// 目前支持etcd, mongo, mysql, redis
type AtomicDBCfg struct {
	Type int

	*timongo.MongoCfg
	*timysql.MySqlDBCfg
	*tiredis.RedisDBCfg
}

var (
	ErrDBNotInit = errors.New("atomic db not init")
)

var (
	idb IAtomicDB
)

// InitWithDB 使用已有数据实例初始化AtomicDB
func InitWithDB(db IAtomicDB) {
	idb = db
}

// InitWithCfg 使用配置文件初始化AtomicDB
func InitWithCfg(cfg *AtomicDBCfg) error {
	if cfg == nil {
		return ErrCfgNil
	}

	var err error
	var db IAtomicDB

	switch cfg.Type {
	case TypeMongo:
		if cfg.MongoCfg == nil {
			return ErrCfgNil
		}
		// 一些默认值
		if cfg.MongoCfg.DBName == "" {
			cfg.MongoCfg.DBName = MongoDBNameDefault
		}
		if cfg.MongoCfg.ColName == "" {
			cfg.MongoCfg.ColName = TableNameDefault
		}
		if cfg.MongoCfg.PriKeyName == "" {
			cfg.MongoCfg.PriKeyName = PriKeyNameDefault
		}

		db, err = timongo.NewMongoConn(cfg.MongoCfg)
	case TypeMySQL:
		if cfg.MySqlDBCfg == nil {
			return ErrCfgNil
		}
		// 一些默认值
		if cfg.MySqlDBCfg.DBName == "" {
			cfg.MySqlDBCfg.DBName = MySQLDBNameDefault
		}
		if cfg.MySqlDBCfg.TableName == "" {
			cfg.MySqlDBCfg.TableName = TableNameDefault
		}
		db, err = timysql.NewMySqlConn(cfg.MySqlDBCfg)
	case TypeRedis:
		if cfg.RedisDBCfg == nil {
			return ErrCfgNil
		}
		db, err = tiredis.NewRedisConn(cfg.RedisDBCfg)
	default:
		return ErrNotSupportType
	}

	if err != nil {
		return err
	}

	idb = db

	return nil
}

// Incr 原子性的返回自增的int值
// 如果对应key之前不存在，新建并返回1
func Incr(key string) (int64, error) {
	if idb == nil {
		return 0, ErrDBNotInit
	}
	return idb.Incr(key)
}
