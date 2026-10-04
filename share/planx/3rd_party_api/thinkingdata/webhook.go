package thinkingdata

import (
	"fmt"
	"strings"
)

// 文档链接：https://docs.thinkingdata.cn/ta-manual/v4.1/user_guide/engage/engage_technical_document/webhook_connection.html

type WebhookRequest []struct {
	// PushID 推送ID
	PushID string `json:"push_id"`

	// Params 模板参数
	Params map[string]interface{} `json:"params,omitempty"`

	// CustomParams 自定义参数
	CustomParams map[string]interface{} `json:"custom_params,omitempty"`

	// OpsReceiptProperties TE运营模块回执字段
	OpsReceiptProperties map[string]interface{} `json:"#ops_receipt_properties"`
}

func (req *WebhookRequest) String() string {
	var sb strings.Builder

	sb.WriteString("WebhookRequest: ")
	for i, p := range *req {
		sb.WriteString("[" + fmt.Sprintf("%+v", p) + "]")
		if i != len(*req)-1 {
			sb.WriteString(", ")
		}
	}
	return sb.String()
}

type WebhookResponse struct {
	// 返回码
	// 0 代表成功（或者部分成功）
	// 1 代表失败
	ReturnCode int `json:"return_code"`

	// 返回信息
	ReturnMessage string `json:"return_message,omitempty"`

	// 返回数据
	Data ResponseData `json:"data,omitempty"`
}

type ResponseData struct {
	FailLists []FailList `json:"fail_list,omitempty"`
}

type FailList struct {
	Index   int    `json:"index"` // 编号从1开始，不是从0开始
	Message string `json:"message"`
}
