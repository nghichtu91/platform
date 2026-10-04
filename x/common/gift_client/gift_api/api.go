package gift_api

import "github.com/nghichtu91/platform/share/x/gift/model/api"

// 根分组
const (
	RootAPI = "/gift_code"
)

// 逻辑服API
const (
	ClaimAPI = api.ClaimGiftCode // 领取兑换码。

)

// GmToolsAPI
const (
	AddGiftCodeAPI      = api.AddGiftCode      // 追加兑换码。
	DownloadGiftCodeAPI = api.DownloadGiftCode // 下载兑换码。
	GenGiftCodeAPI      = api.GenGiftCode      // 生成兑换码。
	QueryGenGiftCodeAPI = api.QueryGenGiftCode // 查询已生成兑换码。
	QueryGiftAPI        = api.QueryGift        // 查询兑换码。
	UpdateGiftAPI       = api.UpdateGift       // 更新兑换码。
	DestroyGiftCodeAPI  = api.DestroyGiftCode  // 销毁兑换码。
	QueryLAInfo         = api.QueryLA          // 查询逻辑与实际大区映射关系。
	UpdateLAInfo        = api.UpdateLA         // 更新逻辑与实际大区映射关系。
)
