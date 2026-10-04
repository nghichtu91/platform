package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nghichtu91/platform/share/x/chat/web/config"
)

var sqlDb *sql.DB

func InitDB() {
	dbCfg := config.Cfg

	mysqlUrls := strings.Split(dbCfg.GidConfig.MysqlUrl, "/")
	if len(mysqlUrls) != 2 {
		panic(fmt.Sprintf("init db mysqlUrl err, %s", dbCfg.GidConfig.MysqlUrl))
	}

	url := mysqlUrls[0]
	dataBase := mysqlUrls[1]

	dbParam := fmt.Sprintf("%s/?charset=utf8&multiStatements=true", url)
	db, err := sql.Open("mysql", dbParam)
	if err != nil {
		panic(fmt.Sprintf("init db error %v", err))
	}
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + dataBase)
	if err != nil {
		panic(fmt.Sprintf("create db error %v", err))
	}

	dbParam = fmt.Sprintf("%s?charset=utf8&multiStatements=true", dbCfg.GidConfig.MysqlUrl)
	db, err = sql.Open("mysql", dbParam)
	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("ping db error %v", err))
	}
	sqlDb = db
}

func GetDB() *sql.DB {
	return sqlDb
}
