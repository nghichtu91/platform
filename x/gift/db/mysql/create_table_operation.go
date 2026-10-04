package mysql

import (
	"database/sql"
	"fmt"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
)

// createTableIfNotExist 创建不存在的表格。
//
// Param-configIndex: 配置表的拆分力度（索引依据）; Param-codeIndex: 兑换码表的拆分力度（索引依据）;
// Param-db: 承载数据库操作。
func CreateTableIfNotExist(configIndex int, codeIndex int, db *sql.DB) (error, string) {
	if err, errMsg := CreateConfigTableIfNotExist(configIndex, db); err != nil {
		return err, errMsg
	}
	if err, errMsg := CreateGenTableIfNotExist(codeIndex, db); err != nil {
		return err, errMsg
	}
	if err, errMsg := CreateCustomTableIfNotExist(db); err != nil {
		return err, errMsg
	}
	if err, errMsg := CreateGidTableIfNotExist(db); err != nil {
		return err, errMsg
	}
	return nil, errs.Success
}

// CreateConfigTableIfNotExist 如果兑换码配置表不存在则创建。
//
// Param-configIndex: 配置表的拆分力度（索引依据）; Param-db: 承载数据库操作。
func CreateConfigTableIfNotExist(configIndex int, db *sql.DB) (error, string) {
	_, err := db.Exec(CreateGiftCodeConfigTableIfNotExistSql(configIndex))
	if err != nil {
		return fmt.Errorf("createConfigTableIfNotExist err: %v", err), errs.ErrDBCheckAndCreateTable
	}
	return nil, errs.Success
}

// CreateGenTableIfNotExist 如果生成兑换码表不存在则创建。
//
// Param-codeIndex: 兑换码表的拆分力度（索引依据）; Param-db: 承载数据库操作。
func CreateGenTableIfNotExist(codeIndex int, db *sql.DB) (error, string) {
	_, err := db.Exec(CreateGenCodeTableIfNotExistSql(codeIndex))
	if err != nil {
		return fmt.Errorf("createGenTableIfNotExist err: %v", err), errs.ErrDBCheckAndCreateTable
	}

	return nil, errs.Success
}

// CreateCustomTableIfNotExist 如果自定义兑换码表不存在则创建。
//
// Param-db: 承载数据库操作。
func CreateCustomTableIfNotExist(db *sql.DB) (error, string) {
	_, err := db.Exec(CreateCustomCodeTableIfNotExistSql())
	if err != nil {
		return fmt.Errorf("createCustomTableIfNotExist err: %v", err), errs.ErrDBCheckAndCreateTable
	}
	return nil, errs.Success
}

// CreateGidTableIfNotExist 如果大区可见性表不存在则创建。
//
// Param-db: 承载数据库操作。
func CreateGidTableIfNotExist(db *sql.DB) (error, string) {
	_, err := db.Exec(CreateGidVisibilityTableIfNotExistSql())
	if err != nil {
		return fmt.Errorf("createGidTableIfNotExist err: %v", err), errs.ErrDBCheckAndCreateTable
	}
	return nil, errs.Success
}

func CreateGlobalTableIfNotExist(db *sql.DB) (error, string) {
	_, err := db.Exec(CreateGlobalInfoTableIfNotExistSql())
	if err != nil {
		return fmt.Errorf("CreateGlobalTableIfNotExist err: %v", err), errs.ErrDBCheckAndCreateTable
	}
	return nil, errs.Success
}
