package nt

// 文档链接：http://confluence.taiyouxi.net/pages/viewpage.action?pageId=86511113

const (
	suffix = "/push"
)

// PushReq 向中台发起推送消息请求
type PushReq struct {
	AcID       string                 `json:"acid"`                // 玩家 ACID
	TemplateID int64                  `json:"templateId"`          // 推送消息模版 ID（来源于中台消息推送后台）
	AppID      int                    `json:"appId"`               // 应用 ID
	ExtraInfo  interface{}            `json:"extraInfo,omitempty"` // 附加信息
	Macros     map[string]interface{} `json:"macros"`              // 宏替换
	Sign       string                 `json:"sign"`                // 签名
}

type PushResp struct {
	Code   int    `json:"code"`             // 返回码
	Reason string `json:"reason,omitempty"` // 返回信息
}

func GetPushRoute() string {
	return requestPath + suffix
}
