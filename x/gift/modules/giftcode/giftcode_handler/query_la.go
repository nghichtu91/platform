package giftcode_handler

import (
	"github.com/nghichtu91/platform/share/x/gift/db"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
)

// TODO JWS2-5018 查询逻辑和实际大区的映射关系（当前为直接返回所有的大区）。

type QueryLARequest struct {
}

type QueryLAResponse struct {
	// TODO L2AInfo map[string][]uint // map[逻辑大区][]实际大区。
	GidSlice []string `form:"gid_slice" bson:"gid_slice" json:"gid_slice"` // 当前配置的全部大区。
}

func (req *QueryLARequest) Handle() (interface{}, error, string) {
	ret, err, errMsg := db.GetGlobalGidInfo()
	if err != nil {
		return nil, err, errMsg
	}

	resp := &QueryLAResponse{
		GidSlice: ret,
	}

	return resp, nil, errs.Success
}

func (req *QueryLARequest) GetDistinguish() (int, error, string) {
	return 0, nil, errs.Success
}
