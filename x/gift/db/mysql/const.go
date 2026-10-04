package mysql

const (
	colGiftCode   = "gift_code"  // 礼包码。
	colUsers      = "users"      // 使用者集合。
	colDestroy    = "destroy"    // 是否被销毁。
	colBatchID    = "batch_id"   // 批次ID。
	colGroupID    = "group_id"   // 组号ID。
	colConfig     = "config"     // 生成配置信息。
	colGid        = "gid"        // 大区号。
	colVisibility = "visibility" // 大区对于批次和组的可见性。
	colLogic      = "logic_gid"  // 逻辑大区。
	colActual     = "actual_gid" // 实际大区。
	colInfoName   = "info_name"  // 全局信息名。
	colInfo       = "info"       // 全局信息。
)

const (
	opInsert         = "insert into"      // 插入操作。
	opUpdate         = "update"           // 更新操作。
	opSelect         = "select"           // 查询操作。
	opShowTables     = "show tables like" // 前缀表列举操作。
	opSelectGreatest = "select greatest"  // 获取最大值。
)

const (
	primaryKey         = "PRIMARY KEY"             // 主键。
	duplicateKeyUpdate = "ON DUPLICATE KEY UPDATE" // 重复时。
)

const (
	global_gid = "gid" // 配置的所有的大区。
)
