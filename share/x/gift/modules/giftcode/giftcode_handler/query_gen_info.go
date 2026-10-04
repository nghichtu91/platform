package giftcode_handler

import (
	"fmt"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/x/gift/db"
	"github.com/nghichtu91/platform/share/x/gift/model/define"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
)

// 获取已生成兑换码信息。

type QueryGenInfoRequest struct {
	Gid []string `form:"gid" bson:"gid" json:"gid"` // GmTools的可见大区。
}

type QueryGenInfoResponse struct {
	GiftCodeInfoList []*define.GiftCodeInfo `form:"info_list" bson:"info_list" json:"info_list"`          // 礼包码信息列表。
	MaxBatchID       int                    `form:"max_batch_id" bson:"max_batch_id" json:"max_batch_id"` // 当前存在的最大批次。
	MaxGroupID       int                    `form:"max_group_id" bson:"max_group_id" json:"max_group_id"` // 最大批次中的最大组号。
}

func (req *QueryGenInfoRequest) Handle() (interface{}, error, string) {
	// 检查请求信息是否有效。
	if len(req.Gid) == 0 {
		return nil, fmt.Errorf("QueryGenInfoRequest please give at least one gid"), errs.ErrHandlerQueryGenInfoRequestGid
	}

	tilogs.L().Infof("QueryGenInfoRequest Handle db.GetGenInfo")
	giftCodeInfoList, maxBatchID, maxGroupID, err, errMsg := db.GetGenInfo(req.Gid)
	if err != nil {
		return nil, err, errMsg
	}

	resp := &QueryGenInfoResponse{
		GiftCodeInfoList: giftCodeInfoList,
		MaxBatchID:       maxBatchID,
		MaxGroupID:       maxGroupID,
	}

	return resp, nil, errs.Success
}

func (req *QueryGenInfoRequest) GetDistinguish() (int, error, string) {
	return 0, nil, errs.Success
}
