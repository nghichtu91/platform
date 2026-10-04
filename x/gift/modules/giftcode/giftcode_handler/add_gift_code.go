package giftcode_handler

import (
	"fmt"

	"github.com/nghichtu91/platform/share/x/gift/config"
	"github.com/nghichtu91/platform/share/x/gift/db"
	"github.com/nghichtu91/platform/share/x/gift/model/define"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
	"github.com/nghichtu91/platform/share/x/gift/modules/common/handler"
	"github.com/nghichtu91/platform/share/x/gift/modules/common/util"
)

// 追加兑换码。

type AddGiftCodeRequest struct {
	BatchID     int `form:"batch_id" bson:"batch_id" json:"batch_id"`    // 需要追加兑换码的批次。
	GroupID     int `form:"group_id" bson:"group_id" json:"group_id"`    // 需要追加兑换码的组号。
	AppendCount int `form:"add_count" bson:"add_count" json:"add_count"` // 需要追加兑换码的数量。
}

type AddGiftCodeResponse struct {
	FileName string // 向GmTools返回的文件名标识。
	// Data     []byte // 追加完毕后不再向GmTools返回，需要运营手动下载。
}

func (req *AddGiftCodeRequest) Handle() (interface{}, error, string) {
	// 检查目标批次和组号是否已经生成。
	exist, oldConfig, errExist, errExistMsg := db.JudgeBatchGroupExist(req.BatchID, req.GroupID)
	if errExist != nil {
		return nil, errExist, errExistMsg
	}
	if !exist {
		return nil, fmt.Errorf("AddGiftCodeRequest Handle BatchID: %v, GroupID: %v not in used", req.BatchID, req.GroupID), errs.ErrHandlerBatchGroupIDHasNotInUsed
	}

	// 填充新的配置信息。
	var info *define.GiftCodeInfo
	info = oldConfig
	info.GenCount = req.AppendCount

	// 对生成参数判断和设置。
	if info.GiftType != config.TypeGiftCodeNormal {
		return nil, fmt.Errorf("AddGiftCodeRequest only normal allow append but now: %v", oldConfig.GiftType), errs.ErrHandlerOnlyNormalAllowAppend
	}

	// 生成追加礼包码
	codes, err, errMsg := handler.GenGiftCode(uint64(config.Cfg.GiftCfg.CodeProjectNum), uint64(info.BatchID), uint64(info.GroupID), uint64(info.GenCount), uint64(oldConfig.TotalCount))
	if err != nil {
		return nil, err, errMsg
	}

	// 生成兑换码写入数据库。
	if err, errMsg := db.InsertGenCodeList(codes, info, db.AppendMode); err != nil {
		return nil, err, errMsg
	}

	// 构造GmTools请求成功处理反馈信息。
	resp := &AddGiftCodeResponse{
		FileName: util.GetBatchGroupFileName(info.BatchID, info.GroupID),
	}

	return resp, nil, errs.Success
}

func (req *AddGiftCodeRequest) GetDistinguish() (int, error, string) {
	return db.GetVisibilityTag(req.BatchID, req.GroupID), nil, errs.Success
}
