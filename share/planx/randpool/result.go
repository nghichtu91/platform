package randpool

import (
	"math/rand"
)

/*
	result包括传出的掉落和参数
*/

// RandomResult Counter运行后返回的结果
type RandomResult struct {
	Loots []*Loot
	IRandomParams
}

func DefaultRandomResult(r *rand.Rand) *RandomResult {
	return &RandomResult{
		IRandomParams: DefaultParam(r),
	}
}

// AddRewards 添加固定掉落
// 注意不要直接添加指针，否则后面乘会改变初始指针的值
func (rr *RandomResult) AddLoots(rewards []*Loot) {
	for i := range rewards {
		if rewards[i] == nil {
			// 一般不会出现这种情况
			continue
		}
		newLoot := new(Loot)
		*newLoot = *rewards[i]
		rr.Loots = append(rr.Loots, newLoot)
	}
}

// AddOthers 添加其他结果的掉落
func (rr *RandomResult) AddOthers(other *RandomResult) {
	rr.AddLoots(other.Loots)
}

// AddOthers 掉落数量乘n
func (rr *RandomResult) Multiple(n uint32) {
	for i := 0; i < len(rr.Loots); i++ {
		rr.Loots[i].Count *= n
	}
}

func (rr *RandomResult) GetLoots() []*Loot {
	return rr.Loots
}
