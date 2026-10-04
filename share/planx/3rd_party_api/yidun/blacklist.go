package yidun

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"

	"github.com/astaxie/beego/httplib"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

const (
	url = "https://as.dun.163yun.com/v2/list/submit"
)

// 添加黑名单请求
type blackListReq struct {
	CommonReq

	// ListType 名单分类
	// 1: 白名单，2: 黑名单 106: 封禁名单
	ListType int `json:"listType"`

	// EntityType 名单类型
	// 1: 用户名单，2: IP名单，3: 设备名单 6: 直播账号名单
	EntityType int `json:"entityType"`

	// releaseTime 释放时间，可选
	// 设置此值表示名单到指定时间失效，单位为ms
	ReleaseTime int64 `json:"releaseTime,omitempty"`

	// Entities 名单列表，json字符串
	Entities string `json:"entities"`
}

type blackListResp struct {
	CommonResp

	Result []struct {
		// UUID
		// 名单唯一标识，通常为32位长度由数字或者字母组成，用来删除修改和查询
		UUID string `json:"uuid"`

		// Entity
		// 添加时传入的名单值
		Entity string `json:"entity"`

		// EntityType
		// 名单类型，1: 用户名单，2: IP名单，3: 设备名单 6: 直播账号名单
		EntityType int `json:"entityType"`

		// Exist
		// 名单是否已存在，false：不存在；true：已存在
		Exist bool `json:"exist"`
	} `json:"result"`
}

func NewBlackListReq() *blackListReq {
	return &blackListReq{
		CommonReq: newCommonReq(),
	}
}

func (req *blackListReq) handle() (int, error) {
	var (
		err error
	)

	resp := &blackListResp{}

	// 所有初始字段放入map[string]string结构
	jsonBytes, _ := json.Marshal(req)
	params := make(map[string]interface{}, 16)
	if err = json.Unmarshal(jsonBytes, &params); err != nil {
		return 0, err
	}

	// http请求
	hReq := httplib.Post(url).
		Header(headerTypeKey, headerTypeValue).
		SetTimeout(HttpTimeOut, HttpTimeOut).
		Retries(Retries)

	// 排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 生成字符串，并加入param
	buff := bytes.NewBuffer(make([]byte, 0, 1024))
	var v string
	for _, key := range keys {
		buff.WriteString(key)
		switch params[key].(type) {
		case string:
			v = params[key].(string)
		case float64:
			v = strconv.FormatFloat(params[key].(float64), 'f', -1, 64)
		default:
			tilogs.L().Errorf("param %v not support", params[key])
			continue
		}
		buff.WriteString(v)
		hReq.Param(key, v)
	}
	buff.WriteString(cfg.secretKey)

	// 生成md5
	h := md5.New()
	h.Write(buff.Bytes())
	sign := hex.EncodeToString(h.Sum(nil))

	// http参数加入签名
	hReq.Param(signKey, sign)

	if err = hReq.ToJSON(resp); err != nil {
		return 0, err
	}

	tilogs.L().Infof("blackListReq handle finished, resp %v", resp)

	return resp.Code, nil
}
