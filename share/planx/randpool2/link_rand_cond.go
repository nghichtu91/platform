package randpool2

func newLinkRandomCondition(id string, rss []IRewardLink) *linkRandomCondition {
	ret := &linkRandomCondition{
		id:   id,
		data: rss,
	}
	return ret
}

type linkRandomCondition struct {
	id   string
	data []IRewardLink
}

func (r *linkRandomCondition) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	var sumWeight uint64
	res := &result{}
	_data := make([]IRewardLink, 0, len(r.data))
	for _, _r := range r.data {
		if _r.GetConditionID() > 0 {
			if info.FitCond(_r.GetConditionID()) {
				sumWeight += _r.GetParam()
				_data = append(_data, _r)
			}
		} else {
			sumWeight += _r.GetParam()
			_data = append(_data, _r)
		}
	}
	if sumWeight <= 0 {
		return res, nil
	}
	rn := info.GetRand().Int63n(int64(sumWeight)) + 1
	for _, w := range _data {
		rn -= int64(w.GetParam())
		if rn <= 0 {
			res.LinkRandomID = w.GetGoto()
			break
		}
	}
	return res, nil
}
