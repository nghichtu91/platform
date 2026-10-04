package giftcode_handler

import (
	"github.com/nghichtu91/platform/share/x/gift/db/oss"

	"github.com/nghichtu91/platform/share/x/gift/db"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
	"github.com/nghichtu91/platform/share/x/gift/modules/common/util"
)

// 下载兑换码。

type DownloadGiftCodeRequest struct {
	BatchID int `form:"batch_id" bson:"batch_id" json:"batch_id"` // 下载兑换码的批次。
	GroupID int `form:"group_id" bson:"group_id" json:"group_id"` // 下载兑换码的组号。
}

type DownloadGiftCodeResponse struct {
	FileName string // 向GmTools返回的文件名标识。
	PathInfo string // 向GmTools返回的OSS路径。
}

func (req *DownloadGiftCodeRequest) Handle() (interface{}, error, string) {
	// 向GmTools返回最终生成的文件名和OSS下载地址。
	resp := &DownloadGiftCodeResponse{
		FileName: util.GetBatchGroupFileName(req.BatchID, req.GroupID),
		PathInfo: oss.GetBatchGroupOSSPath(req.BatchID, req.GroupID),
	}

	return resp, nil, errs.Success
}

func (req *DownloadGiftCodeRequest) GetDistinguish() (int, error, string) {
	return db.GetVisibilityTag(req.BatchID, req.GroupID), nil, errs.Success
}
