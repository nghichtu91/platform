package randpool2

import (
	"fmt"
	"math/rand"
)

func newLinkSeed(id string, rs IRewardLink, cep *counterExpireParam) *linkSeed {
	ret := &linkSeed{
		id:   id,
		data: rs,
		seedC: &seedC{
			counterExpireParam: cep,
			data:               rs,
		},
	}
	return ret
}

type linkSeed struct {
	id   string
	data IRewardLink
	*seedC
}

func (r *linkSeed) init() error {
	if r.counterExpireParam == nil {
		return fmt.Errorf("%w id %s ExpireParam is nil", ErrInvalidConfig, r.id)
	}
	if len(r.data.GetPoolRange()) != 2 {
		return fmt.Errorf("%w id %s, seed PoolRange len !=2: %v", ErrInvalidParams, r.id, r.data)
	}
	if r.data.GetPoolRange()[0] >= r.data.GetPoolRange()[1] {
		return fmt.Errorf("%w id %s, seed PoolRange [0] >= [1]: %v", ErrInvalidParams, r.id, r.data)
	}
	if len(r.data.GetWeightRange()) != 2 {
		return fmt.Errorf("%w id %s, seed WeightRange len !=2: %v", ErrInvalidParams, r.id, r.data)
	}
	if r.data.GetPoolRange()[0] > r.data.GetPoolRange()[1] {
		return fmt.Errorf("%w id %s, seed WeightRange [0] > [1]: %v", ErrInvalidParams, r.id, r.data)
	}
	if r.data.GetHitLink() == "" {
		return fmt.Errorf("%w id %s, HitLink nil: %v", ErrInvalidParams, r.id, r.data)
	}
	if r.data.GetMissLink() == "" {
		return fmt.Errorf("%w id %s, MissLink nil: %v", ErrInvalidParams, r.id, r.data)
	}
	return nil
}

func (r *linkSeed) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
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
type seedC struct {
	*counterExpireParam
	data IRewardLink
}

func (c *seedC) reset(r *rand.Rand, counterData ICounterData) {
	sumLowLimit := c.data.GetPoolRange()[0]
	sumUpperLimit := c.data.GetPoolRange()[1]
	prizeLowLimit := c.data.GetWeightRange()[0]
	prizeUpperLimit := c.data.GetWeightRange()[1]
	if sumUpperLimit == sumLowLimit {
		counterData.SetSumCount(sumLowLimit)
	} else {
		counterData.SetSumCount(uint32(r.Int63n(int64(sumUpperLimit-sumLowLimit))) + sumLowLimit)
	}
	if prizeUpperLimit == prizeLowLimit {
		counterData.SetPrizeCount(prizeLowLimit)
	} else {
		counterData.SetPrizeCount(uint32(r.Int63n(int64(prizeUpperLimit-prizeLowLimit))) + prizeLowLimit)
	}
}

func (c *seedC) getCount(id string, mgr ICounterMgr, tNow int64, r *rand.Rand) HitMissIdx {
	counterData := c.refresh(id, mgr, tNow)
	if counterData.GetSumCount() <= 0 {
		c.reset(r, counterData)
	}

	n := uint32(r.Int63n(int64(counterData.GetSumCount())) + 1)
	var ret HitMissIdx
	if n <= counterData.GetPrizeCount() { // 中奖
		counterData.SetPrizeCount(counterData.GetPrizeCount() - 1)
		ret = HitLink
	} else { // 未中奖
		ret = MissLink
	}
	counterData.SetSumCount(counterData.GetSumCount() - 1)
	return ret
}

type HitMissIdx int

const (
	HitLink  HitMissIdx = iota // 0
	MissLink                   // 1
)
