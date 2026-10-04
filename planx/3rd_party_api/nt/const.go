package nt

import (
	"errors"
	"time"
)

const (
	timeout      = 3 * time.Second // 请求超时时间：1秒
	sendChanSize = 16384           // 发送队列大小
)

var (
	ErrNtNotInit       = errors.New("nt not init")       // 未初始化
	ErrSendChannelFull = errors.New("nt send chan full") // 发送队列满
)

const (
	KeySign = "sign"
)

const (
	RespStatusSuccess = "success" // 请求成功
)

const (
	requestPath = "/game/request" // 请求接口
)

// 中台对应错误码
const (
	ErrInner = -1 // 中台内部错误
	ErrParam = -2 // 参数错误
	ErrSign  = -3 // 签名错误

	ErrRequestTooQuick        = 1001 // 请求过于频繁
	ErrTemplateNotExist       = 1002 // 模版不存在
	ErrAppNotExist            = 1003 // 应用不存在
	ErrDuplicateContent       = 1004 // 消息防重限制(同样的消息 5min 内最多发一条）
	ErrSpamBlock              = 1005 // 消息防轰炸限制(单个用户不同消息 1min 内最多1条）
	ErrRequestToChannelFailed = 1006 // 请求渠道失败
	ErrNoClientData           = 1007 // 客户端未上报数据
	ErrPlatformNotSupport     = 1008 // 消息推送暂不支持除 IOS 和 Android 以外的其他平台
)
