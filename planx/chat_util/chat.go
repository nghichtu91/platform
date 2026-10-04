package chat_util

import (
	"regexp"
)

// RegCheckFilterEmoji 目前客户端实现的表情符号，策划配表实现，目前都是[x]的形式，x为数字和字母，可以拓展
var RegCheckFilterEmoji = regexp.MustCompile("(\\[[0-9a-zA-Z]\\])+")

// IsPureEmoji 检查是否为纯表情
func IsPureEmoji(text string) bool {
	indexes := RegCheckFilterEmoji.FindAllStringIndex(text, -1)
	textLength := len(text)
	for _, element := range indexes {
		textLength -= element[1] - element[0]
	}
	return textLength == 0
}
