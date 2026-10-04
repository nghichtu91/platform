package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nghichtu91/platform/share/x/gift/modules/common/util"

	"github.com/nghichtu91/platform/share/x/gift/db/oss"

	"github.com/nghichtu91/platform/share/x/gift/db/mysql"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/x/gift/config"
	"github.com/nghichtu91/platform/share/x/gift/model/define"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
)

const (
	GenMode    = false // 插入兑换码时采用生成的模式（需要修改配置）。
	AppendMode = true  // 插入兑换码时采用追加的模式（无需修改配置）。
)

const (
	Duplicate = "Duplicate"
)

// opCodeConfigAndGidTransaction 以事务的形式执行判断、更新兑换码配置的操作。
func opCodeConfigAndGidTransaction(codes []string, codeConfig *define.GiftCodeInfo, index int, mode bool, sqlTS *sql.Tx) (string, error, string) {
	if mode == GenMode {
		// 执行生成时的兑换码配置表相关的操作。
		codesJson, err, errMsg := opGiftCodeConfig(codes, codeConfig, index, sqlTS)
		if err != nil {
			return errs.InvalidCodesJson, err, errMsg
		}

		// 执行生成时的大区可见性表相关的操作。
		if err, errMsg := opGenGidVisibility(codeConfig, sqlTS); err != nil {
			return errs.InvalidCodesJson, err, errMsg
		}

		return codesJson, nil, errs.Success
	} else {
		// 判断获取生成时或上次追加时的批次和组的信息。
		giftCodeConfig, giftCodes, errQuery, errQueryMsg := appendQuery(codeConfig, index, sqlTS)
		if errQuery != nil {
			return errs.InvalidCodesJson, errQuery, errQueryMsg
		}

		// 执行追加时的兑换码配置表相关的操作。
		codesJson, err, errMsg := opAppendCodeConfig(giftCodes, codes, giftCodeConfig, codeConfig, index, sqlTS)
		if err != nil {
			return errs.InvalidCodesJson, err, errMsg
		}

		return codesJson, nil, errs.Success
	}
}

// opGiftCodeConfig 生成时对兑换码配置表执行相关操作。
//
// Param-codes: 本次生成的所有字符码; Param-codeConfig: 本次的生成配置信息;
// Param-index: 需要操作的配置表的索引; Param-sqlTS: 事务执行的承载。
func opGiftCodeConfig(codes []string, codeConfig *define.GiftCodeInfo, index int, sqlTS *sql.Tx) (string, error, string) {
	// 生成时，总数量等于本次生成数量。
	codeConfig.TotalCount = codeConfig.GenCount

	// 结果JsonMarshal。
	configResultJson, codesResultJson, jsonErr, jsonErrMsg := configAndCodes2Json(codeConfig, codes)
	if jsonErr != nil {
		return errs.InvalidCodesJson, jsonErr, jsonErrMsg
	}

	// 将更新后的数据写入数据库。
	if _, err := sqlTS.Exec(mysql.InsertGiftCodeConfigSql(string(configResultJson), codeConfig.BatchID, codeConfig.GroupID, index)); err != nil {
		return errs.InvalidCodesJson, fmt.Errorf("opGiftCodeConfig InsertGiftCodeConfigSql err: %v", err), errs.ErrDBInsertGiftCodeConfig
	}

	return string(codesResultJson), nil, errs.Success
}

// opAppendCodeConfig 追加时的对兑换码配置表执行操作。
//
// Param-codesOld: 本次追加前已经存在的兑换码; Param-codesNew: 本次追加的兑换码; Param-codeConfigOld: 本次追加前的兑换码配置;
// Param-codeConfigNew: 本次追加时采用的生成配置; Param-index: 需要操作的配置表的索引; Param-sqlTS: 事务执行的承载。
func opAppendCodeConfig(codesOld []string, codesNew []string, codeConfigOld *define.GiftCodeInfo, codeConfigNew *define.GiftCodeInfo, index int, sqlTS *sql.Tx) (string, error, string) {
	// 获取更新后的配置信息（生成时间和自定义兑换码由第一次生成确定，不允许修改）。
	totalCount := codeConfigOld.TotalCount + codeConfigNew.GenCount
	codeConfigOld.GenCount = codeConfigNew.GenCount
	codeConfigOld.TotalCount = totalCount

	// 将追加的兑换码加入原码集。
	codesOld = append(codesOld, codesNew...)

	// 结果JsonMarshal。
	configResultJson, codesResultJson, jsonErr, jsonErrMsg := configAndCodes2Json(codeConfigOld, codesOld)
	if jsonErr != nil {
		return errs.InvalidCodesJson, jsonErr, jsonErrMsg
	}

	// 将更新后的信息写入数据库。
	if err, errMsg := updateGiftCodeConfig(string(configResultJson), codeConfigNew.BatchID, codeConfigNew.GroupID, index, sqlTS); err != nil {
		return errs.InvalidCodesJson, err, errMsg
	}

	return string(codesResultJson), nil, errs.Success
}

// updateGiftCodeConfig 更新礼包码的配置信息。
//
// Param-configJson: JsonMarshal后的配置信息>
func updateGiftCodeConfig(configJson string, batchID, groupID, index int, sqlTS *sql.Tx) (error, string) {
	// 将更新后的信息写入数据库。
	if _, err := sqlTS.Exec(mysql.UpdateGiftCodeConfigSql(configJson, batchID, groupID, index)); err != nil {
		return fmt.Errorf("updateGiftCodeConfig update err: %v", err), errs.ErrDBUpdateGiftCodeConfig
	}

	return nil, errs.Success
}

// opGenGidVisibility 执行生成对兑换码大区可见性表执行操作。
//
// Param-codeConfig: 新的生成配置信息; Param-sqlTS: 承载事务操作。
func opGenGidVisibility(codeConfig *define.GiftCodeInfo, sqlTS *sql.Tx) (error, string) {
	// 获取批次、组号对应可见性标识。
	tag := GetVisibilityTag(codeConfig.BatchID, codeConfig.GroupID)

	// 向可见性配置表中加入新的可见性标识。
	if err, errMsg := addVisibilityGids(codeConfig.Gid, tag, sqlTS); err != nil {
		return err, errMsg
	}

	return nil, errs.Success
}

// updateGidVisibility 通过新旧配置比较更新大区可见性表。
//
// Param-codeConfigOld: 数据库中旧的生成配置信息; Param-codeConfigNew: 新的生成配置信息;
// Param-sqlTS: 承载事务操作。
func updateGidVisibility(codeConfigOld *define.GiftCodeInfo, codeConfigNew *define.GiftCodeInfo, sqlTS *sql.Tx) (error, string) {
	// 获取批次、组号对应可见性标识。
	tag := GetVisibilityTag(codeConfigNew.BatchID, codeConfigNew.GroupID)

	// 获取有变化的大区信息（新增与删除）。
	addGids, deleteGids := getVisibilityChangedGid(codeConfigOld, codeConfigNew)

	// 向可见性配置表中加入新的可见性标识。
	if err, errMsg := addVisibilityGids(addGids, tag, sqlTS); err != nil {
		return err, errMsg
	}

	// 从可见性配置表中删除旧的可见性标识。
	if err, errMsg := deleteVisibilityGid(deleteGids, tag, sqlTS); err != nil {
		return err, errMsg
	}

	return nil, errs.Success
}

// getVisibilityChangedGid 获取有变化的大区信息。
//
// Note: 函数复杂度为O(N^2)，可以采用Map[Gid]struct{}构建的方式降低到O(N)。
// 由于通常情况下大区不会很多，此处O(N^2)认为处于可以接受的范围内。
//
// Return-[]string: 相较旧的配置增加的Gid; Return-[]string: 相较于旧的配置减少的Gid。
func getVisibilityChangedGid(codeConfigOld *define.GiftCodeInfo, codeConfigNew *define.GiftCodeInfo) ([]string, []string) {
	addGids := make([]string, 0)
	deleteGids := make([]string, 0)

	// 获取新增的Gid集合。
	for _, newGid := range codeConfigNew.Gid {
		// 在旧的Gids中查找新的Gid是否存在。
		find := false
		for _, oldGid := range codeConfigOld.Gid {
			if oldGid == newGid {
				find = true
				break
			}
		}

		// 如果不存在代表是新增的Gid
		if !find {
			addGids = append(addGids, newGid)
		}
	}

	// 获取减少的Gid集合。
	for _, oldGid := range codeConfigOld.Gid {
		// 在新的Gids中查找旧的Gid是否存在。
		find := false
		for _, newGid := range codeConfigNew.Gid {
			if oldGid == newGid {
				find = true
				break
			}
		}

		// 如果不存在代表是减少的Gid
		if !find {
			deleteGids = append(deleteGids, oldGid)
		}
	}

	return addGids, deleteGids
}

// deleteVisibilityGid 从可见性配置表中删除可见性标识。
func deleteVisibilityGid(gids []string, tag int, sqlTS *sql.Tx) (error, string) {
	// 对需要删除标识的大区逐一操作（修改）。
	for _, gid := range gids {
		var visibility []int
		if result, err := sqlTS.Query(mysql.QueryGidVisibilitySql(gid)); err != nil {
			return fmt.Errorf("deleteVisibilityGid query gid err: %v", err), errs.ErrDBQueryGidVisibility
		} else {
			// 此处不能隐式关闭（没有获取到终止）,执行显示关闭。
			if err, errMsg := func() (error, string) {
				defer func() {
					if err := result.Close(); err != nil {
						tilogs.L().Errorf("close mysql conn err: %v", err)
					}
				}()

				if result.Next() {
					// 如果找到则需要在原来的基础上修改。
					var visibilityJson string
					if err := result.Scan(&visibilityJson); err != nil {
						return fmt.Errorf("deleteVisibilityGid gid: %v scan err", gid), errs.ErrDBScan
					}
					// 将Json字符串转换为对应的数据结构。
					if errVisibility := json.Unmarshal([]byte(visibilityJson), &visibility); errVisibility != nil {
						return fmt.Errorf("deleteVisibilityGid json unmarshal err: %v", errVisibility), errs.ErrDBJsonUnmarshal
					}
					if visibility == nil {
						return fmt.Errorf("deleteVisibilityGid gid: %v get nil data", gid), errs.ErrDBData
					}
					// 将可见性标志从对应大区的可见性列表删除。
					for index, oldTag := range visibility {
						if oldTag == tag {
							visibility = append(visibility[:index], visibility[index+1:]...)
							break
						}
					}

					// 获取删除可见性标识的JsonMarshal结果。
					resultBytes, errJson := json.Marshal(visibility)
					if errJson != nil {
						return fmt.Errorf("deleteVisibilityGid json marshal err: %v", errJson), errs.ErrDBJsonMarshal
					}

					// 关闭上一个Sql连接。
					if err := result.Close(); err != nil {
						return fmt.Errorf("deleteVisibilityGid close sql conn err: %v", err), errs.ErrDBCloseSqlConn
					}

					// 将JsonMarshal后的更新结果写回数据库。
					if _, err := sqlTS.Exec(mysql.UpdateGidVisibilitySql(gid, string(resultBytes))); err != nil {
						return fmt.Errorf("deleteVisibilityGid updateGidVisibilitySql exec err: %v", err), errs.ErrDBUpdateGidVisibility
					}
				}

				return nil, errs.Success
			}(); err != nil {
				return err, errMsg
			}
		}
	}

	return nil, errs.Success
}

// getAllTableNamesWithPrefix 获取所有以 prefix 为前缀的表名。
func getAllTableNamesWithPrefix(prefix string, sqlDB *sql.DB) ([]string, error, string) {
	resTables := make([]string, 0)

	if rows, err := sqlDB.Query(mysql.ShowTablesLike(prefix)); err != nil {
		return nil, fmt.Errorf("getAllTableNamesWithPrefix query err: %v", err), errs.ErrDBShowTablesWithPrefix
	} else {
		// 隐式关闭。
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				return nil, fmt.Errorf("getAllTableNamesWithPrefix scan err:%v", err), errs.ErrDBScan
			}
			resTables = append(resTables, tableName)
		}
	}
	return resTables, nil, errs.Success
}

func getGreatestBatchID(configTables []string, sqlDB *sql.DB) (int, error, string) {
	// 尚未生成配置表。
	if len(configTables) == 0 {
		return errs.IDNotGen, nil, errs.Success
	}

	if rows, err := sqlDB.Query(mysql.QueryGreatestBatch(configTables)); err != nil {
		return -1, fmt.Errorf("getGreatestBatch query err:%v", err), errs.ErrDBQueryGreatest
	} else {
		return func() (int, error, string) {
			if rows.Next() {
				var maxBatchID int
				if err := rows.Scan(&maxBatchID); err != nil {
					return -1, fmt.Errorf("getGreatestBatchID scan err: %v", maxBatchID), errs.ErrDBScan
				}
				return maxBatchID, nil, errs.Success
			} else {
				// 配置表均为空表。
				return errs.IDNotGen, nil, errs.Success
			}
		}()
	}
}

func getGreatestGroupInBatch(configTables []string, batchID int, sqlDB *sql.DB) (int, error, string) {
	// 尚未生成配置表。
	if len(configTables) == 0 {
		return errs.IDNotGen, nil, errs.Success
	}

	// 还未生成批次的情况下，组号必然不存在。
	if batchID < 0 {
		return errs.IDNotGen, nil, errs.Success
	}

	if rows, err := sqlDB.Query(mysql.QueryGreatestGroupInBatch(configTables, batchID)); err != nil {
		return -1, fmt.Errorf("getGreatestGroupInBatch query err:%v", err), errs.ErrDBQueryGreatest
	} else {
		return func() (int, error, string) {
			if rows.Next() {
				var maxGroupID int
				if err := rows.Scan(&maxGroupID); err != nil {
					return -1, fmt.Errorf("getGreatestBatchID scan err: %v", maxGroupID), errs.ErrDBScan
				}
				return maxGroupID, nil, errs.Success
			} else {
				return -1, nil, errs.Success
			}
		}()
	}
}

// getConfigsByVisibilityTags 依据可见性标识集合获取相应的配置信息。
func getConfigsByVisibilityTags(tags []int, sqlDB *sql.DB) ([]*define.GiftCodeInfo, error, string) {
	resSlice := make([]*define.GiftCodeInfo, 0)

	for _, tag := range tags {
		batchID, groupID := GetBatchGroupByVisibility(tag)
		index := batchID / config.GiftCodeConfigSplitByBatch

		if err, errMsg := mysql.CreateConfigTableIfNotExist(index, sqlDB); err != nil {
			return nil, err, errMsg
		}

		if result, err := sqlDB.Query(mysql.QueryGiftCodeConfigSql(batchID, groupID, index)); err != nil {
			return nil, fmt.Errorf("getGiftCodeConfigsByVisibilityTags query config err: %v", err), errs.ErrDBQueryGiftCodeConfig
		} else {
			configInfo, err, errMsg := func() (*define.GiftCodeInfo, error, string) {
				defer func() {
					if err := result.Close(); err != nil {
						tilogs.L().Errorf("close mysql conn err: %v", err)
					}
				}()

				if result.Next() {
					// 读取以Json格式存储的兑换码配置。
					var giftCodeConfigJson string
					if err := result.Scan(&giftCodeConfigJson); err != nil {
						return nil, fmt.Errorf("getConfigsByVisibilityTags batchID: %v, groupID: %v scan err", batchID, groupID), errs.ErrDBScan
					}

					// 将Json字符串转换为对应的数据结构。
					var giftCodeConfig *define.GiftCodeInfo
					if errConfig := json.Unmarshal([]byte(giftCodeConfigJson), &giftCodeConfig); errConfig != nil {
						return nil, fmt.Errorf("getConfigsByVisibilityTags json unmarshal err: %v", errConfig), errs.ErrDBJsonUnmarshal
					}

					// 判断是否从数据库中读取了异常的数据
					if giftCodeConfig == nil {
						return nil, fmt.Errorf("getConfigsByVisibilityTag batchID: %v, groupID: %v get nil data", batchID, groupID), errs.ErrDBData
					}

					return giftCodeConfig, nil, errs.Success
				} else {
					return nil, fmt.Errorf("getConfigsByVisibilityTags batchID: %v, groupID: %v not found", batchID, groupID), errs.ErrDBBatchIDGroupIDNotInUsed
				}
			}()
			if err != nil {
				return nil, err, errMsg
			}

			resSlice = append(resSlice, configInfo)
		}
	}

	return resSlice, nil, errs.Success
}

// getVisibilityGids 获取可见性配置表中的可见性信息。
func getVisibilityGids(gids []string, sqlDB *sql.DB) ([]int, error, string) {
	resSlice := make([]int, 0)
	for _, gid := range gids {
		visibility := make([]int, 0)
		// 获取对应大区的可见性信息。
		if result, err := sqlDB.Query(mysql.QueryGidVisibilitySql(gid)); err != nil {
			return nil, fmt.Errorf("getVisibilityGids query gid err: %v", err), errs.ErrDBQueryGidVisibility
		} else {
			if err, errMsg := func() (error, string) {
				defer func() {
					if err := result.Close(); err != nil {
						tilogs.L().Errorf("close mysql conn err: %v", err)
					}
				}()

				if result.Next() {
					var visibilityJson string
					if err := result.Scan(&visibilityJson); err != nil {
						return fmt.Errorf("getVisibilityGids gid: %v scan err", gid), errs.ErrDBScan
					}
					// 将Json字符串转换为对应的数据结构。
					if errVisibility := json.Unmarshal([]byte(visibilityJson), &visibility); errVisibility != nil {
						return fmt.Errorf("getVisibilityGids json unmarshal err: %v", errVisibility), errs.ErrDBJsonUnmarshal
					}

					resSlice = util.MergeSliceNoRepetition(resSlice, visibility)
				}
				return nil, errs.Success
			}(); err != nil {
				return nil, err, errMsg
			}
		}
	}
	return resSlice, nil, errs.Success
}

// addVisibilityGids 向可见性配置表中加入新的可见性标识。
func addVisibilityGids(gids []string, tag int, sqlTS *sql.Tx) (error, string) {
	// 对需要加入标识的大区逐一操作（新建或修改）。
	for _, gid := range gids {
		var visibility []int
		// 检查对应的大区可见性是否已存在。
		if result, err := sqlTS.Query(mysql.QueryGidVisibilitySql(gid)); err != nil {
			return fmt.Errorf("addVisibilityGids query gid err: %v", err), errs.ErrDBQueryGidVisibility
		} else {
			// 此处不能隐式关闭（没有获取到终止）,执行显示关闭。
			if err, errMsg := func() (error, string) {
				defer func() {
					if err := result.Close(); err != nil {
						tilogs.L().Errorf("close mysql conn err: %v", err)
					}
				}()

				if result.Next() {
					// 如果找到则需要在原来的基础上修改。
					var visibilityJson string
					if err := result.Scan(&visibilityJson); err != nil {
						return fmt.Errorf("addVisibilityGids gid: %v scan err", gid), errs.ErrDBScan
					}
					// 将Json字符串转换为对应的数据结构。
					if errVisibility := json.Unmarshal([]byte(visibilityJson), &visibility); errVisibility != nil {
						return fmt.Errorf("addVisibilityGids json unmarshal err: %v", errVisibility), errs.ErrDBJsonUnmarshal
					}

					if visibility == nil {
						return fmt.Errorf("addVisibilityGids gid: %v get nil data", gid), errs.ErrDBData
					}

					// 将可见性标志加入对应大区的可见性列表。
					visibility = append(visibility, tag)

					// 获取插入新的可见性标记后的JsonMarshal结果。
					resultBytes, errJson := json.Marshal(visibility)
					if errJson != nil {
						return fmt.Errorf("addVisibilityGids json marshal err: %v", errJson), errs.ErrDBJsonMarshal
					}

					// 关闭上一个Sql连接。
					if err := result.Close(); err != nil {
						return fmt.Errorf("addVisibilityGids close sql conn err: %v", err), errs.ErrDBCloseSqlConn
					}

					// 将JsonMarshal后的更新结果写回数据库。
					if _, err := sqlTS.Exec(mysql.UpdateGidVisibilitySql(gid, string(resultBytes))); err != nil {
						return fmt.Errorf("addVisibilityGidsn updateGidVisibilitySql exec err: %v", err), errs.ErrDBUpdateGidVisibility
					}
				} else {
					// 若没有找到则需要新增一条大区可见。
					visibility = make([]int, 0, 1)
					visibility = append(visibility, tag)

					// 获取插入新的可见性标记后的JsonMarshal结果。
					resultBytes, errJson := json.Marshal(visibility)
					if errJson != nil {
						return fmt.Errorf("addVisibilityGids json marshal err: %v", errJson), errs.ErrDBJsonMarshal
					}

					// 关闭上一个Sql连接。
					if err := result.Close(); err != nil {
						return fmt.Errorf("addVisibilityGids close sql conn err: %v", err), errs.ErrDBCloseSqlConn
					}

					// 将JsonMarshal后的新增结果写入数据库。
					if _, err := sqlTS.Exec(mysql.InsertGidVisibilitySql(gid, string(resultBytes))); err != nil {
						return fmt.Errorf("addVisibilityGids insertGidVisibilitySql exec err: %v", err), errs.ErrDBInsertGidVisibility
					}
				}
				return nil, errs.Success
			}(); err != nil {
				return err, errMsg
			}
		}
	}

	return nil, errs.Success
}

// appendQuery 追加兑换码更新配置时检查是否已生成，并获取信息。
//
// Note: 若没有对应批次、组，说明还未生成，不允许追加，直接返回异常和异常信息。
func appendQuery(codeConfig *define.GiftCodeInfo, index int, sqlTS *sql.Tx) (*define.GiftCodeInfo, []string, error, string) {
	giftCodeConfig := &define.GiftCodeInfo{}
	giftCodes := make([]string, 0)

	// 依据批次号、组号、索引号获取旧的配置信息。
	if result, err := sqlTS.Query(mysql.QueryGiftCodeConfigSql(codeConfig.BatchID, codeConfig.GroupID, index)); err != nil {
		return nil, nil, fmt.Errorf("opCodeConfigAndGidTransaction query old config err: %v", err), errs.ErrDBQueryGiftCodeConfig
	} else {
		// 此处不能隐式关闭（没有获取到终止）,执行显示关闭。
		if err, errMsg := func() (error, string) {
			defer func() {
				if err := result.Close(); err != nil {
					tilogs.L().Errorf("close mysql conn err: %v", err)
				}
			}()

			if result.Next() {
				// 读取以Json格式存储的兑换码配置。
				var giftCodeConfigJson string
				if err := result.Scan(&giftCodeConfigJson); err != nil {
					return fmt.Errorf("opCodeConfigAndGidTransaction batchID: %v, groupID: %v scan err", codeConfig.BatchID, codeConfig.GroupID), errs.ErrDBScan
				}

				// 将Json字符串转换为对应的数据结构。
				if errConfig := json.Unmarshal([]byte(giftCodeConfigJson), &giftCodeConfig); errConfig != nil {
					return fmt.Errorf("opCodeConfigAndGidTransaction json unmarshal err: %v", errConfig), errs.ErrDBJsonUnmarshal
				}
			} else {
				return fmt.Errorf("opCodeConfigAndGidTransaction batchID: %v, groupID: %v not found", codeConfig.BatchID, codeConfig.GroupID), errs.ErrDBBatchIDGroupIDNotInUsed
			}
			return nil, errs.Success
		}(); err != nil {
			return nil, nil, err, errMsg
		}
	}

	// 从OSS数据库中获取兑换码信息。
	codesJson, errCodes, errCodesMsg := oss.DownloadFromOSS(codeConfig.BatchID, codeConfig.GroupID)
	if errCodes != nil {
		return nil, nil, errCodes, errCodesMsg
	}
	if err := json.Unmarshal(codesJson, &giftCodes); err != nil {
		return nil, nil, fmt.Errorf("opCodeConfigAndGidTransaction json unmarshal err: %v", err), errs.ErrDBJsonUnmarshal
	}

	return giftCodeConfig, giftCodes, nil, errs.Success
}

// configAndCodes2Json 将配置和码集Json化。
func configAndCodes2Json(giftCodeConfig *define.GiftCodeInfo, giftCodes []string) ([]byte, []byte, error, string) {
	configResultJson, errConfig := json.Marshal(giftCodeConfig)
	if errConfig != nil {
		return nil, nil, fmt.Errorf("configAndCodes2Json config json marshal err: %v", errConfig), errs.ErrDBJsonMarshal
	}
	codesResultJson, errCodes := json.Marshal(giftCodes)
	if errCodes != nil {
		return nil, nil, fmt.Errorf("configAndCodes2Json codes json marshal err: %v", errCodes), errs.ErrDBJsonMarshal
	}

	return configResultJson, codesResultJson, nil, errs.Success
}

// insertGiftCodeTransaction 以事务的形式执行插入兑换码的操作。
//
// Note: 生成和追加都是直接插入，生成码自身保证唯一性。
//
// Param-codes: 需要执行插入的兑换码; Param-index: 需要插入到目标表的索引; Param-sqlTS: 承载本次操作的Sql事务。
func insertGiftCodeTransaction(codes []string, codeConfig *define.GiftCodeInfo, index int, sqlTS *sql.Tx) (error, string) {
	// 判断当前码的类别以确定应插入到哪张表。
	isCustomCode := false
	if codeConfig.CustomCode != "" {
		isCustomCode = true
	}

	// 空的使用者列表的JsonMarshal信息。
	users := make([]string, 0)
	usersResult, errResult := json.Marshal(users)
	if errResult != nil {
		return fmt.Errorf("insertGiftCodeTransaction json marshal err: %v", errResult), errs.ErrDBJsonMarshal
	}

	if isCustomCode {
		// 执行自定义兑换码插入的操作。
		if _, err := sqlTS.Exec(mysql.InsertCustomCodeSql(codes[0], string(usersResult), codeConfig.BatchID, codeConfig.GroupID)); err != nil {
			return fmt.Errorf("insertGiftCodeTransaction err: %v", err), errs.ErrDBInsertCustomCode
		}
	} else {
		// 生成码的插入操作。
		for {
			if len(codes) == 0 {
				break
			}

			// 获取本轮插入的兑换码集合。
			tempCodes := make([]string, 0, config.InsertGenCodeBatchNum)
			if len(codes) >= config.InsertGenCodeBatchNum {
				tempCodes = codes[:config.InsertGenCodeBatchNum]
				codes = codes[config.InsertGenCodeBatchNum:]
			} else {
				tempCodes = codes
				codes = make([]string, 0, config.InsertGenCodeBatchNum)
			}

			// 执行本轮兑换码插入的操作。
			if _, err := sqlTS.Exec(mysql.BatchInsertGenCodeSql(tempCodes, string(usersResult), codeConfig.BatchID, codeConfig.GroupID, index)); err != nil {
				return fmt.Errorf("insertGiftCodeTransaction err: %v", err), errs.ErrDBInsertGenCode
			}
		}
	}
	return nil, errs.Success
}

func getUseInfoMapByQueryCodeResult(code string, result *sql.Rows) (map[string]*define.GiftCodeQueryInfo, error, string) {
	codeUseMap := make(map[string]*define.GiftCodeQueryInfo)

	// result隐式关闭。
	for result.Next() {
		// 读取以Json格式存储的兑换码使用者信息。
		var hasDestroy bool
		var usersJson string
		if err := result.Scan(&usersJson, &hasDestroy); err != nil {
			return nil, fmt.Errorf("GetUseInfoByCodeAndBatch scan err: %v", err), errs.ErrDBScan
		}

		// 将Json字符串转换为对应的数据结构。
		var users []string
		if errUsers := json.Unmarshal([]byte(usersJson), &users); errUsers != nil {
			return nil, fmt.Errorf("GetUseInfoByCodeAndBatch json unmarshal err: %v", errUsers), errs.ErrDBJsonUnmarshal
		}

		if users == nil {
			return nil, fmt.Errorf("GetUseInfoByCodeAndBatch data is nil"), errs.ErrDBData
		}
		codeUseMap[code] = &define.GiftCodeQueryInfo{
			Users:      users,
			HasDestroy: hasDestroy,
		}
	}

	return codeUseMap, nil, errs.Success
}

func getUseInfoMapByBatchGroup(result *sql.Rows) (map[string]*define.GiftCodeQueryInfo, error, string) {
	codeUseMap := make(map[string]*define.GiftCodeQueryInfo)

	// result隐式关闭。
	for result.Next() {
		// 读取以Json格式存储的兑换码使用者信息。
		var giftCode string
		var hasDestroy bool
		var usersJson string
		if err := result.Scan(&giftCode, &hasDestroy, &usersJson); err != nil {
			return nil, fmt.Errorf("GetUseInfoByCodeAndBatch scan err: %v", err), errs.ErrDBScan
		}
		if giftCode == "" {
			return nil, fmt.Errorf("GetUseInfoByCodeAndBatch data is nil"), errs.ErrDBData
		}

		// 将Json字符串转换为对应的数据结构。
		var users []string
		if errUsers := json.Unmarshal([]byte(usersJson), &users); errUsers != nil {
			return nil, fmt.Errorf("GetUseInfoByCodeAndBatch json unmarshal err: %v", errUsers), errs.ErrDBJsonUnmarshal
		}

		if users == nil {
			return nil, fmt.Errorf("GetUseInfoByCodeAndBatch data is nil"), errs.ErrDBData
		}

		codeUseMap[giftCode] = &define.GiftCodeQueryInfo{
			Users:      users,
			HasDestroy: hasDestroy,
		}
	}

	return codeUseMap, nil, errs.Success
}

// destroyCustomCodesTransaction 以事务的形式执行自定义码的销毁操作。
func destroyCustomCodesTransaction(codes []string, sqlTS *sql.Tx) (error, string) {
	// 逐一销毁兑换码，如果某一兑换码销毁失败，则全部回滚。
	for _, code := range codes {
		if _, err := sqlTS.Exec(mysql.DestroyCustomCodeSql(code)); err != nil {
			return fmt.Errorf("destroyCustomCodesTransaction %v destroy err: %v", code, err), errs.ErrHandlerDestroyCustomCode
		}
	}

	return nil, errs.Success
}

// destroyGenCodesTransaction 以事务的形式执行生成码的销毁操作。
func destroyGenCodesTransaction(codes []string, index int, sqlTS *sql.Tx) (error, string) {
	// 逐一销毁兑换码，如果某一兑换码销毁失败，则全部回滚。
	for _, code := range codes {
		if _, err := sqlTS.Exec(mysql.DestroyGenCodeSql(code, index)); err != nil {
			return fmt.Errorf("destroyGenCodesTransaction %v destroy err: %v", code, err), errs.ErrHandlerDestroyGenCode
		}
	}

	return nil, errs.Success
}

// GetVisibilityTag 通过批次ID和组号ID获取可见性唯一标识。
//
// Return-int: 可见性标识。
func GetVisibilityTag(batchID, groupID int) int {
	return batchID*config.CodeGroupCoding + groupID
}

// GetBatchGroupByVisibility 通过可见性唯一标识获取批次号和组号。
//
// Param-visibility: 可见性标识。
//
// Return-int: 批次号; Return-int: 组号。
func GetBatchGroupByVisibility(visibility int) (int, int) {
	return visibility / config.CodeGroupCoding, visibility % config.CodeGroupCoding
}
