package define

type GiftCodeInfo struct {
	BatchID    int      `form:"batch_id" bson:"batch_id" json:"batch_id"`          // 批次ID（上限为: config.CodeMaxBatchNum）。
	GroupID    int      `form:"group_id" bson:"group_id" json:"group_id"`          // 组号ID（上限为: config.CodeMaxGroupNum）。
	Gid        []string `form:"gid" bson:"gid" json:"gid"`                         // 兑换码适用大区。
	ChannelIds []int    `form:"channel_ids" bson:"channel_ids" json:"channel_ids"` // 兑换码适用渠道（可多选）。
	GiftName   string   `form:"gift_name" bson:"gift_name" json:"gift_name"`       // 兑换码名字/标识符。
	StartTime  int      `form:"start_time" bson:"start_time" json:"start_time"`    // 兑换码可兑换开始时间。
	EndTime    int      `form:"end_time" bson:"end_time" json:"end_time"`          // 兑换码可兑换结束时间
	GiftType   int      `form:"gift_type" bson:"gift_type" json:"gift_type"`       // 兑换码类型（1-普通兑换码;2-通用兑换码;3-多用兑换码）
	Items      []Item   `form:"items" bson:"items" json:"items"`                   // 兑换码附带物品信息。
	GenCount   int      `form:"gen_count" bson:"gen_count" json:"gen_count"`       // 本次生成的数量。
	TotalCount int      `form:"total_count" bson:"total_count" json:"total_count"` // 累计生成的数量（不需要gm_tools给与）（为了知道追加时上次的Index）。
	GenTime    int      `form:"gen_time" bson:"gen_time" json:"gen_time"`          // 首次生成的时间（不需要gm_tools给与）（追加生成时不改变此值）。
	UseCount   int      `form:"use_count" bson:"use_count" json:"use_count"`       // 允许使用人数（只有多用兑换码需要gm_tools给与，其余默认填写）。
	CustomCode string   `form:"custom_code" bson:"custom_code" json:"custom_code"` // 自定义兑换码。
}

type GiftCodeQueryInfo struct {
	Users      []string `form:"users" bson:"users" json:"users"`                   // 使用者列表。
	HasDestroy bool     `form:"has_destroy" bson:"has_destroy" json:"has_destroy"` // 是否已被销毁。
}

type BatchGroupInfo struct {
	BatchID int `form:"batch_id" bson:"batch_id" json:"batch_id"` // 批次ID。
	GroupID int `form:"group_id" bson:"group_id" json:"group_id"` // 组号ID。
}
