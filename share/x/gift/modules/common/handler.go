package common

// 请求方请求的处理接口。
type HandlerInterface interface {
	Handle() (interface{}, error, string) // 处理函数，error用于内部的异常信息显示（英文字符），string用于请求方异常显示（中文字符注释+error）。
	GetDistinguish() (int, error, string) // 获取用于Handler分发的数字标识依据。
}

type Cmd struct {
	ResChan chan *Ret        // gin的POST handler将阻塞，直到超时或resChan给出反馈。
	ReqAPI  string           // cmd装载的API信息。
	ReqInfo HandlerInterface // cmd装载的请求信息。
}

type Ret struct {
	Data interface{} // 处理完成后的Response内容。
	Err  error       // 内部显示的异常信息。
	Msg  string      // 请求方额外显示的异常信息。
}
