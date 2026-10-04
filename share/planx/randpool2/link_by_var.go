package randpool2

import "fmt"

func NewLinkByVar(id string, typ RewardType) *LinkByVar {
	ret := &LinkByVar{
		id:  id,
		typ: typ,
	}
	return ret
}

type LinkByVar struct {
	id  string
	typ RewardType
	*linkTimesSimple
}

func (r *LinkByVar) init(rss []IRewardLink) error {
	_s, err := newLinkTimesSimple(r.id, rss)
	if err != nil {
		return err
	}
	r.linkTimesSimple = _s
	return nil
}

func (r *LinkByVar) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	res := &result{}
	var n uint64
	switch r.typ {
	case RewardTypeByLevel:
		n = uint64(info.GetLevel())
	case RewardTypeByVip:
		n = uint64(info.GetVip())
	case RewardTypeByDate:
		n = uint64(info.GetTimeStamp())
	case RewardTypeServerOpenDays:
		n = uint64(info.GetServerOpenDaysOrWarZoneLongest())
	default:
		return nil, fmt.Errorf("%w id %s %s not define", ErrInvalidConfig, r.id, r.typ)
	}
	res.LinkRandomID = r.execL2S(n)
	return res, nil
}
