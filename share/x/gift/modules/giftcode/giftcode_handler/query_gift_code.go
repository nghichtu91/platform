package giftcode_handler

import (
	"fmt"
	"regexp"

	"github.com/nghichtu91/platform/share/x/gift/modules/common/handler"

	"github.com/nghichtu91/platform/share/x/gift/config"
	"github.com/nghichtu91/platform/share/x/gift/db"
	"github.com/nghichtu91/platform/share/x/gift/model/define"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
)

// 查询兑换码信息。

type QueryGiftCodeRequest struct {
	GiftCode string `form:"gift_code" bson:"gift_code" json:"gift_code"` // 要查询的指定兑换码，为空时按照批次和组号查询。
	BatchID  int    `form:"batch_id" bson:"batch_id" json:"batch_id"`    // 需要查询的批次。
	GroupID  int    `form:"group_id" bson:"group_id" json:"group_id"`    // 需要查询的组号。
}

type QueryGiftCodeResponse struct {
	GiftCodeConfig *define.GiftCodeInfo                 `form:"gift_code_info" bson:"gift_code_info" json:"gift_code_info"` // 兑换码的生成配置信息。
	CodeUsers      map[string]*define.GiftCodeQueryInfo `form:"code_users" bson:"code_users" json:"code_users"`             // 兑换码的使用者信息。
}

func (req *QueryGiftCodeRequest) Handle() (interface{}, error, string) {
	// 检查目标批次和组号是否存在。
	exist, codeConfig, errExist, errExistMsg := db.JudgeBatchGroupExist(req.BatchID, req.GroupID)
	if errExist != nil {
		return nil, errExist, errExistMsg
	}
	if !exist {
		return nil, fmt.Errorf("QueryGiftCodeRequest cannot find"), errs.ErrHandlerTargetNotExist
	}

	// 判断查询的目标是生成码还是自定义码。
	var result map[string]*define.GiftCodeQueryInfo
	var err error
	var errMsg string
	isCustom, _ := regexp.Match(config.RegexpContainNum, []byte(codeConfig.CustomCode))
	if req.GiftCode != "" {
		// 按单一指定码查询。
		if result, err, errMsg = db.GetUseInfoByCodeAndBatch(req.GiftCode, req.BatchID, isCustom); err != nil {
			return nil, err, errMsg
		}
	} else {
		// 按批次和组号查询。
		if result, err, errMsg = db.GetUseInfoByBatchGroup(req.BatchID, req.GroupID, isCustom); err != nil {
			return nil, err, errMsg
		}
	}

	if result == nil || len(result) == 0 {
		return nil, fmt.Errorf("QueryGiftCodeRequest cannot find info"), errs.ErrHandlerCannotFindInfo
	}

	resp := &QueryGiftCodeResponse{
		GiftCodeConfig: codeConfig,
		CodeUsers:      result,
	}

	return resp, nil, errs.Success
}

func (req *QueryGiftCodeRequest) GetDistinguish() (int, error, string) {
	if req.GiftCode == "" {
		// 如果查询目标为批次和组。
		return db.GetVisibilityTag(req.BatchID, req.GroupID), nil, errs.Success
	} else {
		isCustom, _ := regexp.Match(config.RegexpContainNum, []byte(req.GiftCode))
		if isCustom {
			// 如果查询目标为自定义兑换码。
			batchID, groupID, err, errMsg := db.GetBatchGroupIDByCustomCode(req.GiftCode)
			if err != nil {
				return -1, err, errMsg
			}

			req.BatchID = batchID
			req.GroupID = groupID

			return db.GetVisibilityTag(batchID, groupID), nil, errs.Success
		} else {
			// 如果查询目标为生成兑换码。
			batchID, groupID, err, errMsg := handler.GetBatchGroupIDByCharCoding(req.GiftCode)
			if err != nil {
				return -1, err, errMsg
			}

			req.BatchID = batchID
			req.GroupID = groupID

			return db.GetVisibilityTag(batchID, groupID), nil, errs.Success
		}

	}
}
