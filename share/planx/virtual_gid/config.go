package virtual_gid

// VirtualGroupCfg 虚拟大区配置
type VirtualGroupCfg struct {
	// 每个分组配置
	Groups []VirtualGroup

	// 默认虚拟大区
	DefaultGroup int
}

type VirtualGroup struct {
	// 虚拟大区号 需要从1开始
	VGroupID int

	// 虚拟大区名称 显示用
	VGroupName string

	// 用于显示服务器序号的偏移量
	// 譬如设置为9000，则9011服会显示为11服
	ShowOffSet int

	// 起始服务器ID
	BeginShardID int

	// 结束服务器ID
	EndShardID int

	// 支持渠道
	Channels []int
}
