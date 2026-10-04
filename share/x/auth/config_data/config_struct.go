package config_data

// 返利通用信息结构。
type RewardInfo struct {
	HasClaimed bool  // 是否已领取。
	MoneyScore int32 // 累充积分。
	LoginDay   int32 // 登陆天数。
	Fair1v1Dan int32 // 公平1v1段位。
	Fair3v3Dan int32 // 公平3v3段位。
	ClaimTime  int64 // 领取时间
}

// 充值返利配置结构。
type PayRewardConfig struct {
	UserID     string `json:"user_id" bson:"user_id"`         // userID。
	MoneyScore int32  `json:"money_score" bson:"money_score"` // 累充积分。
}

// 登陆和排名返利配置结构。
type LoginAndRankConfig struct {
	UserID     string `json:"user_id" bson:"user_id"`           // userID。
	LoginDay   int32  `json:"login_day" bson:"login_day"`       // 登陆天数。
	Fair1v1Dan int32  `json:"fair_1v1_dan" bson:"fair_1v1_dan"` // 公平1v1段位。
	Fair3v3Dan int32  `json:"fair_3v3_dan" bson:"fair_3v3_dan"` // 公平3v3段位。
}

// 返利配置文件的加载结构。
type RewardLoadData struct {
	PayRewardMap    map[string]*PayRewardConfig    // 充值返利。
	LoginAndRankMap map[string]*LoginAndRankConfig // 登陆和排名返利。
}
