package etcd

type XSceneInfo struct {
	SerID    string          // xscene的serverid
	Addr     string          // xscene的内网地址
	ZoneID   uint64          // 战区分组
	Shards   []uint32        // 给哪些gamex服务
	SceneIDs map[int32]int32 // key：策划配的跨服分组地图，value：暂没用
	// TTL      int64           // 过期时间，暂不使用
}
