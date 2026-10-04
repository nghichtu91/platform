package feishu

// SendFeishu 发送飞书消息
// 注意：context的格式和内容需要自己预先处理
func SendFeishu(url, content string, isAtAll bool) error {
	if sender == nil {
		return ErrFeishuNotInit
	}

	select {
	case sender.recChan <- Req{
		Url:     url,
		Data:    content,
		IsAtAll: isAtAll,
	}:
	default:
		return ErrFeishuSendChannelFull
	}

	return nil
}
