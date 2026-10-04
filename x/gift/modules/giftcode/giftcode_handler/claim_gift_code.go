package giftcode_handler

import (
	"fmt"
	"regexp"
	"time"

	"github.com/nghichtu91/platform/share/x/gift/model/define"

	"github.com/nghichtu91/platform/share/x/gift/config"
	"github.com/nghichtu91/platform/share/x/gift/db"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
	"github.com/nghichtu91/platform/share/x/gift/modules/common/handler"
)

type ClaimGiftCodeRequest struct {
	GiftCode     string `form:"gift_code" bson:"gift_code" json:"gift_code"`                // 要领取的兑换码。
	HasUsedBatch []int  `form:"has_used_batch" bson:"has_used_batch" json:"has_used_batch"` // 玩家已用的批次。
	ChannelID    string `form:"channel_id" bson:"channel_id" json:"channel_id"`             // 玩家的渠道编号。
	ACID         string `form:"acid" bson:"acid" json:"acid"`                               // 玩家唯一标识符。
	BatchID      int    `form:"batch_id" bson:"batch_id" json:"batch_id"`                   // 需要领取的批次。
	GroupID      int    `form:"group_id" bson:"group_id" json:"group_id"`                   // 需要领取的组号。
	Gid          string `form:"gid" bson:"gid" json:"gid"`                                  // 玩家所在的大区。
}

type ClaimGiftCodeResponse struct {
	BatchID     int           `form:"batch_id" bson:"batch_id" json:"batch_id"` // 领取的批次。
	Items       []define.Item `form:"items" bson:"items" json:"items"`          // 玩家可以领取到的道具。
	MaxUseTimes int           `form:"mut" bson:"mut" json:"mut"`                // 最大使用次数。
	CurUseTimes int           `form:"cut" bson:"cut" json:"cut"`                // 当前使用次数。
}

const ChannelNoLimit = -1

func (req *ClaimGiftCodeRequest) Handle() (interface{}, error, string) {
	// 玩家是否已经领取此批次（非自定义码在Game做初步拦截验证）。
	if intContains(req.HasUsedBatch, req.BatchID) {
		return nil, fmt.Errorf("ClaimGiftCodeRequest player have used this batch"), errs.ErrHandlerPlayerHasUseThisBatch
	}

	// 获取此批次配置信息。
	exist, codeConfig, errExist, errExistMsg := db.JudgeBatchGroupExist(req.BatchID, req.GroupID)
	if errExist != nil {
		return nil, errExist, errExistMsg
	}
	if !exist {
		return nil, fmt.Errorf("ClaimGiftCodeRequest Handle BatchID: %v, GroupID: %v not in used", req.BatchID, req.GroupID), errs.ErrHandlerBatchGroupIDHasNotInUsed
	}

	// 获取此批次使用信息。
	isCustom, _ := regexp.Match(config.RegexpContainNum, []byte(req.GiftCode))
	info, err, errMsg := db.GetUseInfoByCodeAndBatch(req.GiftCode, req.BatchID, isCustom)
	if err != nil {
		return nil, err, errMsg
	}
	useInfo, ok := info[req.GiftCode]
	if !ok {
		return nil, fmt.Errorf("ClaimGiftCodeRequest get use info err"), errs.ErrHandlerFindUseInfo
	}

	// 此码是否已经被销毁。
	if useInfo.HasDestroy {
		return nil, fmt.Errorf("ClaimGiftCodeRequest code has been destroy, code: %v", req.GiftCode), errs.ErrHandlerCodeHasBeenDestroy
	}

	// 是否已达使用的上限。
	if codeConfig.UseCount >= 0 {
		if len(useInfo.Users) >= codeConfig.UseCount {
			return nil, fmt.Errorf("ClaimGiftCodeRequest code has reach use limit, code: %v, limit: %v", req.GiftCode, codeConfig.UseCount), errs.ErrHandlerCodeHasReachUseLimit
		}
	}

	// 是否在码的有效期内。
	nowTime := time.Now().Unix()
	if nowTime < int64(codeConfig.StartTime) || nowTime > int64(codeConfig.EndTime) {
		return nil, fmt.Errorf("ClaimGiftCodeRequest code not in valid time"), errs.ErrHandlerCodeNotInValidTime
	}

	// 渠道是否在可领列表。
	if !intContains(codeConfig.ChannelIds, ChannelNoLimit) {
		if !judgeChannel(codeConfig.ChannelIds, req.ChannelID) {
			return nil, fmt.Errorf("ClaimGiftCodeRequest channel not match, allow: %v, now: %v", codeConfig.ChannelIds, req.ChannelID), errs.ErrHandlerChannelNotMatch
		}
	}

	// 大区是否在可领列表。
	if !stringContains(codeConfig.Gid, req.Gid) {
		return nil, fmt.Errorf("ClaimGiftCodeRequest Gid not match, allow: %v, now: %v", codeConfig.Gid, req.Gid), errs.ErrHandlerGidNotMatch
	}

	// 记录兑换码领取信息。
	useInfo.Users = append(useInfo.Users, req.ACID)
	if err, errMsg := db.UpdateUsesInfo(req.GiftCode, req.BatchID, useInfo.Users, isCustom); err != nil {
		return nil, err, errMsg
	}

	// 构造请求返回结构体。
	resp := &ClaimGiftCodeResponse{
		BatchID:     req.BatchID,
		Items:       codeConfig.Items,
		MaxUseTimes: codeConfig.UseCount, // 礼包码最大使用次数。
		CurUseTimes: len(useInfo.Users) + 1,
	}
	return resp, nil, errs.Success
}

func (req *ClaimGiftCodeRequest) GetDistinguish() (int, error, string) {
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

func judgeChannel(allowChannel []int, channel string) bool {
	for _, allow := range allowChannel {
		if fmt.Sprint(allow) == channel {
			return true
		}
	}
	return false
}

func intContains(slice []int, num int) bool {
	for _, elem := range slice {
		if elem == num {
			return true
		}
	}
	return false
}

func stringContains(slice []string, target string) bool {
	for _, elem := range slice {
		if elem == target {
			return true
		}
	}
	return false
}
