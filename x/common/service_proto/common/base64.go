package common

import (
	"github.com/nghichtu91/platform/share/planx/secure"
)

// Enc auth的base64编码
func Enc(src string) string {
	return secure.DefaultEncode.Encode64ForNet([]byte(src))
}

// Dec auth的base64解码
func Dec(src string) string {
	d, err := secure.DefaultEncode.Decode64FromNet(src)
	if err != nil {
		return ""
	}
	return string(d)
}
