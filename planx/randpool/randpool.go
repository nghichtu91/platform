package randpool

import (
	"fmt"
	"math/rand"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

// RandomContext 是运行随机过程需要的参数
type RandomContext struct {
	IRandomConfig
	IRandomConfigDistribution
	IRandomProfile
	IRandomParams
}

// Run 掉落入口
func Run(id string, rc RandomContext) (*RandomResult, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	if rc.IRandomConfig == nil {
		return nil, ErrInvalidConfig
	}

	if rc.IRandomProfile == nil {
		return nil, ErrInvalidProfile
	}

	if rc.IRandomParams == nil {
		return nil, ErrInvalidParams
	}

	result := DefaultRandomResult(rc.IRandomParams.GetRand())
	err := result.RunRecursively(id, rc)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// RunRecursively 递归执行
func (rr *RandomResult) RunRecursively(id string, rc RandomContext) error {
	if id == "" {
		return nil
	}
	data := rc.GetData(id)
	if data == nil {
		tilogs.L().Errorf("id %s not found", id)
		return ErrInvalidID
	}

	if rr.AddRecursionCount(1) != nil {
		return ErrMaxRecursionExceed
	}

	switch data.GetType() {
	case RandomTypeReward:
		ld, ok := data.(LootData)
		if !ok {
			return ErrInvalidLootData
		}
		ll, err := ld.GetLoots(rc.GetRand(), rc.IRandomConfigDistribution)
		if err != nil {
			return err
		}
		rr.AddLoots(ll)
		return nil
	case RandomTypeRepeat:
		for _, d := range data.(*RepeatConfig).RepeatData {
			newRr := DefaultRandomResult(rc.GetRand())
			// 这里不处理异常是因为newRr递归计数为0，不太可能报错
			newRr.AddRecursionCount(rr.GetRecursionCount())
			switch d.RepeatType {
			case RepeatTypeNone:
				return ErrInvalidRepeatType
			case RepeatTypeMultiple:
				err := newRr.RunRecursively(d.GroupID, rc)
				if err != nil {
					return err
				}
				newRr.Multiple(d.Amount)
			case RepeatTypeRepeat:
				var i uint32
				for ; i < d.Amount; i++ {
					err := newRr.RunRecursively(d.GroupID, rc)
					if err != nil {
						return err
					}
				}
			}
			rr.AddOthers(newRr)
		}
		return nil
	case RandomTypeRandom:
		groupID := data.(*RandomConfig).Exe(rc.GetRand())
		if groupID == "" {
			return fmt.Errorf("RunRecursively id %s RandomConfig groupID nil", id)
		}
		return rr.RunRecursively(groupID, rc)
	case RandomTypeShuffle:
		counter, err := HandleCounter(id, rc.GetTimestamp(), rc.IRandomProfile, data, rc.GetRand(), rc.GetDailyRefreshTime())
		if err != nil {
			return err
		}
		wIdx := counter.GetCount(rc.GetRand())
		wd := data.(*ShuffleConfig).ShuffleWeightData
		//if wIdx >= len(wd) {
		//	// TODO：如果热更数据导致配置变化，在这里做处理，
		//}
		return rr.RunRecursively(wd[wIdx].GroupID, rc)
	case RandomTypeRange:
		rd := data.(*RangeConfig)
		switch rd.RangeVariantType {
		case Times:
			counter, err := HandleCounter(id, rc.GetTimestamp(), rc.IRandomProfile, data, rc.GetRand(), rc.GetDailyRefreshTime())
			if err != nil {
				return err
			}
			return rr.RunRecursively(rd.Exe(counter.GetCount(rc.GetRand())), rc)
		// TODO: 将来有其他特殊类型参数需要处理，在这里添加
		default:
			return rr.RunRecursively(rd.Exe(rc.GetValue(rd.RangeVariantType)), rc)
		}
	case RandomTypeSeed:
		rd := data.(*SeedConfig)
		counter, err := HandleCounter(id, rc.GetTimestamp(), rc.IRandomProfile, data, rc.GetRand(), rc.GetDailyRefreshTime())
		if err != nil {
			return err
		}
		return rr.RunRecursively(rd.Exe(counter.GetCount(rc.GetRand())), rc)
	case RandomTypeSeedTimes:
		rd := data.(*SeedTimesConfig)
		counter, err := HandleCounter(id, rc.GetTimestamp(), rc.IRandomProfile, data, rc.GetRand(), rc.GetDailyRefreshTime())
		if err != nil {
			return err
		}
		return rr.RunRecursively(rd.Exe(counter.GetCount(rc.GetRand())), rc)

	}

	return ErrInvalidRandomData
}

// HandleCounter 处理counter和profile之间关系 TODO 需要有,没有计数器的逻辑
func HandleCounter(id string, ts int64, profile IRandomProfile, data RandomData, r *rand.Rand, dailyRefreshTime string) (RandomCounter, error) {
	counter := profile.GetCounter(id)

	if counter != nil {
		if counter.IsAvailable(ts) {
			return counter, nil
		}

		profile.DelCounter(id)
	}

	counter = data.GenCounter(r, dailyRefreshTime)
	if counter == nil || !counter.IsAvailable(ts) {
		tilogs.L().Errorf("id %s counter not found or expired", id)
		return nil, ErrInvalidID
	}

	switch counter.(type) {
	case *ExpireCounter:
		if counter.(*ExpireCounter).ExpireTime == ExpireTimeNever { // ExpireTimeNever 是Random真随机用，没必要计数器
			return nil, nil
		}
	}

	profile.SetCounter(id, counter)

	return counter, nil
}
