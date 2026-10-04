package command

//React前端页面通信协议
const (
	GmCommandToken             = "/api/login"   // 网页应用授权
	GmCommandLogin             = "/admin/check" // 用户登录
	GmCommandQueryGid          = "/api/login/gid"
	GmCommandQueryRecord       = "/api/record/query"
	GmCommandUpdateUser        = "/api/user/update"
	GmCommandQueryUser         = "/api/user/get"
	GmCommandDelUser           = "/api/user/del"
	GmCommandGetOneUser        = "/api/user/getOne"
	GmCommandCreatePower       = "/api/role/set"
	GmCommandGetPower          = "/api/role/get"
	GmCommandDelPower          = "/api/role/del"
	GetShardIds                = "/api/get/shardIds"
	UploadSensitiveWordsFile   = "/api/chatSystem/uploadSensitiveWordsFile"
	SearchCharInfo             = "/api/chatSystem/searchChatInfo"
	QueryChannelEtcd           = "/api/chatSystem/queryChannelEtcd"
	EditChannelEtcd            = "/api/chatSystem/editChannelEtcd"
	DownloadSensitiveWordsFile = "/api/chatSystem/downloadSensitiveWordsFile"
	QueryChatStoreTime         = "/api/chatSystem/queryChatStoreTime"
	EditChatStoreTime          = "/api/chatSystem/editChatStoreTime"

	// moreCommand
)

//操作记录所需中文对照
var CommandName = map[string]string{
	GmCommandToken:       "网页应用授权", // "/api/login"
	GmCommandLogin:       "登录",     // "/admin/check"
	GmCommandQueryRecord: "获取操作记录", // "/api/record/query"
	GmCommandUpdateUser:  "更新用户信息", // "/api/user/update"
	GmCommandQueryUser:   "获取用户列表", // "/api/user/get"
	GmCommandDelUser:     "删除用户",   // "/api/user/del"
	GmCommandGetOneUser:  "",       // "/api/user/getOne"
	GmCommandCreatePower: "创建用户组",  // "/api/role/set"
	GmCommandGetPower:    "获取用户组",  // "/api/role/get"
	GmCommandDelPower:    "删除用户组",  // "/api/role/del"
}
