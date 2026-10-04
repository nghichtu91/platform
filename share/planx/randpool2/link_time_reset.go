package randpool2

import (
	"fmt"
)

func newLinkTimesReset(id string, datas []IRewardLink) *linkTimesReset {
	ret := &linkTimesReset{
		id:    id,
		datas: datas,
	}
	return ret
}

type linkTimesReset struct {
	id    string
	datas []IRewardLink
}

func (r *linkTimesReset) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	res := &result{}
	for _, data := range r.datas {
		id := data.GetGoto()
		if data.GetSaveType() != ExpireTypeStrNoActivityId {
			if activityId != "" {
				id = fmt.Sprintf("%s:%s", id, activityId)
			}
		}
		c, ok := counters.GetCounter(id)
		if !ok {
			c = counters.NewCounter(id)
		}
		c.SetCount(data.GetParam())
	}
	return res, nil
}
