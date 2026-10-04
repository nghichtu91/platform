package randpool2

func NewOutputStatic(id string, rss []IRewardStaticOutput) *OutputStatic {
	return &OutputStatic{
		id:   id,
		data: rss,
	}
}

type OutputStatic struct {
	id   string
	data []IRewardStaticOutput
}

func (r *OutputStatic) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	res := &result{
		Loots: make([]*Loot, 0, DefaultTmpMapLength),
	}
	for _, _d := range r.data {
		l := genLoot(_d, info)
		if l.Count > 0 {
			res.Loots = append(res.Loots, l)
		}
	}
	return res, nil
}

func genLoot(rs IRewardStaticOutput, info IProfileInfo) *Loot {
	if rs.GetRangeDown() <= 0 && rs.GetRangeUp() <= 0 { // 若上限和下限都为0，则此行跳过
		return &Loot{}
	}
	if rs.GetRangeUp() <= 0 || rs.GetRangeUp() == rs.GetRangeDown() { // 若上限为0，则直接用下限值做为奖励数量
		return &Loot{
			ItemID: rs.GetOutputID(),
			Count:  rs.GetRangeDown(),
		}
	}
	var n uint32
	if rs.GetDistribution() > 0 {
		_dis := Getter.GetRandomRewardDistribution(rs.GetDistribution())
		dd := _dis.GetDistribution()
		n = uint32(float32(rs.GetRangeUp()+1-rs.GetRangeDown())*dd[info.GetRand().Intn(len(dd))]) + rs.GetRangeDown()
	} else {
		n = uint32(info.GetRand().Int31n(int32(rs.GetRangeUp()+1-rs.GetRangeDown()))) + rs.GetRangeDown()
	}
	return &Loot{
		ItemID: rs.GetOutputID(),
		Count:  n,
	}
}
