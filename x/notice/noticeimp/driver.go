package noticeimp

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type Driver interface {
	Get(gid uint, version string) []*NoticeInfo
}

func NewDriver(mysqlUrl string) Driver {
	db, err := sql.Open("mysql", mysqlUrl)
	if err != nil {
		tilogs.L().Errorf("init db error %v", err)
		panic(err)
	}
	if err := db.Ping(); err != nil {
		tilogs.L().Errorf("ping db error %v", err)
		panic(err)
	}
	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxIdleConns(5)

	tilogs.L().Infof("[START]init db ok")
	return &MysqlDriver{
		db: db,
	}
}
