package virtual_gid

import (
	"encoding/json"
	"fmt"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var (
	// 虚拟大区是否开启
	virtualGroupEnabled bool

	// 虚拟大区配置
	cfg *VirtualGroupCfg
)

const MinShardId = 10000

const (
	VirtualGid  = 1 // 无虚拟大区的全大区
	VirtualGidA = 2 // 有虚拟大区的虚A(官网)
	VirtualGidB = 3 // 有虚拟大区的虚B(渠道)
	VirtualGidC = 4 // 有虚拟大区的虚C(激战)
)

// Init 初始化虚拟大区，注意这里etcd地址是devops
func Init(devopsRoot string, gid uint) error {
	key := fmt.Sprintf("%s/%d/%s", devopsRoot, gid, etcd.VirtualGID)

	v, err := etcd.Get(key)
	if err != nil {
		return err
	}

	// 没有配置不开启虚拟大区
	if v == "" {
		tilogs.L().Infof("virtual group config not found, virtual group disabled")
		return nil
	}

	vc := &VirtualGroupCfg{}
	err = json.Unmarshal([]byte(v), vc)
	if err != nil {
		tilogs.L().Errorf("unmarshal virtual group config failed, err %s", err.Error())
		return err
	}

	virtualGroupEnabled = true
	cfg = vc
	cfg.DefaultGroup = VirtualGidB

	return nil
}

// IsVirtualGroupEnabled 虚拟大区是否开启
func IsVirtualGroupEnabled() bool {
	return virtualGroupEnabled
}

// GetVirtualGroupCfg 获取虚拟大区配置
func GetVirtualGroupCfg() *VirtualGroupCfg {
	return cfg
}

// GetVirtualGroupID 查找sid对应的虚拟大区ID
func GetVirtualGroupID(shard int) int {
	if !virtualGroupEnabled {
		return 0
	}

	for _, g := range cfg.Groups {
		if g.BeginShardID <= shard && shard <= g.EndShardID {
			return g.VGroupID
		}
	}

	return 0
}

// GetVirtualGroupIDByChannel 查找渠道对应的虚拟大区ID
func GetVirtualGroupIDByChannel(channel int) int {
	if !virtualGroupEnabled {
		return 0
	}

	for _, g := range cfg.Groups {
		for _, ch := range g.Channels {
			if ch == channel {
				return g.VGroupID
			}
		}
	}

	return cfg.DefaultGroup
}

// GetShardIDWithOffset 对ShardID进行偏移
// shard不在虚拟大区范围内，或者虚拟大区未开启，都返回shardID
func GetShardIDWithOffset(shard int) int {
	if shard/MinShardId == 0 {
		panic(fmt.Errorf("shards should greater than 10000, id : %d", shard))
	}
	if !virtualGroupEnabled {
		return shard
	}

	for _, g := range cfg.Groups {
		if g.BeginShardID <= shard && shard <= g.EndShardID {
			return shard - g.ShowOffSet
		}
	}

	return shard
}

// IsShardInChannel 检查Shard是否对指定渠道开放
// 虚拟大区不配置渠道时，匹配任意渠道
func IsShardInChannel(shard, channel int) bool {
	if channel == 0 {
		return true
	}

	for _, g := range cfg.Groups {
		if g.BeginShardID <= shard && shard <= g.EndShardID {
			// 没有配置渠道，对所有渠道开放
			if len(g.Channels) == 0 {
				return true
			}
			// 匹配渠道
			for _, ch := range g.Channels {
				if ch == channel {
					return true
				}
			}
			return false
		}
	}

	// 没有默认渠道，且此渠道号所有虚拟大区不存在
	return false
}

// GetGroupMaxShardID 获取对应虚拟大区分组的最大服务器ID
// 如果分组ID不存在或为0，返回0
func GetGroupMaxShardID(groupID int) int {
	for _, g := range cfg.Groups {
		if g.VGroupID == groupID {
			return g.EndShardID
		}
	}

	return 0
}
