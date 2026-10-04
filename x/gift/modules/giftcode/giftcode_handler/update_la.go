package giftcode_handler

import (
	"github.com/nghichtu91/platform/share/planx/arrayhelper"
	"github.com/nghichtu91/platform/share/x/gift/db"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
)

// TODO JWS2-5018 更新逻辑和实际大区的映射关系（当前为直接返回所有的大区）。

const (
	Del = "Delete"   // 从配置大区中删除指定的大区。
	Add = "Addition" // 从配置大区中新增指定的大区。
)

type UpdateLARequest struct {
	// TODO L2AInfo map[string][]uint `form:"la_info" bson:"la_info" json:"la_info"` // map[逻辑大区][]实际大区。
	Gid    string `form:"gid" bson:"gid" json:"gid"`             // 要操作的大区ID。
	OpType string `form:"op_type" bson:"op_type" json:"op_type"` // 操作的类型。
}

type UpdateLAResponse struct {
}

func (req *UpdateLARequest) Handle() (interface{}, error, string) {
	resp := &UpdateLAResponse{}

	if req.Gid == "" {
		return resp, nil, errs.Success
	}

	ret, err, errMsg := db.GetGlobalGidInfo()
	if err != nil {
		return nil, err, errMsg
	}

	if req.OpType == Add {
		if arrayhelper.ContainsString(ret, req.Gid) {
			return resp, nil, errs.Success
		}
		ret = append(ret, req.Gid)
		if err, errMsg := db.UpdateGlobalGidInfo(ret); err != nil {
			return nil, err, errMsg
		}
	} else {
		if !arrayhelper.ContainsString(ret, req.Gid) {
			return resp, nil, errs.Success
		}
		for index, value := range ret {
			if value == req.Gid {
				ret = append(ret[:index], ret[index+1:]...)
				if err, errMsg := db.UpdateGlobalGidInfo(ret); err != nil {
					return nil, err, errMsg
				}
				break
			}
		}

	}

	return resp, nil, errs.Success
}

// GetDistinguish 获取请求的Hash依据。
//
// Note: 当前LA的请求均被Hash到同一个Channel中，后续处理均为串行。
// TODO 此处有时间优化加入LA_Handler，数据库操作已支持并发，无需改动。
// TODO 不加入LA_Handler的话可以将无Hash要求的分发增加随机操作。
func (req *UpdateLARequest) GetDistinguish() (int, error, string) {
	return 0, nil, errs.Success
}
