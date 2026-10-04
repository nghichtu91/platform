package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nghichtu91/platform/share/x/gift/db/oss"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/gift/config"
	"github.com/nghichtu91/platform/share/x/gift/db/mysql"
	"github.com/nghichtu91/platform/share/x/gift/model/define"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
)

// JudgeBatchGroupExist 判断指定的批次和组号是否已经存在，若存在返回配置信息。
//
// Param-codeConfig: 需要判断的配置信息。
//
// Return-bool: 对应批次和组是否已经存在; Return-*define.GiftCodeInfo: 如果存在，则同时返回先前的生成配置.
func JudgeBatchGroupExist(batchID, groupID int) (bool, *define.GiftCodeInfo, error, string) {
	// 获取数据库连接池对象。
	sqlDB := mysql.GetDB()

	// 获取目标表的索引值。
	giftCodeConfigIndex := batchID / config.GiftCodeConfigSplitByBatch

	// 如果兑换码配置表不存在则创建。
	if err, errMsg := mysql.CreateConfigTableIfNotExist(giftCodeConfigIndex, sqlDB); err != nil {
		return false, nil, err, errMsg
	}

	// 尝试获取对应批次和组号的配置信息。
	if result, err := sqlDB.Query(mysql.QueryJudgeGenCodeConfigSql(batchID, groupID, giftCodeConfigIndex)); err != nil {
		return false, nil, fmt.Errorf("JudgeBatchGroupExist query config err: %v", err), errs.ErrDBQueryGiftCodeConfig
	} else {
		// 判断对应批次和组是否已经存在。
		exist, oldConfig, err, errMsg := func() (bool, *define.GiftCodeInfo, error, string) {
			defer func() {
				if err := result.Close(); err != nil {
					tilogs.L().Errorf("close mysql conn err: %v", err)
				}
			}()

			if result.Next() {
				// 读取以Json格式存储的兑换码配置。
				var giftCodeConfigJson string
				if err := result.Scan(&giftCodeConfigJson); err != nil {
					return false, nil, fmt.Errorf("JudgeBatchGroupExist batchID: %v, groupID: %v scan err", batchID, groupID), errs.ErrDBScan
				}

				// 将Json字符串转换为对应的数据结构。
				var giftCodeConfig *define.GiftCodeInfo
				if errConfig := json.Unmarshal([]byte(giftCodeConfigJson), &giftCodeConfig); errConfig != nil {
					return false, nil, fmt.Errorf("JudgeBatchGroupExist json unmarshal err: %v", errConfig), errs.ErrDBJsonUnmarshal
				}

				if giftCodeConfig == nil {
					return false, nil, fmt.Errorf("JudgeBatchGroupExist data is nil"), errs.ErrDBData
				}

				return true, giftCodeConfig, nil, errs.Success
			}

			return false, nil, nil, errs.Success
		}()
		return exist, oldConfig, err, errMsg
	}
}

// InsertGenCodeList 向对应表中插入新生成的兑换码信息。
//
// Note: 注意！执行方式为事务，多表操作要么全部执行，要么全部不做。
//
// Param-codes: 本次生成的所有兑换字符码; Param-batchID: 本次操作的批次号; Param-groupID: 本次操作的组号;
// Param-codeConfig: 本次操作的兑换码配置; Param-append:本次操作是否为追加模式。
func InsertGenCodeList(codes []string, codeConfig *define.GiftCodeInfo, append bool) (error, string) {
	// 获取数据库连接池对象。
	db := mysql.GetDB()

	// 获取需要操作的表索引。
	giftCodeConfigIndex := codeConfig.BatchID / config.GiftCodeConfigSplitByBatch
	genCodeIndex := codeConfig.BatchID / config.GenCodeSplitByBatch

	// 如果表不存在则创建（DDL语句无法也不需要执行回滚，不放在事务中）。
	if err, errMsg := mysql.CreateTableIfNotExist(giftCodeConfigIndex, genCodeIndex, db); err != nil {
		return err, errMsg
	}

	// 开启数据库事务。
	ts, errTS := mysql.GetDB().Begin()
	if errTS != nil {
		return fmt.Errorf("InsertGenCodeList transaction begin err: %v", errTS), errs.ErrDBTransactionBegin
	}

	// 判断、更新兑换码配置信息表与大区可见性表。
	codesJson, errConfig, errConfigMsg := opCodeConfigAndGidTransaction(codes, codeConfig, giftCodeConfigIndex, append, ts)
	if errConfig != nil {
		if err := ts.Rollback(); err != nil {
			return fmt.Errorf("opCodeConfigAndGidTransaction op err : %v; rollback err: %v", errConfig, err), errs.ErrDBOpGenCodeTablesTransactionAndRollback
		}
		return errConfig, errConfigMsg
	}

	// 将生成的兑换码插入至生成兑换码表。
	if errGenInsert, errGenMsg := insertGiftCodeTransaction(codes, codeConfig, genCodeIndex, ts); errGenInsert != nil {
		if err := ts.Rollback(); err != nil {
			return fmt.Errorf("insertGiftCodeTransaction insert err : %v; rollback err: %v", errGenInsert, err), errs.ErrDBInsertGenCodeTransactionAndRollback
		}

		if strings.Contains(errGenInsert.Error(), Duplicate) {
			return errGenInsert, errs.ErrHandlerDuplicateCode
		}

		return errGenInsert, errGenMsg
	}

	// 执行提交结果。
	if err, errMsg := mysql.Commit(ts); err != nil {
		return err, errMsg
	}

	// codesJson上传至OSS。
	if err, errMsg := oss.UploadToOSS([]byte(codesJson), codeConfig.BatchID, codeConfig.GroupID); err != nil {
		return fmt.Errorf("InsertGenCodeList codes UploadToOSS err, only effect download: %v", err), errMsg + errs.ErrDBOSSUploadNoEffect
	}

	return nil, errs.Success
}

// UpdateGiftCodeConfig 更新礼包码的配置信息。
//
// Note: 注意！执行方式为事务，配置表和可见性表要么全部修改，要么全部不做。
//
// Param-oldConfig: 数据库原有的配置信息; Param-newConfig: 更新且通过检查的配置信息。
func UpdateGiftCodeConfig(oldConfig, newConfig *define.GiftCodeInfo) (error, string) {
	// JsonMarshal新的配置信息。
	configResultJson, errConfig := json.Marshal(newConfig)
	if errConfig != nil {
		return fmt.Errorf("UpdateGiftCodeConfig json marshal err: %v", errConfig), errs.ErrDBJsonMarshal
	}

	// 获取数据库连接池对象。
	sqlDB := mysql.GetDB()

	index := newConfig.BatchID / config.GiftCodeConfigSplitByBatch

	// 如果配置表不存在，创建配置表。
	if err, errMsg := mysql.CreateConfigTableIfNotExist(index, sqlDB); err != nil {
		return err, errMsg
	}

	// 如果可见表不存在，创建可见表。
	if err, errMsg := mysql.CreateGidTableIfNotExist(sqlDB); err != nil {
		return err, errMsg
	}

	// 开启Mysql事务。
	sqlTS, sqlTSErr := sqlDB.Begin()
	if sqlTSErr != nil {
		return sqlTSErr, errs.ErrDBTransactionBegin
	}

	// 修改礼包码配置表。
	if errConfig, errConfigMsg := updateGiftCodeConfig(string(configResultJson), newConfig.BatchID, newConfig.GroupID, index, sqlTS); errConfig != nil {
		if err := sqlTS.Rollback(); err != nil {
			return fmt.Errorf("updateGiftCodeConfig update err : %v; rollback err: %v", errConfig, err), errs.ErrDBOpGenCodeTablesTransactionAndRollback
		}
		return errConfig, errConfigMsg
	}

	// 修改大区可见性表。
	if errGid, errGidMsg := updateGidVisibility(oldConfig, newConfig, sqlTS); errGid != nil {
		if err := sqlTS.Rollback(); err != nil {
			return fmt.Errorf("updateGidVisibility update err : %v; rollback err: %v", errGid, err), errs.ErrDBOpGenCodeTablesTransactionAndRollback
		}
		return errGid, errGidMsg
	}

	// 执行提交结果
	if err, errMsg := mysql.Commit(sqlTS); err != nil {
		return err, errMsg
	}

	return nil, errs.Success
}

// DestroyGiftCodes 销毁给定的礼包码。
//
// Note: 注意！执行方式为事务，销毁要么全部完成，要么均未销毁。
//
// Param-codes: 需要销毁的兑换码; Param-batchID: 销毁限定验证批次。
// Param-isCustom: 是否为自定义兑换码（自定义码与生成码的存放位置不同）。
func DestroyGiftCodes(codes []string, batchID int, isCustom bool) (error, string) {
	// 获取数据库连接池对象。
	sqlDB := mysql.GetDB()

	if isCustom {
		// 如果自定义兑换码表不存在则创建。
		if err, errMsg := mysql.CreateCustomTableIfNotExist(sqlDB); err != nil {
			return err, errMsg
		}

		// 开启Mysql事务。
		sqlTS, sqlTSErr := sqlDB.Begin()
		if sqlTSErr != nil {
			return sqlTSErr, errs.ErrDBTransactionBegin
		}

		// 以事务的形式执行兑换码销毁操作。
		if errDestroy, errDestroyMsg := destroyCustomCodesTransaction(codes, sqlTS); errDestroy != nil {
			if err := sqlTS.Rollback(); err != nil {
				return fmt.Errorf("DestroyGiftCodes err rollback err"), errs.ErrDBDestroyCodesTransactionAndRollback
			}
			return errDestroy, errDestroyMsg
		}

		// 执行提交结果。
		if errCommit := sqlTS.Commit(); errCommit != nil {
			if err := sqlTS.Rollback(); err != nil {
				return fmt.Errorf("commit err : %v; rollback err: %v", errCommit, err), errs.ErrDBCommitTransactionAndRollback
			}
			return errCommit, errs.ErrDBCommitTransaction
		}

		return nil, errs.Success
	} else {
		// 获取生成表索引
		index := batchID / config.GenCodeSplitByBatch

		// 如果生成兑换码表不存在则创建。
		if err, errMsg := mysql.CreateGenTableIfNotExist(index, sqlDB); err != nil {
			return err, errMsg
		}

		// 开启Mysql事务。
		sqlTS, sqlTSErr := sqlDB.Begin()
		if sqlTSErr != nil {
			return sqlTSErr, errs.ErrDBTransactionBegin
		}

		// 以事务的形式执行兑换码销毁操作。
		if errDestroy, errDestroyMsg := destroyGenCodesTransaction(codes, index, sqlTS); errDestroy != nil {
			if err := sqlTS.Rollback(); err != nil {
				return fmt.Errorf("DestroyGiftCodes err rollback err"), errs.ErrDBDestroyCodesTransactionAndRollback
			}
			return errDestroy, errDestroyMsg
		}

		// 执行提交结果
		if err, errMsg := mysql.Commit(sqlTS); err != nil {
			return err, errMsg
		}

		return nil, errs.Success
	}
}

// GetBatchGroupIDByCustomCode 通过自定义码获取所在的批次和组号。
//
// Note: 若没有获取到所在的批次和组号，按异常处理。
//
// Param-customCode: 自定义兑换码。
//
// Return-int: 自定义码所在批次; Return-int: 自定义码所在组号。
func GetBatchGroupIDByCustomCode(customCode string) (int, int, error, string) {
	// 获取数据库连接池对象。
	sqlDB := mysql.GetDB()

	// 如果自定义兑换码表不存在则创建。
	if err, errMsg := mysql.CreateCustomTableIfNotExist(sqlDB); err != nil {
		return -1, -1, err, errMsg
	}

	// 尝试获取自定义兑换码的批次和组号。
	if result, err := sqlDB.Query(mysql.QueryBatchGroupIDByCustomSql(customCode)); err != nil {
		return -1, -1, fmt.Errorf("GetBatchGroupIDByCustomCode err: %v", err), errs.ErrDBQueryCustomCode
	} else {
		batchID, groupID, err, errMsg := func() (int, int, error, string) {
			defer func() {
				if err := result.Close(); err != nil {
					tilogs.L().Errorf("close mysql conn err: %v", err)
				}
			}()

			if result.Next() {
				// 找到了自定义兑换码的批次和组号信息。
				var batchID, groupID int
				if err := result.Scan(&batchID, &groupID); err != nil {
					return -1, -1, fmt.Errorf("GetBatchGroupIDByCustomCode scan err"), errs.ErrDBScan
				}
				return batchID, groupID, nil, errs.Success
			} else {
				// 没有找到自定义兑换码的批次和组号信息。
				return -1, -1, fmt.Errorf("GetBatchGroupIDByCustomCode %v not exist", customCode), errs.ErrHandlerCustomCodeNotExist
			}
		}()
		return batchID, groupID, err, errMsg
	}
}

// GetUseInfoByCodeAndBatch 依据兑换码和所在批次，获取使用者信息。
//
// Param-code: 需要查询的兑换码; Param-batchID: 兑换码所在批次; Param-isCustom: 是否为自定义兑换码（自定义码与生成码的存放位置不同）。
//
// Return-map[string]*define.GiftCodeQueryInfo: map[礼包码]礼包码使用信息（使用者列表和是否被销毁）。
func GetUseInfoByCodeAndBatch(code string, batchID int, isCustom bool) (map[string]*define.GiftCodeQueryInfo, error, string) {
	// 获取数据库连接池对象。
	sqlDB := mysql.GetDB()

	if isCustom {
		// 如果自定义兑换码表不存在则创建。
		if err, errMsg := mysql.CreateCustomTableIfNotExist(sqlDB); err != nil {
			return nil, err, errMsg
		}

		// 尝试获取自定义码对应的使用者信息。
		if result, err := sqlDB.Query(mysql.QueryUseInfoByCustomCodeSql(code)); err != nil {
			return nil, fmt.Errorf("GetUseInfoByCodeAndBatch err: %v", err), errs.ErrDBQueryUseInfoByCodeAndBatch
		} else {
			if result, err, errMsg := getUseInfoMapByQueryCodeResult(code, result); err != nil {
				return nil, err, errMsg
			} else {
				return result, nil, errs.Success
			}
		}
	} else {
		// 获取生成表索引。
		index := batchID / config.GenCodeSplitByBatch

		// 如果生成兑换码表不存在则创建。
		if err, errMsg := mysql.CreateGenTableIfNotExist(index, sqlDB); err != nil {
			return nil, err, errMsg
		}

		// 尝试获取生成码对应的使用者信息。
		if result, err := sqlDB.Query(mysql.QueryUseInfoByGenCodeSql(code, index)); err != nil {
			return nil, fmt.Errorf("GetUseInfoByCodeAndBatch err: %v", err), errs.ErrDBQueryUseInfoByCodeAndBatch
		} else {
			if result, err, errMsg := getUseInfoMapByQueryCodeResult(code, result); err != nil {
				return nil, err, errMsg
			} else {
				return result, nil, errs.Success
			}
		}
	}
}

// GetUseInfoByBatchGroup 依据批次和组，获取所有兑换码的使用信息。
//
// Param-batchID: 要查询的目标批次; Param-groupID: 要查询的目标组; Param-isCustom: 查询目标是否为自定义码（自定义码与生成码的存放位置不同）。
//
// Return-map[string]*define.GiftCodeQueryInfo: map[礼包码]礼包码使用信息（使用者列表和是否被销毁）。
func GetUseInfoByBatchGroup(batchID int, groupID int, isCustom bool) (map[string]*define.GiftCodeQueryInfo, error, string) {
	// 获取数据库连接池对象。
	sqlDB := mysql.GetDB()

	if isCustom {
		// 如果自定义兑换码表不存在则创建。
		if err, errMsg := mysql.CreateCustomTableIfNotExist(sqlDB); err != nil {
			return nil, err, errMsg
		}

		// 尝试获取自定义码对应的使用者信息。
		if result, err := sqlDB.Query(mysql.QueryCustomUseInfoByBatchGroup(batchID, groupID)); err != nil {
			return nil, fmt.Errorf("GetUseInfoByBatchGroup err: %v", err), errs.ErrDBQueryUseInfoByBatchGroup
		} else {
			if result, err, errMsg := getUseInfoMapByBatchGroup(result); err != nil {
				return nil, err, errMsg
			} else {
				return result, nil, errs.Success
			}
		}
	} else {
		index := batchID / config.GenCodeSplitByBatch

		// 如果生成兑换码表不存在则创建。
		if err, errMsg := mysql.CreateGenTableIfNotExist(index, sqlDB); err != nil {
			return nil, err, errMsg
		}

		// 尝试获取生成码对应的使用者信息。
		if result, err := sqlDB.Query(mysql.QueryGenUseInfoByBatchGroup(batchID, groupID, index)); err != nil {
			return nil, fmt.Errorf("GetUseInfoByBatchGroup err: %v", err), errs.ErrDBQueryUseInfoByBatchGroup
		} else {
			if result, err, errMsg := getUseInfoMapByBatchGroup(result); err != nil {
				return nil, err, errMsg
			} else {
				return result, nil, errs.Success
			}
		}
	}
}

// GetCodesByBatchGroupID 获取指定批次和组号的所有兑换码。
func GetCodesByBatchGroupID(batchID, groupID int) ([]string, error, string) {
	var giftCodes []string
	// 从OSS数据库中获取兑换码信息。
	codesJson, errCodes, errCodesMsg := oss.DownloadFromOSS(batchID, groupID)
	if errCodes != nil {
		return nil, errCodes, errCodesMsg
	}
	if err := json.Unmarshal(codesJson, &giftCodes); err != nil {
		return nil, fmt.Errorf("GetCodesByBatchGroupID json unmarshal err: %v", err), errs.ErrDBJsonUnmarshal
	}

	// 判断是否从数据库中读取了异常的数据。
	if giftCodes == nil {
		return nil, fmt.Errorf("GetCodesByBatchGroupID batchID: %v, groupID: %v get nil data", batchID, groupID), errs.ErrDBData
	}

	return giftCodes, nil, errs.Success
}

// GetGenInfo 获取已经生成的兑换码的相关信息。
//
// Param-gids: 需要查询的大区列表。
//
// Return-[]*define.GiftCodeInfo: 大区列表可见的所有配置; Return-int: 当前存在的最大的批次号（没有任何批次返回-1）;
// Return-int: 当前存在的最大的批次号下的最大组号（没有任何批次返回-1）。
func GetGenInfo(gids []string) ([]*define.GiftCodeInfo, int, int, error, string) {
	// 获取数据库连接池对象。
	sqlDB := mysql.GetDB()

	tilogs.L().Infof("QueryGenInfoRequest Handle table exist")
	// 如果大区可见性表不存在则创建。
	if err, errMsg := mysql.CreateGidTableIfNotExist(sqlDB); err != nil {
		return nil, -1, -1, err, errMsg
	}

	// 获取大区列表中的所有可见性标识。
	tilogs.L().Infof("QueryGenInfoRequest Handle getVisibilityGids")
	visibilityTags, errGid, errGidMsg := getVisibilityGids(gids, sqlDB)
	if errGid != nil {
		return nil, -1, -1, errGid, errGidMsg
	}

	// 获取大区列表中可见配置。
	tilogs.L().Infof("QueryGenInfoRequest Handle getConfigsByVisibilityTags")
	codeInfoSlice, errConfigs, errConfigMsg := getConfigsByVisibilityTags(visibilityTags, sqlDB)
	if errConfigs != nil {
		return nil, -1, -1, errConfigs, errConfigMsg
	}

	// 获取具有配置表前缀的所有表。
	tilogs.L().Infof("QueryGenInfoRequest Handle getAllTableNamesWithPrefix")
	tables, errTables, errTablesMsg := getAllTableNamesWithPrefix(config.TableGiftCodeConfig, sqlDB)
	if errTables != nil {
		return nil, -1, -1, errTables, errTablesMsg
	}

	// 获取当前存在的最大批次号和该批次号下最大的组。
	tilogs.L().Infof("QueryGenInfoRequest Handle getGreatestBatchID")
	maxBatchID, errBatch, errBatchMsg := getGreatestBatchID(tables, sqlDB)
	if errBatch != nil {
		return nil, -1, -1, errBatch, errBatchMsg
	}

	tilogs.L().Infof("QueryGenInfoRequest Handle getGreatestGroupInBatch")
	maxGroupID, errGroup, errGroupMsg := getGreatestGroupInBatch(tables, maxBatchID, sqlDB)
	if errGroup != nil {
		return nil, -1, -1, errGroup, errGroupMsg
	}

	return codeInfoSlice, maxBatchID, maxGroupID, nil, errs.Success
}

func UpdateUsesInfo(code string, batchID int, newUses []string, isCustom bool) (error, string) {
	resJson, err := json.Marshal(newUses)
	if err != nil {
		return fmt.Errorf("UpdateUsesInfo json marshal err: %v", err), errs.ErrDBJsonMarshal
	}

	// 获取数据库连接池对象。
	sqlDB := mysql.GetDB()

	index := batchID / config.GenCodeSplitByBatch

	if isCustom {
		if err, errMsg := mysql.CreateCustomTableIfNotExist(sqlDB); err != nil {
			return err, errMsg
		}
		if _, err := sqlDB.Exec(mysql.UpdateUsesCustomCodeSql(code, string(resJson))); err != nil {
			return fmt.Errorf("UpdateUsesInfo UpdateUsesCustomCodeSql err: %v", err), errs.ErrDBUpdateCustomUsesInfo
		}
	} else {
		if err, errMsg := mysql.CreateGenTableIfNotExist(index, sqlDB); err != nil {
			return err, errMsg
		}

		if _, err := sqlDB.Exec(mysql.UpdateUsesGenCodeSql(code, string(resJson), index)); err != nil {
			return fmt.Errorf("UpdateUsesInfo UpdateUsesGenCodeSql err: %v", err), errs.ErrDBUpdateGenUsesInfo
		}
	}
	return nil, errs.Success
}

// GetGlobalGidInfo 获取全局Gid配置信息。
func GetGlobalGidInfo() ([]string, error, string) {
	// 获取数据库连接池对象。
	sqlDB := mysql.GetDB()

	// 如果全局配置表不存在则创建。
	if err, errMsg := mysql.CreateGlobalTableIfNotExist(sqlDB); err != nil {
		return nil, err, errMsg
	}

	// 查询当前的所有的配置Gid。
	if result, err := sqlDB.Query(mysql.QueryGlobalGidInfoSql()); err != nil {
		return nil, err, errs.ErrDBQueryGlobalGid
	} else {
		return func() ([]string, error, string) {
			defer func() {
				if err := result.Close(); err != nil {
					tilogs.L().Errorf("close mysql conn err: %v", err)
				}
			}()

			ret := make([]string, 0)
			if result.Next() {
				var gidJson string
				if err := result.Scan(&gidJson); err != nil {
					return nil, err, errs.ErrDBScan
				}

				errUn := json.Unmarshal([]byte(gidJson), &ret)
				if errUn != nil {
					return nil, errUn, errs.ErrDBJsonUnmarshal
				}

				return ret, nil, errs.Success
			} else {
				return ret, nil, errs.Success
			}
		}()
	}
}

// 更新全局Gid配置信息。
func UpdateGlobalGidInfo(gidSlice []string) (error, string) {
	contentJson, errJson := json.Marshal(gidSlice)
	if errJson != nil {
		return errJson, errs.ErrDBJsonMarshal
	}

	// 获取数据库连接池对象。
	sqlDB := mysql.GetDB()

	// 如果全局配置表不存在则创建。
	if err, errMsg := mysql.CreateGlobalTableIfNotExist(sqlDB); err != nil {
		return err, errMsg
	}

	// 更新全局配置中的Gid信息。
	if _, err := sqlDB.Exec(mysql.UpsertGlobalGidInfoSql(string(contentJson))); err != nil {
		return err, errs.ErrDBUpdateGlobalGid
	}

	return nil, errs.Success
}
