package json_define

// 向gm server 请求禁言信息
type SendMsgRet struct {
	Code    int    `json:"code"`
	EndTime int64  `json:"entTime"`
	Reason  string `json:"reason"`
}
