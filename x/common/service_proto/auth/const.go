package auth

const (
	// 最基础的请求路径，只返回ip，不需要指定header
	echoPath = "/test/echoip"

	// Dev/QA环境使用，需要开启cheat，需要指定header
	// stress使用此接口进行测试
	devAuthPath = "/auth/v1/user/reg/"

	// 正式环境使用，需要指定header
	// servercheck使用此接口登录
	prodAuthPath = "/auth/v2/sdk/login"

	// 获取gate的接口
	getGatePath = "/login/v1/getgate"
)
