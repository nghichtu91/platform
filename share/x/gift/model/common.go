package model

const (
	CodeSuccess   = 200
	CodeCommonErr = 400
)

const (
	MsgSuccess = "成功"
	MsgFail    = "失败"
)

/**
通用Response结构。
*/
type CommonResult struct {
	Code    int         `json:"code"`    // 成功与失败的Code标识。
	Message string      `json:"message"` // 反馈信息。
	Tag     string      `json:"tag"`     // 标记。
	Data    interface{} `json:"data"`    // 请求执行后的Response信息。
}

// 创建默认成功的返回信息。
func CreateDefaultOkResult() *CommonResult {
	cr := &CommonResult{}
	cr.Code = CodeSuccess
	cr.Message = MsgSuccess
	return cr
}

// 创建默认失败的返回信息。
func CreateDefaultFailResult() *CommonResult {
	cr := &CommonResult{}
	cr.Code = CodeCommonErr
	cr.Message = MsgFail
	return cr
}

func (cr *CommonResult) SetMessage(message string) {
	cr.Message = message
}

func (cr *CommonResult) SetTag(tag string) {
	cr.Tag = tag
}

type TEST struct {
	resultData interface{}
}

func (cr *CommonResult) SetData(resultData interface{}) {
	cr.Data = resultData
}
