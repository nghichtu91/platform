package giftcode_handler

import (
	"encoding/json"
	"fmt"

	"github.com/nghichtu91/platform/share/x/gift/db"
	"github.com/nghichtu91/platform/share/x/gift/model/define"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
)

// 修改兑换码信息。

type UpdateGiftCodeRequest struct {
	Info string `form:"gen_info" bson:"gen_info" json:"gen_info"` // GmTools的JsonMarshal后的更新配置。
}

type UpdateGiftCodeResponse struct {
}

func (req *UpdateGiftCodeRequest) Handle() (interface{}, error, string) {
	// Unmarshal 获取兑换码的请求信息。
	info := &define.GiftCodeInfo{}
	errUnMarshal := json.Unmarshal([]byte(req.Info), &info)
	if errUnMarshal != nil {
		return nil, fmt.Errorf("UpdateGiftCodeRequest Handle Json Unmarshal Err: %v", errUnMarshal), errs.ErrHandlerJsonUnMarshal
	}

	// 检查目标批次和组号是否已经存在。
	exist, codeConfig, errExist, errExistMsg := db.JudgeBatchGroupExist(info.BatchID, info.GroupID)
	if errExist != nil {
		return nil, errExist, errExistMsg
	}
	if !exist {
		return nil, fmt.Errorf("UpdateGiftCodeRequest Handle BatchID: %v, GroupID: %v", info.BatchID, info.GroupID), errs.ErrHandlerBatchGroupIDHasNotInUsed
	}

	// 对更新参数设置和检查。
	if len(info.Gid) == 0 {
		return nil, fmt.Errorf("UpdateGiftCodeRequest at least one gid visibility"), errs.ErrHandlerAtLeastOneGidVisibility
	}
	if info.GiftType != codeConfig.GiftType {
		return nil, fmt.Errorf("UpdateGiftCodeRequest not allow change gift type, old: %v, change: %v", codeConfig.GiftType, info.GiftType), errs.ErrHandlerUpdateNotAllowChangeType
	}
	if info.CustomCode != codeConfig.CustomCode {
		return nil, fmt.Errorf("UpdateGiftCodeRequest not allow change custom old: %v, new: %v", codeConfig.CustomCode, info.CustomCode), errs.ErrHandlerUpdateNotAllowChangeCustom
	}
	info.GenCount = codeConfig.GenCount
	info.TotalCount = codeConfig.TotalCount
	info.GenTime = codeConfig.GenTime

	if info.GiftType != 3 {
		info.UseCount = codeConfig.UseCount
	}

	if err, errMsg := db.UpdateGiftCodeConfig(codeConfig, info); err != nil {
		return nil, err, errMsg
	}

	return &UpdateGiftCodeResponse{}, nil, errs.Success
}

func (req *UpdateGiftCodeRequest) GetDistinguish() (int, error, string) {
	// Unmarshal 获取兑换码的更新信息。
	info := &define.GiftCodeInfo{}
	errUnMarshal := json.Unmarshal([]byte(req.Info), &info)
	if errUnMarshal != nil {
		return -1, fmt.Errorf("UpdateGiftCodeRequest GetNumDistinguish Json Unmarshal Err: %v", errUnMarshal), errs.ErrHandlerJsonUnMarshal
	}

	return db.GetVisibilityTag(info.BatchID, info.GroupID), nil, errs.Success
}
