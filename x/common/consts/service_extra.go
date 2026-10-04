package consts

type GateExtra struct {
	Ccu    int64
	IsOpen bool
}

type BattleExtra struct {
	Ccu int64
}

type BattleCheckExtra struct {
	TaskCount int64
}

type AuthExtraInfo struct {
	InternalAddr    string // 内网的ip和端口
	PublicAddr      string // 外网ip和端口
	LoginGateRegUrl string // gateregister的完整url
	LoginLoginUrl   string // notifylogin的完整url
	LoginLogoutUrl  string // notifylogout的完整url
	LoginNotifyUser string // notifyuserinfo的完整url

	AuthBanUrl          string // ban的完整url
	AuthIsBanUrl        string // 玩家是否禁言
	AuthGagUrl          string // gag的完整url
	AuthKickUrl         string // kick的完整url
	GetShardNewUseCount string // 获取最新服务器注册角色数

	AuthBanPlayerUrl   string // 封禁角色
	AuthIsBanPlayerUrl string // 玩家是否被封禁

	QuerySdkIdUrl          string // SdkId与Uid相关信息
	InsertOrUpdateSdkIdUrl string
	DeleteSdkIdUrl         string

	// 白名单操作路径
	QueryWhiteList          string
	DeleteWhiteList         string
	InsertOrUpdateWhiteList string

	//特殊活动奖励操作
	QueryUserSpecialAward  string
	UpdateUserSpecialAward string

	//充值返钻
	PayRewardForUser string
}
