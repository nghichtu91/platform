package merge_db

// 表、池序列。
const (
	IdxProfile = iota // 玩家存档昵称修改。
	IdxGuild
	IdxProfileDel // 玩家删档。
	IdxCount
)

// 表名称序列。
const (
	NameProfile = "profile"
	NameGuild   = "guild"
	ProfileDel  = "profile_del"
)

const (
	DefaultExpireSeconds = 259200 // 需要删除的DB，默认过期时间为72h
	DefaultRandRange     = 900    // 随机范围值, 防止缓存雪崩
)
