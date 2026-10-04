package timysql

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var (
	ErrMySQLNotInit = errors.New("my sql not init")
)

const (
	NameGlobalDB  = "jws2_global"
	NameTableMisc = "Misc"
	KeyUIDType    = "UIDType"
	KeyNextUID    = "NextUID"
)

// MySqlDB 简单封装的SQL
// 用于实现一些简单的功能
type MySqlDB struct {
	db *sql.DB
}

// MySqlDBCfg 初始化MySQL需要的配置
type MySqlDBCfg struct {
	Driver     string
	URL        string
	DBName     string
	TableName  string
	User       string
	Pwd        string
	PriKeyName string
}

func initDB(cfg *MySqlDBCfg, withDBName bool) (*sql.DB, error) {
	dbName := cfg.DBName
	if !withDBName {
		dbName = ""
	}
	dbParam := fmt.Sprintf("%s:%s@%s/%s?charset=utf8&multiStatements=true", cfg.User, cfg.Pwd, cfg.URL, dbName)
	db, err := sql.Open(cfg.Driver, dbParam)
	if err != nil {
		tilogs.L().Errorf("init db error %v", err)
		return nil, err
	}
	if err := db.Ping(); err != nil {
		tilogs.L().Errorf("ping db error %v", err)
		return nil, err
	}
	return db, nil
}

// NewGlobalMySqlConn 通过mysql dsn构造global conn
// DBName会被改写不生效，其他键值使用const里的默认值
func NewGlobalMySqlConn(dsn string) (*MySqlDB, error) {
	cfg, err := ParseDSNForGlobalDB(dsn)
	if err != nil {
		return nil, err
	}
	return NewMySqlConn(cfg)
}

// NewMySqlConn 初始化mysql
func NewMySqlConn(cfg *MySqlDBCfg) (*MySqlDB, error) {
	db, err := initDB(cfg, false)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + cfg.DBName)
	if err != nil {
		tilogs.L().Errorf("create db error %v", err)
		return nil, err
	}

	db, err = initDB(cfg, true)
	if err != nil {
		return nil, err
	}

	// TODO 这个是Atomic独有的表结构，后续需要根据不同的表类型定义不同的表结构，譬如把建表的query写cfg里面
	cmd := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s(%s varchar(64) primary key, NextUID int(10) not null default 0)", cfg.TableName, cfg.PriKeyName)
	_, err = db.Exec(cmd)
	if err != nil {
		tilogs.L().Errorf("create table error %v", err)
		return nil, err
	}

	return &MySqlDB{db: db}, nil
}

// Close 关闭mysql链接
func (db *MySqlDB) Close() error {
	if db == nil {
		return nil
	}
	return db.db.Close()
}

// Incr 原子性的返回自增的int值
// 如果对应key之前不存在，返回1
//
// 实现IAtomicDB接口
func (db *MySqlDB) Incr(key string) (int64, error) {
	if db.db == nil {
		return 0, ErrMySQLNotInit
	}

	tx, err := db.db.Begin()
	if err != nil {
		return 0, err
	}

	// upsert
	_, err = tx.Exec("INSERT Misc SET UIDType = ? ON DUPLICATE KEY UPDATE NextUID = NextUID + 1", key)
	if err != nil {
		tilogs.L().Errorf("upsert failed, err %v", err)
		return 0, err
	}

	// query
	var uid int64
	err = tx.QueryRow("SELECT NextUID FROM Misc WHERE UIDType = ?", key).Scan(&uid)
	if err != nil {
		tilogs.L().Errorf("query failed, err %v", err)
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	tilogs.L().Infof("Incr success, uid type %s last uid %d", key, uid)

	return uid + 1, nil
}

func ParseDSNForGlobalDB(param string) (*MySqlDBCfg, error) {
	cfg, err := mysql.ParseDSN(param)
	if err != nil {
		return nil, err
	}
	return &MySqlDBCfg{
		Driver:     "mysql",
		URL:        fmt.Sprintf("%s(%s)", cfg.Net, cfg.Addr),
		DBName:     NameGlobalDB, // 替换其他dbname为
		TableName:  NameTableMisc,
		User:       cfg.User,
		Pwd:        cfg.Passwd,
		PriKeyName: KeyUIDType,
	}, nil
}
