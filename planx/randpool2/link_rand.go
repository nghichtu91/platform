package randpool2

func newLinkRandom(id string, rss []IRewardLink) *linkRandom {
	ret := &linkRandom{
		id:   id,
		data: rss,
	}
	for _, r := range rss {
		ret.sumWeight += r.GetParam()
	}
	return ret
}

type linkRandom struct {
	id        string
	data      []IRewardLink
	sumWeight uint64
}

func (r *linkRandom) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	res := &result{}
	if r.sumWeight <= 0 {
		return res, nil
	}
	rn := info.GetRand().Int63n(int64(r.sumWeight)) + 1
	for _, w := range r.data {
		rn -= int64(w.GetParam())
		if rn <= 0 {
			res.LinkRandomID = w.GetGoto()
			break
		}
	}
	return res, nil
}
