package giftcode_handler

import (
	"fmt"
	"regexp"

	"github.com/nghichtu91/platform/share/x/gift/modules/common/handler"

	"github.com/nghichtu91/platform/share/x/gift/config"

	"github.com/nghichtu91/platform/share/x/gift/db"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
	"github.com/nghichtu91/platform/share/x/gift/modules/common/util"
)

// 销毁指定兑换码。

type DestroyGiftCodeRequest struct {
	BatchID int    `form:"batch_id" bson:"batch_id" json:"batch_id"` // 需要销毁兑换码的批次。
	GroupID int    `form:"group_id" bson:"group_id" json:"group_id"` // 需要销毁兑换码的组号。
	Data    []byte `form:"data" bson:"data" json:"data"`             // 需要销毁的兑换码。
	Num     int    `form:"num" bson:"num" json:"num"`                // 需要销毁的数量。
}

type DestroyGiftCodeResponse struct {
}

func (req *DestroyGiftCodeRequest) Handle() (interface{}, error, string) {
	// 判断要销毁的兑换码合集是否满足设定。
	if req.Data == nil {
		return nil, fmt.Errorf("DestroyGiftCodeRequest not destroy codes"), errs.ErrHandlerDestroyContent
	}
	split := util.Bytes2Codes(req.Data)
	codes := split[:len(split)-1]
	if len(codes) == 0 {
		return nil, fmt.Errorf("DestroyGiftCodeRequest not destroy codes"), errs.ErrHandlerDestroyContent
	}
	if len(codes) != req.Num {
		return nil, fmt.Errorf("DestroyGiftCodeRequest codes num %v not match request %v", len(codes), req.Num), errs.ErrHandlerDestroyCodesNotMatchRequest
	}
	repeated, code := judgeRepeated(codes)
	if repeated {
		return nil, fmt.Errorf("DestroyGiftCodeRequest repeated code: %v", code), errs.ErrHandlerDestroyRepeated
	}

	// 获取要销毁的兑换码的配置信息。
	exist, codeConfig, err, errMsg := db.JudgeBatchGroupExist(req.BatchID, req.GroupID)
	if err != nil {
		return nil, err, errMsg
	}
	if !exist {
		return nil, fmt.Errorf("DestroyGiftCodeRequest Limit batchID: %v, groupID: %v not exist", req.BatchID, req.GroupID), errs.ErrHandlerDestroyLimitNotExist
	}

	// 判断要销毁的兑换码是否为自定义兑换码。
	isCustom, _ := regexp.Match(config.RegexpContainNum, []byte(codeConfig.CustomCode))

	// 验证销毁兑换码的批次和组是否满足限定。
	if err, errMsg := judgeCodesBatchGroupLimit(codes, req.BatchID, req.GroupID, isCustom); err != nil {
		return nil, err, errMsg
	}

	// 执行销毁操作。
	if err, errMsg := db.DestroyGiftCodes(codes, req.BatchID, isCustom); err != nil {
		return nil, err, errMsg
	}

	return &DestroyGiftCodeResponse{}, nil, errs.Success
}

func (req *DestroyGiftCodeRequest) GetDistinguish() (int, error, string) {
	return db.GetVisibilityTag(req.BatchID, req.GroupID), nil, errs.Success
}

// judgeCodesBatchGroupLimit 验证目标销毁集合是否包含重复码。
func judgeRepeated(codes []string) (bool, string) {
	check := make(map[string]struct{}, 64)

	for _, code := range codes {
		if _, ok := check[code]; ok {
			return true, code
		}
		check[code] = struct{}{}
	}
	return false, ""
}

// judgeCodesBatchGroupLimit 验证销毁兑换码的批次和组号是否满足限定。
func judgeCodesBatchGroupLimit(codes []string, batchID int, groupID int, isCustom bool) (error, string) {
	if isCustom {
		// 自定义码限定校验。
		if len(codes) != 1 {
			return fmt.Errorf("judgeCodesBatchGroupLimit destroy custom code num only can be one"), errs.ErrHandlerDestroyCustomCodeOnlyOne
		}

		batchResID, groupResID, err, errMsg := db.GetBatchGroupIDByCustomCode(codes[0])
		if err != nil {
			return err, errMsg
		}

		if batchResID != batchID || groupResID != groupID {
			return fmt.Errorf("judgeCodesBatchGroupLimit code:%v resBatch: %v, resGroup: %v, batchID: %v, groupID: %v not match", codes[0], batchResID, groupResID, batchID, groupID), errs.ErrHandlerDestroyCodesNotMatchRequest
		}

		return nil, errs.Success
	} else {
		// 生成码限定校验。
		for _, code := range codes {
			batchResID, groupResID, err, errMsg := handler.GetBatchGroupIDByCharCoding(code)
			if err != nil {
				return err, errMsg
			}

			if batchResID != batchID || groupResID != groupID {
				return fmt.Errorf("judgeCodesBatchGroupLimit code:%v, resBatch: %v, resGroup: %v, batchID: %v, groupID: %v not match", code, batchResID, groupResID, batchID, groupID), errs.ErrHandlerDestroyCodesNotMatchRequest
			}
		}

		return nil, errs.Success
	}
}
