package yidun

import (
	"strings"

	"github.com/opentracing/opentracing-go"
)

// Check 对指定文本内容进行易盾检查
// 返回易盾判断结果和error
// 如果有错误，判断结果一定为pass，不会阻塞游戏逻辑
func Check(msg ITextValidateReq, span opentracing.Span) (int, []*Lable, error) {
	if cfg == nil {
		return RetPass, nil, ErrYiDunNotInit
	}

	req := getTVReq()
	defer putTVReq(req)

	req.Content = msg.GetText()
	req.Account = msg.GetAcid()
	req.NickName = msg.GetNickname()
	req.ReceiveUid = msg.GetReceiver()
	req.DeviceId = strings.ToUpper(msg.GetDeviceID())
	req.IP = msg.GetIp()
	req.ExtStr1 = msg.GetNtID()
	req.ExtStr2 = msg.GetDeviceOS()
	req.ExtLon1 = msg.GetMessageType()

	return req.handle(span)
}

// BlackList 将指定账号加入黑名单
// 解封时间填0表示永远封禁
func BlackList(releaseTS int64, acids ...string) (int, error) {
	if cfg == nil {
		return 0, ErrYiDunNotInit
	}

	if len(acids) == 0 {
		return 0, ErrNoAcidForBlackList
	}

	req := NewBlackListReq()
	if releaseTS != 0 {
		req.ReleaseTime = releaseTS * 1000 // 易盾接收时间戳是毫秒
	}

	req.ListType = 2   // 黑名单
	req.EntityType = 1 // 用户名单
	req.Version = "v2"

	// json化
	builder := strings.Builder{}
	builder.WriteString("[")
	for i, acid := range acids {
		if i != 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("\"" + acid + "\"")
	}
	builder.WriteString("]")
	req.Entities = builder.String()

	return req.handle()
}
