package common

import (
	"time"

	"github.com/astaxie/beego/httplib"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

const (
	defaultTimeOut = 3 * time.Second // http协议的超时设置
)

// NewReq 设置请求特殊头和超时，避免重复代码
func NewReq(url string, fn func(string) *httplib.BeegoHTTPRequest) *httplib.BeegoHTTPRequest {
	return fn(url).Header(consts.Spec_Header, consts.Spec_Header_Content).SetTimeout(defaultTimeOut, defaultTimeOut)
}
