package protocol

type ProtocolHandler interface {
	// Init template command
	Init()
	// 使用固定配置好的模版发送
	FixedRequest(index string) []byte
	// 手动输入参数 调用请求
	ControlledRequest(params []string) []byte
	// 将返回的包打印出来
	ResponseString(msg []byte) string
}
