package randpool

import (
	"math/rand"
)

/*
	Counter记录抽取状态
*/
type IRandomCounterData interface {
}

type IRandomTimesCounterData interface {
	GetExpireTime() int64
	GetCount() uint32
	SetExpireTime(v int64)
	SetCount(v uint32)
}

// RandomCounter 计数器
type RandomCounter interface {
	IsAvailable(tNow int64) bool // 判断Counter是否有效
	GetCount(r *rand.Rand) int   // 获取当前计数，同时counter增加一次记录。Times类型：返回计数 Shuffle类型：返回序号
}

// ExpireCounter 记录和计算过期时间
type ExpireCounter struct {
	ExpireTime int64
}

// IsAvailable 判断Counter是否有效
func (ec *ExpireCounter) IsAvailable(tNow int64) bool {
	return ec.ExpireTime == ExpireTimeNever || tNow < ec.ExpireTime
}

// GetCount 获取当前计数 ExpireCounter没有计数，永远返回-1
func (ec *ExpireCounter) GetCount(r *rand.Rand) int {
	return GetCountNotAvailable
}

// TimesCounter 次数状态记录
type TimesCounter struct {
	ExpireCounter
	Count int // 抽了多少次
}

// GetCount 获取当前计数，同时counter增加一次记录
// 注意计数是从1开始的，因此先加再发
func (tc *TimesCounter) GetCount(r *rand.Rand) int {
	tc.Count++
	return tc.Count
}

// TimesResetCounter 带重置的次数状态记录
type TimesResetCounter struct {
	ExpireCounter
	Count      int // 抽了多少次
	ResetPoint int // 第几次重置
	ResetValue int // 重置成几
}

// GetCount 获取当前计数，同时counter增加一次记录
// 注意计数是从1开始的，因此先加再发
func (trc *TimesResetCounter) GetCount(r *rand.Rand) int {
	trc.Count++
	if trc.Count >= trc.ResetPoint {
		trc.Count = trc.ResetValue
	}
	return trc.Count
}

// Shuffler 随机状态记录
type ShuffleCounter struct {
	ExpireCounter
	ShuffleData
}

// ShuffleData 乱序状态记录
type ShuffleData struct {
	TotalWeight int64    // 用于取随机数
	Copies      []uint32 // 原始份数
	Weights     []int64  // 原始权重
	WWeights    []int64  // 加权后的权重
	WeightIndex []int    // 掉落组的序号
}

// IsAvailable 判断Counter是否有效
func (sc *ShuffleCounter) IsAvailable(tNow int64) bool {
	return len(sc.Copies) > 0 && (sc.ExpireTime == ExpireTimeNever || tNow < sc.ExpireTime)
}

// GetCount 获取当前计数，同时counter增加一次记录
func (sc *ShuffleCounter) GetCount(r *rand.Rand) int {
	var id, idx int

	// 随机
	num := r.Int63n(sc.TotalWeight) + 1

	// 这里判断应该落到哪个区间
	for i := range sc.Copies {
		num -= sc.WWeights[i]
		if num <= 0 {
			id = sc.WeightIndex[i]
			idx = i
			break
		}
	}

	// 计数，把抽到的去掉
	sc.Copies[idx]--
	sc.WWeights[idx] -= sc.Weights[idx]
	sc.TotalWeight -= sc.Weights[idx]

	// 清空降到0的组，这部分其实比较耗时，并且换成map性能反而会降低
	if sc.Copies[idx] <= 0 {
		sc.Copies = append(sc.Copies[:idx], sc.Copies[idx+1:]...)
		sc.Weights = append(sc.Weights[:idx], sc.Weights[idx+1:]...)
		sc.WWeights = append(sc.WWeights[:idx], sc.WWeights[idx+1:]...)
		sc.WeightIndex = append(sc.WeightIndex[:idx], sc.WeightIndex[idx+1:]...)
	}

	return id
}

// Seed 计数器
type SeedCounter struct {
	ExpireCounter
	sumCount   uint32 // 抽奖总次数，为0重置
	prizeCount uint32 // 奖品次数，为0则没有奖品了
}

func (sc *SeedCounter) IsAvailable(tNow int64) bool {
	return sc.sumCount > 0 && (sc.ExpireTime == ExpireTimeNever || tNow < sc.ExpireTime)
}

func (sc *SeedCounter) GetCount(r *rand.Rand) int {
	n := uint32(r.Int63n(int64(sc.sumCount)) + 1)
	var ret int
	if n <= sc.prizeCount { // 中奖
		sc.prizeCount--
		ret = HitLink
	} else { // 未中奖
		ret = MissLink
	}
	sc.sumCount--
	return ret
}

// SeedTimes计数器
type SeedTimesCounter struct {
	ExpireCounter
	sumCount   int // 抽奖总次数
	prizeIndex int // 第几次是奖品
	curIndex   int // 当前抽到第几次了, 等于sumCount时重置
}

func (sc *SeedTimesCounter) IsAvailable(tNow int64) bool {
	return sc.sumCount <= sc.curIndex && (sc.ExpireTime == ExpireTimeNever || tNow < sc.ExpireTime)
}

func (sc *SeedTimesCounter) GetCount(r *rand.Rand) int {
	sc.curIndex++
	if sc.curIndex == sc.prizeIndex {
		return HitLink // 中奖
	}
	return MissLink // 未中奖
}
