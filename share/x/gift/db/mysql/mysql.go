package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/x/gift/model/errs"

	"github.com/nghichtu91/platform/share/x/gift/config"

	_ "github.com/go-sql-driver/mysql"
)

var mysqlDB *sql.DB

// 初始化Mysql数据库链接池。
func InitDB() error {
	// 创建MySql连接池对象。
	giftCfg := config.Cfg.GiftCfg
	db, openErr := sql.Open(giftCfg.DBDriver, config.GetDBConnMySql(giftCfg.DBUrl))
	if openErr != nil {
		return fmt.Errorf("InitMySqlDB sql.Open err: %v", openErr)
	}

	// 与Mysql最大连接限制。
	db.SetMaxOpenConns(giftCfg.DBMaxOpenConns)
	// 与Mysql最大闲置连接。
	db.SetMaxIdleConns(giftCfg.DBMaxIdleConns)
	// 与Mysql连接最大时长。
	db.SetConnMaxLifetime(time.Second * time.Duration(giftCfg.DBMaxLifeTime))

	// 验证数据库链接池有效。
	if pingErr := db.Ping(); pingErr != nil {
		return fmt.Errorf("InitMySqlDB ping err: %v", pingErr)
	}

	mysqlDB = db
	return nil
}

// 关闭MySql链接对象。
func CloseDB() {
	if mysqlDB != nil {
		if errClose := mysqlDB.Close(); errClose != nil {
			tilogs.L().Errorf("mysqlDB.Close err: %v", errClose)
		}
	}
}

// GetDB 获取Mysql连接池对象。
func GetDB() *sql.DB {
	return mysqlDB
}

// Commit 执行Mysql提交。
func Commit(sqlTS *sql.Tx) (error, string) {
	// 执行提交结果。
	if errCommit := sqlTS.Commit(); errCommit != nil {
		if err := sqlTS.Rollback(); err != nil {
			return fmt.Errorf("commit err : %v; rollback err: %v", errCommit, err), errs.ErrDBCommitTransactionAndRollback
		}
		return errCommit, errs.ErrDBCommitTransaction
	}

	return nil, errs.Success
}
