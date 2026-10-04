package mergeinfo

// IManager 用于维护合服关系的管理接口
// 目前依赖于etcd实现
// 后续不排除使用数据库、gamedata等
type IManager interface {
	// GetShardInfoBySID 查询对应SID的信息
	GetShardInfoBySID(sid int) *ShardInfo

	// GetGIDBySID 根据SID查询所属GID
	// 不存在的Shard返回-1
	// 未启动的Shard返回GID2SID配置内的GID
	GetGIDBySID(sid int) int

	// GetRealSIDBySID 根据SID查询主服SID
	// 不存在的Shard返回-1
	// 未启动的Shard返回GID2SID配置内的SID
	GetRealSIDBySID(sid int) int

	// GetVerInfo 当前合服信息
	// 由于每个Manager实现不一样，使用通用的json格式作为返回值，方便解析
	GetVerInfo() []byte

	// SetInfo 使用当前一批数据更新所有合服信息
	// 会对数据有效性进行检查
	SetInfo(s2d Sid2GidInfo, si ...*ShardInfo) error

	// UpdateSid2Gid 更新GID映射信息
	// 会对数据有效性进行检查
	UpdateSid2Gid(s2d Sid2GidInfo) error

	// UpdateShard 更新单个Shard信息
	UpdateShard(info *ShardInfo) error

	// Clone 复制单个Manager
	// 用于检查下一批数据正确性等不能改变原数据的情况
	Clone() IManager

	// Stop 关闭Manager
	Stop()
}
