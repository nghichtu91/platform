package randpool2

func newLinkMultiple(id string, rss []IRewardLink) *linkMultiple {
	ret := &linkMultiple{
		id:      id,
		repeats: &repeats{},
	}
	ret.repeats.init(rss)
	return ret
}

type linkMultiple struct {
	id string
	*repeats
}

func (r *linkMultiple) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	res := &result{
		Loots: make([]*Loot, 0, DefaultTmpMapLength),
	}
	for i := range r.data {
		rp := r.data[i]
		if rp.amount <= 0 {
			continue
		}
		loots, err := _Run(rp.groupID, counters, info, recursionC, activityId)
		if err != nil {
			return nil, err
		}
		for i := range loots {
			l := loots[i]
			l.Count = l.Count * rp.amount
			res.Loots = append(res.Loots, l)
		}
	}
	return res, nil
}
