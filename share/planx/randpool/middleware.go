package randpool

import (
	"fmt"
	"strconv"
)

// RewardMiddleWare 奖励类型中间件
type RewardMiddleWare struct {
}

func (rm *RewardMiddleWare) ToData(s RandomSource) (RandomData, error) {
	rc := new(LootConfig)
	rc.RewardOutputType = RewardOutputMap[RewardType(s.GetRewardType())]
	rc.Loots = make([]*LootInfo, 0, len(s.GetParam()))

	for _, _p := range s.GetParam() {
		p, ok := _p.(*OutputRewardParam)
		if !ok {
			return nil, fmt.Errorf("%w, ID=%s, not OutputRewardParam: %v", ErrInvalidParams, s.GetID(), _p)
		}
		if p.RangeUp <= 0 && p.RangeDown <= 0 {
			return nil, fmt.Errorf("%w, ID=%s, RangeUp <= 0 && RangeDown <= 0, %v", ErrInvalidConfig, s.GetID(), _p)
		}

		l := &LootInfo{
			LootBase: LootBase{
				SubID:      p.GetSubID(),
				OutputID:   p.OutputID,
				Weight:     p.Weight,
				UpperLimit: p.RangeUp,
				LowerLimit: p.RangeDown,
			},
		}
		if p.Distribution > 0 {
			l.DistributionType = Density
			l.DistributionID = p.Distribution
		}
		rc.Loots = append(rc.Loots, l)
	}
	return rc, nil
}

// RepeatMiddleWare 重复类型中间件
type RepeatMiddleWare struct {
}

// ToData
func (rm *RepeatMiddleWare) ToData(s RandomSource) (RandomData, error) {
	rc := new(RepeatConfig)
	rc.RepeatData = make([]*Repeat, 0, len(s.GetParam()))

	for _, _p := range s.GetParam() {
		p, ok := _p.(*LinkRewardParam)
		if !ok {
			return nil, fmt.Errorf("%w, not LinkRewardParam: %v", ErrInvalidParams, _p)
		}
		rType := RewardRepeatMap[RewardType(s.GetRewardType())]
		if rType <= RepeatTypeNone || rType >= RepeatTypeCount {
			return nil, ErrInvalidRepeatType
		}

		rc.RepeatData = append(rc.RepeatData, &Repeat{
			RepeatType: rType,
			GroupID:    p.Goto,
			Amount:     uint32(p.Param),
		})
	}

	return rc, nil
}

// ShuffleMiddleware 乱序类型中间件
type ShuffleMiddleWare struct {
}

// ToData
func (sm *ShuffleMiddleWare) ToData(s RandomSource) (RandomData, error) {
	sc := new(ShuffleConfig)
	sc.ShuffleWeightData = make([]*ShuffleWeight, 0, len(s.GetParam()))

	for _, _p := range s.GetParam() {
		p, ok := _p.(*LinkRewardParam)
		if !ok {
			return nil, fmt.Errorf("%w, not LinkRewardParam: %v", ErrInvalidParams, _p)
		}
		if p.Factor == 0 || p.Param == 0 || p.Goto == "" {
			continue
		}

		sc.ShuffleWeightData = append(sc.ShuffleWeightData, &ShuffleWeight{
			Copy:    p.Factor,
			Weight:  p.Param,
			GroupID: p.Goto,
		})
	}

	return sc, nil
}

// RandomMiddleWare 区间类型中间件
type RandomMiddleWare struct {
}

// ToData
func (rm *RandomMiddleWare) ToData(s RandomSource) (RandomData, error) {
	rc := new(RandomConfig)
	rc.RandomWeightData = make([]*RandomWeight, 0, len(s.GetParam()))

	for _, _p := range s.GetParam() {
		p, ok := _p.(*LinkRewardParam)
		if !ok {
			return nil, fmt.Errorf("%w, not LinkRewardParam: %v", ErrInvalidParams, _p)
		}
		rc.RandomWeightData = append(rc.RandomWeightData, &RandomWeight{
			Weight:  p.Param,
			GroupID: p.Goto,
		})
		rc.SumWeight += p.Param
	}

	return rc, nil
}

// RangeMiddleWare 区间类型中间件
type RangeMiddleWare struct {
}

// ToData
func (rm *RangeMiddleWare) ToData(s RandomSource) (RandomData, error) {
	rc := new(RangeConfig)
	rc.Sections = make([]int64, 0, DefaultTmpMapLength)
	rc.GroupIDs = make([]string, 0, DefaultTmpMapLength)

	rangeMap := make(map[int64]string, DefaultTmpMapLength)

	for _, _p := range s.GetParam() {
		p, ok := _p.(*LinkRewardParam)
		if !ok {
			return nil, fmt.Errorf("%w, not LinkRewardParam: %v", ErrInvalidParams, _p)
		}
		if p.Goto == TimesReset {
			rc.MaxSection = p.Param
			_rv, err := strconv.ParseInt(p.GotoPage, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w, reset GotoPage not int: %v", ErrInvalidParams, _p)
			}
			rc.ResetValue = _rv
		}
		rangeMap[p.Param] = p.Goto
	}

	if err := rc.parseSortedDataFromSrc(rangeMap); err != nil {
		return nil, err
	}

	return rc, nil
}

// Seed类型中间件
type SeedMiddleWare struct {
}

func (rm *SeedMiddleWare) ToData(s RandomSource) (RandomData, error) {
	rc := new(SeedConfig)
	if len(s.GetParam()) != 1 {
		return nil, fmt.Errorf("%w, %s param != 1", ErrInvalidParams, s.GetID())
	}
	for _, _p := range s.GetParam() {
		p, ok := _p.(*LinkRewardParam)
		if !ok {
			return nil, fmt.Errorf("%w, not LinkRewardParam: %v", ErrInvalidParams, _p)
		}
		if len(p.PoolRange) != 2 {
			return nil, fmt.Errorf("%w, seed PoolRange invalid: %v", ErrInvalidParams, _p)
		}
		if len(p.WeightRange) != 2 {
			return nil, fmt.Errorf("%w, seed WeightRange invalid: %v", ErrInvalidParams, _p)
		}
		if p.HitLink == "" {
			return nil, fmt.Errorf("%w, HitLink nil: %v", ErrInvalidParams, _p)
		}
		rc.SumLowLimit = p.PoolRange[0]
		rc.SumUpperLimit = p.PoolRange[1]
		rc.PrizeLowLimit = p.WeightRange[0]
		rc.PrizeUpperLimit = p.WeightRange[1]
		rc.Groups = make([]string, 2)
		rc.Groups[0] = p.HitLink
		rc.Groups[1] = p.MissLink
	}
	return rc, nil
}

// SeedTimes类型中间件
type SeedTimesMiddleWare struct {
}

func (rm *SeedTimesMiddleWare) ToData(s RandomSource) (RandomData, error) {
	rc := new(SeedTimesConfig)
	if len(s.GetParam()) != 1 {
		return nil, fmt.Errorf("%w, %s param != 1", ErrInvalidParams, s.GetID())
	}
	for _, _p := range s.GetParam() {
		p, ok := _p.(*LinkRewardParam)
		if !ok {
			return nil, fmt.Errorf("%w, not LinkRewardParam: %v", ErrInvalidParams, _p)
		}
		if len(p.WeightRange) <= 1 {
			return nil, fmt.Errorf("%w, seed WeightRange invalid: %v", ErrInvalidParams, _p)
		}
		if p.HitLink == "" {
			return nil, fmt.Errorf("%w, HitLink nil: %v", ErrInvalidParams, _p)
		}
		rc.WeightRange = p.WeightRange
		for _, w := range rc.WeightRange {
			rc.SumWeight += w
		}
		rc.Groups = make([]string, 2)
		rc.Groups[0] = p.HitLink
		rc.Groups[1] = p.MissLink
	}
	return rc, nil
}
