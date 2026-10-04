package mysql

import (
	"fmt"
	"strings"

	"github.com/nghichtu91/platform/share/x/gift/config"
)

// BatchInsertGenCodeSql 获取生成兑换码批量插入的Sql语句。
//
// Param-codes: 需要获取批量插入语句的字符码; Param-index: 需要插入到的目标表的索引。
func BatchInsertGenCodeSql(codes []string, usersJson string, batchID int, groupID int, index int) string {
	batchHeader := fmt.Sprintf("%v %v_%v(%v, %v, %v, %v) values",
		opInsert, config.TableGenCode, index,
		colGiftCode, colUsers, colBatchID, colGroupID)

	buf := make([]byte, 0)
	buf = append(buf, batchHeader...)

	for _, code := range codes {
		buf = append(buf, " ( '"+code+"', '"+usersJson+"', "+fmt.Sprint(batchID)+", "+fmt.Sprint(groupID)+" ),"...)
	}

	buf = buf[:len(buf)-1]
	buf = append(buf, ";"...)

	return string(buf)
}

// InsertCustomCodeSql 获取自定义兑换码插入的Sql语句。
//
// Param-code: 当前批次、组包含的自定义码; Param-usersJson: 使用者列表的Json字符串。
func InsertCustomCodeSql(code string, usersJson string, batchID int, groupID int) string {
	return fmt.Sprintf("%v %v(%v, %v, %v, %v) values ( '%v', '%v', %v, %v );",
		opInsert, config.TableCustomCode,
		colGiftCode, colUsers, colBatchID, colGroupID,
		code, usersJson, batchID, groupID)
}

// InsertGiftCodeConfigSql 获取插入礼包码配置表的Sql语句。
func InsertGiftCodeConfigSql(configJson string, batchID, groupID, index int) string {
	sql := fmt.Sprintf("%v %v_%v(%v, %v, %v) values ( %v, %v, '%v');",
		opInsert, config.TableGiftCodeConfig, index,
		colBatchID, colGroupID, colConfig,
		batchID, groupID, configJson)

	return sql
}

func UpsertGlobalGidInfoSql(newConfig string) string {
	sql := fmt.Sprintf("%v %v(%v, %v) values ( '%v', '%v') %v %v='%v';",
		opInsert, config.TableGlobalInfo,
		colInfoName, colInfo,
		global_gid, newConfig,
		duplicateKeyUpdate,
		colInfo, newConfig)
	return sql
}

// InsertGidVisibilitySql 新增大区可见性的Sql语句。
func InsertGidVisibilitySql(gid string, visibility string) string {
	return fmt.Sprintf("%v %v(%v, %v) values ( '%v', '%v');",
		opInsert, config.TableGidVisibility,
		colGid, colVisibility,
		gid, visibility)
}

// UpdateGiftCodeConfigSql 获取更新礼包码配置表的配置字段的Sql语句。
func UpdateGiftCodeConfigSql(configJson string, batchID, groupID, index int) string {
	return fmt.Sprintf("%v %v_%v set %v='%v' where %v='%v' and %v='%v';",
		opUpdate, config.TableGiftCodeConfig, index,
		colConfig, configJson,
		colBatchID, batchID,
		colGroupID, groupID)
}

// UpdateGidVisibilitySql 更改大区可见性的Sql语句。
func UpdateGidVisibilitySql(gid string, visibility string) string {
	return fmt.Sprintf("%v %v set %v='%v' where %v='%v';",
		opUpdate, config.TableGidVisibility,
		colVisibility, visibility,
		colGid, gid)
}

// DestroyCustomCodeSql 获取销毁自定义兑换码表的指定兑换码Sql。
func DestroyCustomCodeSql(code string) string {
	return fmt.Sprintf("%v %v set %v=%v where %v='%v';",
		opUpdate, config.TableCustomCode,
		colDestroy, true,
		colGiftCode, code)
}

// DestroyGenCodeSql 获取销毁生成兑换码表的指定兑换码Sql。
func DestroyGenCodeSql(code string, index int) string {
	return fmt.Sprintf("%v %v_%v set %v=%v where %v='%v';",
		opUpdate, config.TableGenCode, index,
		colDestroy, true,
		colGiftCode, code)
}

func UpdateUsesCustomCodeSql(code string, useJson string) string {
	return fmt.Sprintf("%v %v set %v='%v' where %v='%v';",
		opUpdate, config.TableCustomCode,
		colUsers, useJson,
		colGiftCode, code)
}

func UpdateUsesGenCodeSql(code string, useJson string, index int) string {
	sql := fmt.Sprintf("%v %v_%v set %v='%v' where %v='%v';",
		opUpdate, config.TableGenCode, index,
		colUsers, useJson,
		colGiftCode, code)
	return sql
}

// QueryJudgeGenCodeConfigSql 获取兑换码配置查询检查Sql语句。
//
// Note: 由于表中含有体量较大的字段所以判断时只取配置列。
func QueryJudgeGenCodeConfigSql(batchID, groupID, index int) string {
	return fmt.Sprintf("%v %v from %v_%v where %v='%v' and %v='%v';",
		opSelect, colConfig, config.TableGiftCodeConfig, index,
		colBatchID, batchID,
		colGroupID, groupID)
}

// QueryGiftCodeConfigSql 获取兑换码配置查询Sql语句。
func QueryGiftCodeConfigSql(batchID, groupID, index int) string {
	return fmt.Sprintf("%v %v from %v_%v where %v='%v' and %v='%v';",
		opSelect, colConfig, config.TableGiftCodeConfig, index,
		colBatchID, batchID,
		colGroupID, groupID)
}

// QueryGidVisibilitySql 查询指定的大区可见性信息
func QueryGidVisibilitySql(gid string) string {
	sql := fmt.Sprintf("%v %v from %v where %v='%v';",
		opSelect, colVisibility, config.TableGidVisibility,
		colGid, gid)

	return sql
}

// QueryBatchGroupIDByCustomSql 获取通过给定的自定义兑换码获取其所在的批次和组号的Sql。
func QueryBatchGroupIDByCustomSql(customCode string) string {
	return fmt.Sprintf("%v %v, %v from %v where %v='%v';",
		opSelect, colBatchID, colGroupID, config.TableCustomCode,
		colGiftCode, customCode)
}

// QueryUseInfoByGenCodeSql 获取依据给定生成兑换码获取使用者信息的Sql语句。
func QueryUseInfoByGenCodeSql(genCode string, index int) string {
	return fmt.Sprintf("%v %v, %v from %v_%v where %v='%v';",
		opSelect, colUsers, colDestroy, config.TableGenCode, index,
		colGiftCode, genCode)
}

// QueryUseInfoByCustomCodeSql
func QueryUseInfoByCustomCodeSql(customCode string) string {
	return fmt.Sprintf("%v %v, %v from %v where %v='%v';",
		opSelect, colUsers, colDestroy, config.TableCustomCode,
		colGiftCode, customCode)
}

// QueryGenUseInfoByBatchGroup 获取通过批次和组信息查询生成兑换码的Sql语句。
func QueryGenUseInfoByBatchGroup(batchID, groupID, index int) string {
	sql := fmt.Sprintf("%v %v, %v, %v from %v_%v where %v=%v and %v=%v;",
		opSelect, colGiftCode, colDestroy, colUsers, config.TableGenCode, index,
		colBatchID, batchID,
		colGroupID, groupID)
	return sql
}

// QueryCustomUseInfoByBatchGroup 获取通过批次和组信息查询自定义兑换码的Sql语句。
func QueryCustomUseInfoByBatchGroup(batchID, groupID int) string {
	sql := fmt.Sprintf("%v %v, %v, %v from %v where %v=%v and %v=%v;",
		opSelect, colGiftCode, colDestroy, colUsers, config.TableCustomCode,
		colBatchID, batchID,
		colGroupID, groupID)
	return sql
}

func QueryGlobalGidInfoSql() string {
	sql := fmt.Sprintf("%v %v from %v where %v='%v';",
		opSelect, colInfo, config.TableGlobalInfo,
		colInfoName, global_gid)
	return sql
}

// QueryGreatestBatch 获取配置表列表中最大批次号的Sql。
//
// Param-configTables： 配置表列表（需要保证非空）。
func QueryGreatestBatch(configTables []string) string {
	var singleMaxSlice []string
	for _, configTable := range configTables {
		singleMaxSlice = append(singleMaxSlice, fmt.Sprintf("coalesce((%v max(%v) from %v), 0)", opSelect, colBatchID, configTable))
	}

	sql := fmt.Sprintf("%v (%v, 0)", opSelectGreatest, strings.Join(singleMaxSlice, ","))

	return sql
}

// QueryGreatestGroupInBatch 获取配置表列表中某批次的最大组号的Sql。
//
// Param-configTables: 配置表列表（需要保证非空）。Param-batchID: 批次号限定条件。
func QueryGreatestGroupInBatch(configTables []string, batchID int) string {
	var singleMaxSlice []string
	for _, configTable := range configTables {
		singleMaxSlice = append(singleMaxSlice, fmt.Sprintf("coalesce((%v max(%v) from %v where %v=%v), 0)", opSelect, colGroupID, configTable, colBatchID, batchID))
	}

	return fmt.Sprintf("%v (%v, 0)", opSelectGreatest, strings.Join(singleMaxSlice, ","))
}

// ShowTablesLike 获取包含指定前缀的所有表名。
//
// Param-tablePrefix: 前缀字符串。
func ShowTablesLike(tablePrefix string) string {
	return fmt.Sprintf("%v '%v%%'", opShowTables, tablePrefix)
}
