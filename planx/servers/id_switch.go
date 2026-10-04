package servers

/*
	shardId的规则是：gid+第几个服，sid预留了4位，即一个大区不超过9999个服，应该够用了
*/

// 为玩家生成playerid
func GenPlayerId(n int32, serverIndex int32) int64 {
	return 65536*int64(n) + int64(serverIndex)
}

// GetSidByPlayerId
//
// Deprecated: 由于GMServer和GMS的对应关系改为1:*，所以此函数无法继续使用（GMServer已禁用PlayerID查询ShardID），无可替代函数。
func GetSidByPlayerId(playerId int64, gid uint) uint {
	return gid*10000 + uint(playerId%65536)
}

func GetServerIndexByShardId(sid uint) uint {
	return sid % 10000
}
