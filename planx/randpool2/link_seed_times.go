package randpool2

import (
	"fmt"
	"math/rand"
)

func newLinkSeedTimes(id string, rs IRewardLink, cep *counterExpireParam) *linkSeedTimes {
	ret := &linkSeedTimes{
		id:   id,
		data: rs,
		seedTimesC: &seedTimesC{
			counterExpireParam: cep,
			data:               rs,
		},
	}
	return ret
}

type linkSeedTimes struct {
	id   string
	data IRewardLink
	*seedTimesC
}

func (r *linkSeedTimes) init() error {
	if r.counterExpireParam == nil {
		return fmt.Errorf("%w id %s ExpireParam is nil", ErrInvalidConfig, r.id)
	}
	if len(r.data.GetWeightRange()) <= 0 {
		return fmt.Errorf("%w id %s, seed WeightRange invalid: %v", ErrInvalidParams, r.id, r.data)
	}
	for _, _w := range r.data.GetWeightRange() {
		if _w <= 0 {
			return fmt.Errorf("%w id %s, seed WeightRange <=0: %v", ErrInvalidParams, r.id, r.data)
		}
	}
	if r.data.GetHitLink() == "" {
		return fmt.Errorf("%w id %s, HitLink nil: %v", ErrInvalidParams, r.id, r.data)
	}
	if r.data.GetMissLink() == "" {
		return fmt.Errorf("%w id %s, MissLink nil: %v", ErrInvalidParams, r.id, r.data)
	}
	for _, w := range r.data.GetWeightRange() {
		r.SumWeight += w
	}
	if r.SumWeight <= 0 {
		return fmt.Errorf("%w id %s, SumWeight==0: %v", ErrInvalidParams, r.id, r.data)
	}
	r.SumCount = uint32(len(r.data.GetWeightRange()))
	return nil
}

func (r *linkSeedTimes) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	res := &result{}
	id := r.id
	if activityId != "" {
		id = fmt.Sprintf("%s:%s", r.id, activityId)
	}

	n := r.getCount(id, counters, info.GetTimeStamp(), info.GetRand())
	switch n {
	case HitLink:
		res.LinkRandomID = r.data.GetHitLink()
	case MissLink:
		res.LinkRandomID = r.data.GetMissLink()
	default:
		return nil, fmt.Errorf("%w id %s seed run res not define, %d", ErrNoResult, r.id, n)
	}
	return res, nil
}

// seed定时器
type seedTimesC struct {
	*counterExpireParam
	data      IRewardLink
	SumWeight uint32
	SumCount  uint32
}

func (c *seedTimesC) reset(r *rand.Rand, counterData ICounterData) {
	rn := uint32(r.Int63n(int64(c.SumWeight)) + 1)
	for i, w := range c.data.GetWeightRange() {
		rn -= w
		if rn <= 0 {
			counterData.SetPrizeIndex(uint32(i + 1))
			break
		}
	}
}

func (c *seedTimesC) getCount(id string, mgr ICounterMgr, tNow int64, r *rand.Rand) HitMissIdx {
	counterData := c.refresh(id, mgr, tNow)
	if counterData.GetPrizeIndex() <= 0 || counterData.GetCurIndex() >= c.SumCount {
		c.reset(r, counterData)
	}
	n := counterData.GetCurIndex() + 1
	counterData.SetCurIndex(n)
	if n == counterData.GetPrizeIndex() {
		return HitLink // 中奖
	}
	return MissLink // 未中奖
}
