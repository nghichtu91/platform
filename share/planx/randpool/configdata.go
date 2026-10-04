package randpool

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/planx/timeutil"
)

/*
	Data是固定的掉落配置
	包括两部分，Config和Data
	Config用于管理Data，通过Set和Get接口实现
*/

// IRandomConfig 存储RandomData
type IRandomConfig interface {
	SetData(id string, rd RandomData)
	GetData(id string) RandomData
}

type IRandomConfigDistribution interface {
	SetDistributionData(id uint32, d []float32)
	GetDistributionData(id uint32) []float32
}

// DefaultRandomConfig 默认的RandomConfig，带锁
type DefaultRandomConfig struct {
	Data map[string]RandomData

	mutex sync.Mutex
}

func DefaultConfig() *DefaultRandomConfig {
	return &DefaultRandomConfig{
		Data: make(map[string]RandomData),
	}
}

// GetData 获取对应配置
func (rc *DefaultRandomConfig) SetData(id string, rd RandomData) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	rc.Data[id] = rd
}

// GetData 获取对应配置
func (rc *DefaultRandomConfig) GetData(id string) RandomData {
	if d, ok := rc.Data[id]; ok {
		return d
	}

	return nil
}

// RandomData 单条随机数据
type RandomData interface {
	SetBaseInfo(bi *BaseInfo)                                       // 设置基础信息
	GetType() RandomType                                            // 获取随机类型
	GenCounter(r *rand.Rand, dailyRefreshTime string) RandomCounter // 生成计数器
}

// BaseInfo 所有config都需要的字段
type BaseInfo struct {
	ID string
	RandomType
	RangeVariantType

	// 过期相关设置
	ExpireType
	ExpiredTimeValue int64 // 类型为Util时，这里是时间戳，类型为activate时，为每次延长时间, 为DateAssign时，为保留多少天

}

func (bi *BaseInfo) GenCounter(r *rand.Rand, dailyRefreshTime string) RandomCounter {
	ec := ExpireCounter{
		ExpireTime: ExpireTimeNever,
	}
	switch bi.ExpireType {
	case ExpireTypeUntil:
		ec.ExpireTime = bi.ExpiredTimeValue
	case ExpireTypeActivate:
		ec.ExpireTime = time.Now().Unix() + bi.ExpiredTimeValue
	case ExpireTypeDaily:
		ec.ExpireTime = timeutil.NextDailyBeginUnix(dailyRefreshTime)
	case ExpireTypeDateAssign:
		ec.ExpireTime = timeutil.TodayBeginUnix(dailyRefreshTime) + bi.ExpiredTimeValue*timeutil.DaySec
	}

	switch bi.RandomType {
	case RandomTypeRange:
		return &TimesCounter{
			ExpireCounter: ec,
		}
	case RandomTypeShuffle:
		return &ShuffleCounter{
			ExpireCounter: ec,
		}
	case RandomTypeSeed:
		return &SeedCounter{
			ExpireCounter: ec,
		}
	case RandomTypeSeedTimes:
		return &SeedTimesCounter{
			ExpireCounter: ec,
		}
	}

	return nil
}

func (bi *BaseInfo) GetType() RandomType {
	return bi.RandomType
}

func (bi *BaseInfo) SetBaseInfo(newBi *BaseInfo) {
	*bi = *newBi
}

// Loot 最终生成的掉落
type Loot struct {
	ItemID uint32
	Count  uint32
}

// RepeatConfig 重复执行多个或多次随机的配置
type RepeatConfig struct {
	BaseInfo
	RepeatData []*Repeat
}

// Repeat 单个repeat项配置
type Repeat struct {
	RepeatType
	GroupID string
	Amount  uint32
}

// RandomConfig 真随机
type RandomConfig struct {
	BaseInfo
	RandomWeightData []*RandomWeight
	SumWeight        int64
}

type RandomWeight struct {
	Weight  int64 // 比重
	GroupID string
}

func (rc *RandomConfig) Exe(r *rand.Rand) string {
	rn := r.Int63n(rc.SumWeight) + 1
	for _, w := range rc.RandomWeightData {
		rn -= w.Weight
		if rn <= 0 {
			return w.GroupID
		}
	}
	return ""
}

// ShuffleConfig 乱序抽取配置
type ShuffleConfig struct {
	BaseInfo
	ShuffleWeightData []*ShuffleWeight
}

// ShuffleWeight 权重配置
type ShuffleWeight struct {
	Copy    uint32 // 几份
	Weight  int64  // 比重
	GroupID string
}

// GenShuffleData 生成计数器初始化数据
func (sc *ShuffleConfig) GenShuffleData() ShuffleData {
	sd := ShuffleData{}

	n := len(sc.ShuffleWeightData)

	sd.WeightIndex = make([]int, 0, n)
	sd.Weights = make([]int64, 0, n)
	sd.WWeights = make([]int64, 0, n)
	sd.Copies = make([]uint32, 0, n)

	for idx, wd := range sc.ShuffleWeightData {
		if wd.Copy == 0 || wd.Weight == 0 || wd.GroupID == "" {
			continue
		}

		sd.WeightIndex = append(sd.WeightIndex, idx)
		sd.Weights = append(sd.Weights, wd.Weight)
		sd.WWeights = append(sd.WWeights, int64(wd.Copy)*wd.Weight)
		sd.Copies = append(sd.Copies, wd.Copy)
		sd.TotalWeight += wd.Weight * int64(wd.Copy)
	}

	return sd
}

// GenCounter 写入shuffle数据
func (sc *ShuffleConfig) GenCounter(r *rand.Rand, dailyRefreshTime string) RandomCounter {
	c := sc.BaseInfo.GenCounter(r, dailyRefreshTime)
	switch c.(type) {
	case *ShuffleCounter:
		nc := c.(*ShuffleCounter)
		nc.ShuffleData = sc.GenShuffleData()
		return nc
	}
	return c
}

// RangeConfig 区间抽取配置，参数+真随机
type RangeConfig struct {
	BaseInfo
	Range
}

// Range 区间配置
// 考虑到区间一般不超过8个，slice足够快，但依然需要策划注意区间数量问题
type Range struct {
	Sections   []int64  // 区间节点
	GroupIDs   []string // 节点对应的跳转ID
	MaxSection int64    // 区间上界，同时也是重置节点
	ResetValue int64    // 重置到的数量，大于0时有效
}

func (rc *RangeConfig) GenCounter(r *rand.Rand, dailyRefreshTime string) RandomCounter {
	c := rc.BaseInfo.GenCounter(r, dailyRefreshTime)
	switch c.(type) {
	case *TimesCounter:
		tc := c.(*TimesCounter)
		if rc.ResetValue != 0 && rc.MaxSection != 0 {
			return &TimesResetCounter{
				ExpireCounter: tc.ExpireCounter,
				ResetPoint:    int(rc.MaxSection),
				ResetValue:    int(rc.ResetValue),
			}
		}
		return tc
	}
	return c
}

// parseSortedDataFromSrc 将json解析出的map整理排序，返回已排序的区间以及对应值
func (rc *RangeConfig) parseSortedDataFromSrc(data map[int64]string) error {
	sectionMap := make(map[int64]string, DefaultTmpMapLength)

	for k, v := range data {
		rc.Sections = append(rc.Sections, k)
		sectionMap[k] = v
	}

	util.Int64s(rc.Sections)

	for _, section := range rc.Sections {
		rc.GroupIDs = append(rc.GroupIDs, sectionMap[section])
	}

	return nil
}

// Exe 快捷执行方法，获得结果不计数
// 用于抽取不放回以及随机
func (r Range) Exe(param int) string {
	for i := 0; i < len(r.Sections); i++ {
		if int64(param) <= r.Sections[i] {
			return r.GroupIDs[i]
		}
	}

	return r.GroupIDs[len(r.GroupIDs)-1]
}

// LootV2Config 第二版的掉落
type LootConfig struct {
	BaseInfo
	RewardOutputType
	Loots []*LootInfo
}

// Loot
type LootInfo struct {
	LootBase
	DistributionType
	DistributionID uint32
}

// LootBase
type LootBase struct {
	SubID      uint32
	OutputID   uint32
	Weight     uint32
	UpperLimit uint32
	LowerLimit uint32
}

// GetLoots 生成掉落
func (lc2 *LootConfig) GetLoots(r *rand.Rand, d IRandomConfigDistribution) ([]*Loot, error) {
	loots := make([]*Loot, 0, len(lc2.Loots))
	switch lc2.RewardOutputType {
	case RewardOutputStatic:
		for _, l := range lc2.Loots {
			loot, err := lc2._GenLoot(r, l, d)
			if err != nil {
				return nil, err
			}
			loots = append(loots, loot)
		}
	case RewardOutputWeight:
		sw := 0
		for _, l := range lc2.Loots {
			sw += int(l.Weight)
		}
		_n := r.Intn(sw) + 1
		var l *LootInfo
		for i, l := range lc2.Loots {
			_n -= int(l.Weight)
			if _n <= 0 {
				l = lc2.Loots[i]
				break
			}
		}
		if l == nil {
			return nil, fmt.Errorf("%w ID %s", ErrLootFail, lc2.BaseInfo.ID)
		}
		loot, err := lc2._GenLoot(r, l, d)
		if err != nil {
			return nil, err
		}
		loots = append(loots, loot)
	}

	return loots, nil
}

func (lc2 *LootConfig) _GenLoot(r *rand.Rand, l *LootInfo, d IRandomConfigDistribution) (*Loot, error) {
	if l.LootBase.UpperLimit <= 0 {
		return &Loot{
			ItemID: l.LootBase.OutputID,
			Count:  l.LootBase.UpperLimit,
		}, nil
	}
	switch l.DistributionType {
	case Liner:
		n := uint32(r.Int31n(int32(l.LootBase.UpperLimit-l.LootBase.LowerLimit))) + l.LootBase.LowerLimit
		return &Loot{
			ItemID: l.LootBase.OutputID,
			Count:  n,
		}, nil
	case Density:
		dd := d.GetDistributionData(l.DistributionID)
		if dd == nil {
			return nil, fmt.Errorf("%w DistributionID %d not found", ErrInvalidConfig, l.DistributionID)
		}
		n := uint32(float32(l.LootBase.UpperLimit-l.LootBase.LowerLimit)*dd[r.Intn(len(dd))]) + l.LootBase.LowerLimit
		return &Loot{
			ItemID: l.LootBase.OutputID,
			Count:  n,
		}, nil
	}
	return nil, fmt.Errorf("%s DistributionType not found %d", lc2.ID, l.DistributionID)
}

type SeedConfig struct {
	BaseInfo
	SumLowLimit     uint32
	SumUpperLimit   uint32
	PrizeLowLimit   uint32
	PrizeUpperLimit uint32
	Groups          []string // 0:HitLink, 1:MissLink
}

func (sc *SeedConfig) GenCounter(r *rand.Rand, dailyRefreshTime string) RandomCounter {
	c := sc.BaseInfo.GenCounter(r, dailyRefreshTime)
	switch c.(type) {
	case *SeedCounter:
		nc := c.(*SeedCounter)
		nc.sumCount = uint32(r.Int63n(int64(sc.SumUpperLimit-sc.SumLowLimit))) + sc.SumLowLimit
		nc.prizeCount = uint32(r.Int63n(int64(sc.PrizeUpperLimit-sc.PrizeLowLimit))) + sc.PrizeLowLimit
		return nc
	}
	return c
}

func (sc *SeedConfig) Exe(param int) string {
	return sc.Groups[param]
}

type SeedTimesConfig struct {
	BaseInfo
	WeightRange []uint32
	SumWeight   uint32
	Groups      []string // 0:HitLink, 1:MissLink
}

func (sc *SeedTimesConfig) GenCounter(r *rand.Rand, dailyRefreshTime string) RandomCounter {
	c := sc.BaseInfo.GenCounter(r, dailyRefreshTime)
	switch c.(type) {
	case *SeedTimesCounter:
		nc := c.(*SeedTimesCounter)
		nc.sumCount = len(sc.WeightRange)
		rn := uint32(r.Int63n(int64(sc.SumWeight)) + 1)
		for i, w := range sc.WeightRange {
			rn -= w
			if rn <= 0 {
				nc.prizeIndex = i + 1
				break
			}
		}
		return nc
	}
	return c
}

func (sc *SeedTimesConfig) Exe(param int) string {
	return sc.Groups[param]
}
