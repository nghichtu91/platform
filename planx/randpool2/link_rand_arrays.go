package randpool2

import (
	"fmt"
	"math/rand"
)

func newLinkArrays(id string, rss []IRewardLink, cep *counterExpireParam) *linkArrays {
	ret := &linkArrays{
		id:   id,
		data: rss,
		randArraysC: &randArraysC{
			counterExpireParam: cep,
		},
	}
	return ret
}

type linkArrays struct {
	id   string
	data []IRewardLink
	*randArraysC
}

func (r *linkArrays) init() error {
	if r.counterExpireParam == nil {
		return fmt.Errorf("%w id %s ExpireParam is nil", ErrInvalidConfig, r.id)
	}
	r.randArraysC.init(r.data)
	//if r.paramInitV.totalWeight <= 0 {
	//	return fmt.Errorf("%w id %s totalWeight<=0", ErrInvalidConfig, r.id)
	//}
	return nil
}

func (r *linkArrays) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	res := &result{}
	id := r.id
	if activityId != "" {
		id = fmt.Sprintf("%s:%s", r.id, activityId)
	}

	n := r.getCount(id, counters, info.GetTimeStamp(), info.GetRand())
	if n < 0 {
		return res, nil
	}
	res.LinkRandomID = r.data[n].GetGoto()
	return res, nil
}

// rand arrays 计数器
type randArraysC struct {
	*counterExpireParam
	paramInitV *randArraysCParam
}

type randArraysCParam struct {
	totalWeight int64    // 用于取随机数
	copies      []uint32 // 原始份数
	weights     []int64  // 原始权重
	wWeights    []int64  // 加权后的权重
	weightIndex []int32  // 掉落组的序号
}

func (c *randArraysC) init(rss []IRewardLink) {
	n := len(rss)
	c.paramInitV = &randArraysCParam{}
	c.paramInitV.weightIndex = make([]int32, 0, n)
	c.paramInitV.weights = make([]int64, 0, n)
	c.paramInitV.wWeights = make([]int64, 0, n)
	c.paramInitV.copies = make([]uint32, 0, n)

	for idx, wd := range rss {
		if wd.GetParam() == 0 || wd.GetGoto() == "" {
			continue
		}

		c.paramInitV.weightIndex = append(c.paramInitV.weightIndex, int32(idx))
		c.paramInitV.weights = append(c.paramInitV.weights, int64(wd.GetParam()))
		c.paramInitV.wWeights = append(c.paramInitV.wWeights, int64(wd.GetFactor())*int64(wd.GetParam()))
		c.paramInitV.copies = append(c.paramInitV.copies, wd.GetFactor())
		c.paramInitV.totalWeight += int64(wd.GetParam()) * int64(wd.GetFactor())
	}
}

func (c *randArraysC) reset(counterData ICounterData) {
	wi := counterData.GetWeightIndex()
	wi.Clear()
	wi.Appends(c.paramInitV.weightIndex)

	ws := counterData.GetWeights()
	ws.Clear()
	ws.Appends(c.paramInitV.weights)

	wws := counterData.GetWWeights()
	wws.Clear()
	wws.Appends(c.paramInitV.wWeights)

	cs := counterData.GetCopies()
	cs.Clear()
	cs.Appends(c.paramInitV.copies)

	counterData.SetTotalWeight(c.paramInitV.totalWeight)
}

func (c *randArraysC) getCount(id string, mgr ICounterMgr, tNow int64, r *rand.Rand) int {
	counterData := c.refresh(id, mgr, tNow)
	if counterData.GetCopies().Len() <= 0 {
		c.reset(counterData)
	}

	if counterData.GetTotalWeight() <= 0 {
		return -1
	}

	var n, idx int

	// 随机
	num := r.Int63n(counterData.GetTotalWeight()) + 1

	// 这里判断应该落到哪个区间
	l := counterData.GetCopies().Len()
	for i := 0; i < l; i++ {
		ww, _ := counterData.GetWWeights().Get(i)
		num -= ww
		if num <= 0 {
			wi, _ := counterData.GetWeightIndex().Get(i)
			n = int(wi)
			idx = i
			break
		}
	}

	// 计数，把抽到的去掉
	_cv, _ := counterData.GetCopies().Get(idx)
	counterData.GetCopies().Set(idx, _cv-1)
	_wwv, _ := counterData.GetWWeights().Get(idx)
	_ws, _ := counterData.GetWeights().Get(idx)
	counterData.GetWWeights().Set(idx, _wwv-_ws)
	counterData.SetTotalWeight(counterData.GetTotalWeight() - _ws)

	// 清空降到0的组，这部分其实比较耗时，并且换成map性能反而会降低
	_cv, _ = counterData.GetCopies().Get(idx)
	if _cv <= 0 {
		counterData.GetCopies().Remove(idx)
		counterData.GetWeights().Remove(idx)
		counterData.GetWWeights().Remove(idx)
		counterData.GetWeightIndex().Remove(idx)
	}
	return n
}
