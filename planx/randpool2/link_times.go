package randpool2

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nghichtu91/platform/share/planx/util"
)

func newLinkTimes(id string, cep *counterExpireParam) *linkTimes {
	ret := &linkTimes{
		id: id,
		timesC: &timesC{
			counterExpireParam: cep,
		},
	}
	return ret
}

type linkTimes struct {
	id string
	*linkTimesSimple
	*timesC
}

func (r *linkTimes) init(rss []IRewardLink) error {
	if r.counterExpireParam == nil {
		return fmt.Errorf("%w id %s ExpireParam is nil", ErrInvalidConfig, r.id)
	}
	_s, err := newLinkTimesSimple(r.id, rss)
	if err != nil {
		return err
	}
	r.linkTimesSimple = _s
	// 检查param是否重复
	_ps := make(map[uint64]struct{}, 4)
	for _, _p := range r.sections {
		_, ok := _ps[_p]
		if ok {
			return fmt.Errorf("%w id %s Param repeated", ErrInvalidConfig, r.id)
		}
		_ps[_p] = struct{}{}
	}
	for i, id := range r.groupIDs {
		if id == TimesReset {
			if r.maxSection > 0 {
				return fmt.Errorf("%w id %s has multiple reset", ErrInvalidConfig, r.id)
			}
			r.maxSection = r.sections[i]
			_p, err := strconv.ParseUint(r.groupPages[i], 10, 64)
			if err != nil {
				return fmt.Errorf("%w id %s reset GotoPage not number", ErrInvalidConfig, r.id)
			}
			r.resetValue = _p
		}
	}
	return nil
}

func (r *linkTimes) run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error) {
	res := &result{}
	id := r.id
	if activityId != "" {
		id = fmt.Sprintf("%s:%s", r.id, activityId)
	}

	n := r.getCount(id, counters, info.GetTimeStamp())
	res.LinkRandomID = r.execS2L(n)
	return res, nil
}

func newLinkTimesSimple(id string, rss []IRewardLink) (*linkTimesSimple, error) {
	ret := &linkTimesSimple{
		sections:   make([]uint64, 0, len(rss)),
		groupIDs:   make([]string, 0, len(rss)),
		groupPages: make([]string, 0, len(rss)),
	}
	m := make(map[uint64]tmpGroup, len(rss))
	for _, l := range rss {
		_p := l.GetParam()
		if l.GetRewardType() == RewardTypeByDate {
			t, err := time.Parse(TimeStampLayOut, fmt.Sprintf("%d", l.GetParam()))
			if err != nil {
				return nil, fmt.Errorf("%w RewardLink表中RandomId为%s的配置对应的配置，rewardType为：%s，解析时间%d时发生错误， err %s",
					ErrInvalidConfig, id, RewardTypeByDate, l.GetParam(), err.Error())
			}
			_p = uint64(t.Unix())
		}
		ret.sections = append(ret.sections, _p)
		m[_p] = tmpGroup{
			groupId:   l.GetGoto(),
			groupPage: l.GetGotoPage(),
		}
	}
	util.Uint64s(ret.sections)
	for _, v := range ret.sections {
		_group := m[v]
		ret.groupIDs = append(ret.groupIDs, _group.groupId)
		ret.groupPages = append(ret.groupPages, _group.groupPage)
	}
	return ret, nil
}

type linkTimesSimple struct {
	sections   []uint64 // 区间节点
	groupIDs   []string // 节点对应的跳转ID
	groupPages []string
}

// 从小到大，左开右闭，超过最大取最大
func (lts *linkTimesSimple) execS2L(n uint64) string {
	for i := 0; i < len(lts.sections); i++ {
		if n <= lts.sections[i] {
			return lts.groupIDs[i]
		}
	}
	return lts.groupIDs[len(lts.groupIDs)-1]
}

// 从大到小，左闭右开，小于最小则没有奖励
func (lts *linkTimesSimple) execL2S(n uint64) string {
	for i := len(lts.sections) - 1; i >= 0; i-- {
		if n >= lts.sections[i] {
			return lts.groupIDs[i]
		}
	}
	return ""
}

type tmpGroup struct {
	groupId   string
	groupPage string
}

// Times计数器
type timesC struct {
	*counterExpireParam
	maxSection uint64 // 区间上界，同时也是重置节点
	resetValue uint64 // 重置到的数量，大于0时有效
}

func (c *timesC) getCount(id string, mgr ICounterMgr, tNow int64) uint64 {
	counterData := c.refresh(id, mgr, tNow)
	n := counterData.GetCount() + 1
	if c.maxSection > 0 && n >= c.maxSection {
		n = c.resetValue
	}
	counterData.SetCount(n)
	return n
}
