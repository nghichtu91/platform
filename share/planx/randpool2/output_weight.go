package randpool2

func NewOutputWeight(id string, rws []IRewardWeightOutput) *OutputWeight {
	_ow := &OutputWeight{
		id:   id,
		data: rws,
	}
	for _, _rw := range rws {
		_ow.sumWeight += _rw.GetWeight()
	}
	return _ow
}

type OutputWeight struct {
	id        string
	data      []IRewardWeightOutput
	sumWeight uint32
}

func (r *OutputWeight) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	res := &result{
		Loots: make([]*Loot, 0, DefaultTmpMapLength),
	}
	if r.sumWeight <= 0 {
		return res, nil
	}
	_n := info.GetRand().Int63n(int64(r.sumWeight)) + 1
	var _rw IRewardWeightOutput
	for _, rw := range r.data {
		_n -= int64(rw.GetWeight())
		if _n <= 0 {
			_rw = rw
			break
		}
	}
	l := genLoot(_rw, info)
	if l.Count > 0 {
		res.Loots = append(res.Loots, l)
	}
	return res, nil
}
