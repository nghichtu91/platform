package errs

import "fmt"

const (
	SERVER = "服务器内部错误"

	DbError = "数据库错误"

	BadType          = "请求类型错误"
	PwdError         = "账号或者密码错误"
	AccessToken      = "版本未授权"
	AccountForbidden = "账号已被屏蔽"
)

var LogicError = fmt.Errorf("GmEmptyError")
