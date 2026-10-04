package giftcode_handler

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/nghichtu91/platform/share/x/gift/modules/common/util"

	"github.com/nghichtu91/platform/share/x/gift/modules/common/handler"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/x/gift/config"

	"github.com/nghichtu91/platform/share/x/gift/db"

	"github.com/nghichtu91/platform/share/x/gift/model/errs"

	"github.com/nghichtu91/platform/share/x/gift/model/define"
)

// 生成兑换码。

type GenGiftCodeRequest struct {
	Info string `form:"gen_info" bson:"gen_info" json:"gen_info"` // GmTools的JsonMarshal后的生成配置。
}

type GenGiftCodeResponse struct {
	FileName string // 向GmTools返回的文件名标识。
	// Record   []byte // 向GmTools返回的文件内容信息。
}

func (req *GenGiftCodeRequest) Handle() (interface{}, error, string) {
	// Unmarshal 生成兑换码的请求信息。
	info := &define.GiftCodeInfo{}
	errUnMarshal := json.Unmarshal([]byte(req.Info), &info)
	if errUnMarshal != nil {
		return nil, fmt.Errorf("GenGiftCodeRequest Handle Json Unmarshal Err: %v", errUnMarshal), errs.ErrHandlerJsonUnMarshal
	}

	// 检查目标批次和组号是否已被使用。
	exist, _, errExist, errExistMsg := db.JudgeBatchGroupExist(info.BatchID, info.GroupID)
	if errExist != nil {
		return nil, errExist, errExistMsg
	}
	if exist {
		return nil, fmt.Errorf("GenGiftCodeRequest Handle BatchID: %v, GroupID: %v in used", info.BatchID, info.GroupID), errs.ErrHandlerBatchGroupIDHasInUsed
	}

	// 设置兑换码生成的时间。
	info.GenTime = int(time.Now().Unix())

	// 对生成参数判断和设置。
	if len(info.Gid) == 0 {
		return nil, fmt.Errorf("GenGiftCodeReques at least one gid visibility"), errs.ErrHandlerAtLeastOneGidVisibility
	}
	if info.CustomCode != "" && info.GiftType == config.TypeGiftCodeNormal {
		return nil, fmt.Errorf("GenGiftCodeRequest Handle normal code not custom code"), errs.ErrHandlerNormalCodeNotCustomCode
	}
	if info.CustomCode != "" {
		if result, _ := regexp.Match(config.RegexpContainNum, []byte(info.CustomCode)); !result {
			return nil, fmt.Errorf("GenGiftCodeRequest Handle custom code must contain num"), errs.ErrHandlerCustomCodeMustContainNum
		}
	}
	if info.GiftType != config.TypeGiftCodeNormal {
		info.GenCount = 1 // 只有普通兑换码可以一次性生成多个。
	}
	if info.GiftType == config.TypeGiftCodeNormal {
		info.UseCount = 1 // 普通兑换码只允许一个人使用。
	}
	if info.GiftType == config.TypeGiftCodeUniversal {
		info.UseCount = -1 // 通用兑换码不设置使用人数上限。
	}

	tilogs.L().Debugf("GenGiftCodeRequest Handle GenCode info: %v", info)

	// 生成兑换码。
	codes := make([]string, 0, info.GenCount)
	if info.CustomCode != "" {
		// 自定义兑换码获取。
		codes = append(codes, info.CustomCode)
	} else {
		// 生成兑换码获取。
		var err error
		var errMsg string
		codes, err, errMsg = handler.GenGiftCode(uint64(config.Cfg.GiftCfg.CodeProjectNum), uint64(info.BatchID), uint64(info.GroupID), uint64(info.GenCount), config.CodeFirstGeneration)
		if err != nil {
			return nil, err, errMsg
		}
	}

	// 生成兑换码写入数据库。
	if err, errMsg := db.InsertGenCodeList(codes, info, db.GenMode); err != nil {
		return nil, err, errMsg
	}

	// 构造GmTools请求成功处理反馈信息。
	resp := &GenGiftCodeResponse{
		FileName: util.GetBatchGroupFileName(info.BatchID, info.GroupID),
	}

	return resp, nil, errs.Success
}

func (req *GenGiftCodeRequest) GetDistinguish() (int, error, string) {
	// Unmarshal 获取兑换码的请求信息。
	info := &define.GiftCodeInfo{}
	errUnMarshal := json.Unmarshal([]byte(req.Info), &info)
	if errUnMarshal != nil {
		return -1, fmt.Errorf("GenGiftCodeRequest GetNumDistinguish Json Unmarshal Err: %v", errUnMarshal), errs.ErrHandlerJsonUnMarshal
	}

	return db.GetVisibilityTag(info.BatchID, info.GroupID), nil, errs.Success
}
