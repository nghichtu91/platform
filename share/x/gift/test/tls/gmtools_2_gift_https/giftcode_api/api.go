package giftcode_api

const (
	RootAPI          = "/gift_code"
	AddGiftCode      = "/api/gift_code/add"       // 追加兑换码。
	DownloadGiftCode = "/api/gift_code/download"  // 下载兑换码。
	GenGiftCode      = "/api/gift_code/gen"       // 生成兑换码。
	QueryGenGiftCode = "/api/gift_code/query_gen" // 查询已生成兑换码。
	QueryGift        = "/api/gift_code/query"     // 查询兑换码。
	UpdateGift       = "/api/gift_code/update"    // 更新兑换码。
	DestroyGiftCode  = "/api/gift_code/destroy"   // 销毁兑换码。
	QueryLA          = "/api/la/query"            // 查询逻辑与实际大区映射关系。
	UpdateLA         = "/api/la/update"           // 更新逻辑与实际大区映射关系。
)
