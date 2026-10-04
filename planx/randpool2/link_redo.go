package randpool2

func newLinkRedo(id string, rss []IRewardLink) *LinkRedo {
	ret := &LinkRedo{
		id:      id,
		repeats: &repeats{},
	}
	ret.repeats.init(rss)
	return ret
}

type LinkRedo struct {
	id string
	*repeats
}

func (r *LinkRedo) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	res := &result{
		Loots: make([]*Loot, 0, DefaultTmpMapLength),
	}
	for i := range r.data {
		rp := r.data[i]
		var n uint32
		for ; n < rp.amount; n++ {
			loots, err := _Run(rp.groupID, counters, info, recursionC, activityId)
			if err != nil {
				return nil, err
			}
			res.Loots = mergeLoot(res.Loots, loots)
		}
	}
	return res, nil
}

type repeat struct {
	groupID string
	amount  uint32
}

type repeats struct {
	data []*repeat
}

func (rs *repeats) init(rss []IRewardLink) {
	rs.data = make([]*repeat, 0, len(rss))
	for _, rl := range rss {
		rs.data = append(rs.data, &repeat{
			groupID: rl.GetGoto(),
			amount:  uint32(rl.GetParam()),
		})
	}
}
