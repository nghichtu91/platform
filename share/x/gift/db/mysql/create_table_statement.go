package mysql

import (
	"fmt"
	"github.com/nghichtu91/platform/share/x/gift/config"
)

const (
	CreateTableIfNotExists = "create table if not exists "
)

// CreateGenCodeTableIfNotExistSql 获取创建 config.TableGenCode 生成兑换码表的Sql语句。
//
// Note: 只有在目标Table不存在时才会创建，主键为生成的兑换码。
//
// TODO 创建表的时候需要同时创建索引。
//
// Param-index: 表的划分索引（按照 config.GenCodeSplitByBatch 个批次作为划分依据）。
func CreateGenCodeTableIfNotExistSql(index int) string {
	return fmt.Sprintf(CreateTableIfNotExists+
		"`%v_%v` ( "+
		"`%v` VARCHAR(255) NOT NULL, "+ // 礼包字符码（生成）。
		"`%v` JSON NOT NULL, "+ // 礼包码当前使用者。
		"`%v` INT NOT NULL, "+
		"`%v` INT NOT NULL, "+
		"`%v` TINYINT NOT NULL DEFAULT 0,"+ // 礼包码是否被销毁。
		"%v (`%v`));",
		config.TableGenCode, index, // 表名与索引。
		colGiftCode, colUsers, colBatchID, colGroupID, colDestroy, // 列名。
		primaryKey, colGiftCode) // 主键。
}

// CreateCustomCodeTableIfNotExistSql 获取创建 config.TableCustomCode 自定义兑换码表的Sql语句。
//
// Note: 只有在目标Table不存在时才会创建，主键为自定义兑换码。
func CreateCustomCodeTableIfNotExistSql() string {
	return fmt.Sprintf(CreateTableIfNotExists+
		"`%v` ( "+
		"`%v` VARCHAR(255) NOT NULL, "+ // 礼包字符码（自定义）。
		"`%v` JSON NOT NULL, "+
		"`%v` INT NOT NULL, "+
		"`%v` INT NOT NULL, "+
		"`%v` TINYINT NOT NULL DEFAULT 0,"+ // 礼包码是否被销毁。
		"%v (`%v`));",
		config.TableCustomCode,                                    // 表名。
		colGiftCode, colUsers, colBatchID, colGroupID, colDestroy, // 列名。
		primaryKey, colGiftCode) // 主键。
}

// CreateGiftCodeConfigTableIfNotExistSql 获取创建 config.TableGiftCodeConfig 兑换码配置表表的Sql语句。
//
// Note: 只有在目标Table不存在时才会创建，主键为批次ID+组号ID。
//
// Param-index: 表的划分索引（按照 config.GiftCodeConfigSplitByBatch 个批次作为划分依据）。
func CreateGiftCodeConfigTableIfNotExistSql(index int) string {
	return fmt.Sprintf(CreateTableIfNotExists+
		"`%v_%v` ( "+
		"`%v` INT NOT NULL, "+ // 批次ID。
		"`%v` INT NOT NULL, "+ // 组号ID。
		"`%v` JSON NOT NULL, "+ // 生成兑换码配置信息。
		"%v (`%v`, `%v`));",
		config.TableGiftCodeConfig, index, // 表名与索引。
		colBatchID, colGroupID, colConfig, // 列名。
		primaryKey, colBatchID, colGroupID) // 主键。
}

// CreateGidVisibilityTableIfNotExistSql 获取创建 config.TableGidVisibility 大区可见性表的Sql语句。
//
// Note: 只有在目标Table不存在时才会创建，主键为大区号。
func CreateGidVisibilityTableIfNotExistSql() string {
	return fmt.Sprintf(CreateTableIfNotExists+
		"`%v` ( "+
		"`%v` VARCHAR(255) NOT NULL, "+
		"`%v` JSON NOT NULL, "+
		"%v (`%v`));",
		config.TableGidVisibility, // 表名。
		colGid, colVisibility,     // 列名。
		primaryKey, colGid) // 主键。
}

// CreateL2ATableIfNotExistSql 获取创建 逻辑到实际大区的映射关系表 的Sql语句。
//
// TODO JWS2-5018 LA映射表。
func CreateL2ATableIfNotExistSql() string {
	return fmt.Sprintf(CreateTableIfNotExists+
		"`%v` ( "+
		"`%v` VARCHAR(255) NOT NULL, "+
		"`%v` JSON NOT NULL, "+
		"%v (`%v`));",
		config.TableL2AGid,  // 表名。
		colLogic, colActual, // 列名。
		primaryKey, colLogic) // 主键。
}

// CreateA2LTableIfNotExistSql 获取创建 实际到逻辑大区的映射关系表 的Sql语句。
//
// TODO JWS2-5018 AL映射表。
func CreateA2LTableIfNotExistSql() string {
	return fmt.Sprintf(CreateTableIfNotExists+
		"`%v` ( "+
		"`%v` VARCHAR(255) NOT NULL, "+
		"`%v` VARCHAR(255) NOT NULL, "+
		"%v (`%v`));",
		config.TableA2LGid,  // 表名。
		colActual, colLogic, // 列名。
		primaryKey, colActual) // 主键。
}

// CreateGlobalInfo 获取创建 全局信息表 的Sql语句
func CreateGlobalInfoTableIfNotExistSql() string {
	return fmt.Sprintf(CreateTableIfNotExists+
		"`%v` ( "+
		"`%v` VARCHAR(255) NOT NULL, "+
		"`%v` VARCHAR(255) NOT NULL, "+
		"%v (`%v`));",
		config.TableGlobalInfo, // 表名。
		colInfoName, colInfo,   // 列名。
		primaryKey, colInfoName) // 主键。
}
