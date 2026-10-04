package randpool2

import (
	"fmt"
	"strings"
	"time"
)

type IRewarder interface {
	run(counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) (*result, error)
}

type RewardDataGetter interface {
	GetRandomRewardData(id string) IRewarder
	GetRandomRewardDistribution(id uint32) IDistribution
	GetDailyResetTime() int64
}

var (
	Getter RewardDataGetter
)

// 加载配置数据，服务器启动和热更的时候调用
func Load(rs []IRewardStaticOutput,
	rw []IRewardWeightOutput,
	rl []IRewardLink,
	ds []IDistribution) (map[string]IRewarder, map[uint32]IDistribution, error) {

	if rs == nil || rw == nil || rl == nil || ds == nil {
		return nil, nil, fmt.Errorf("%w, reward配置中有表为空 ", ErrInvalidConfig)
	}
	if err := checkIdDuplicate(rs, rw, rl); err != nil {
		return nil, nil, err
	}
	// Distribution
	_distributions := make(map[uint32]IDistribution, len(ds))
	for i := range ds {
		_ds := ds[i]
		_distributions[_ds.GetID()] = _ds
	}

	_datas := make(map[string]IRewarder, 1024)
	// RewardStaticOutput
	tmp_rs := make(map[string][]IRewardStaticOutput, 1024)
	for i := range rs {
		_rs := rs[i]
		if _rs.GetDistribution() > 0 {
			if _, ok := _distributions[_rs.GetDistribution()]; !ok {
				return nil, nil, fmt.Errorf("%w RewardStaticOutput表里RandomID为%s，SubID为%d的Distribution在RandomDistribution表中没找到, Distribution为:%d",
					ErrInvalidConfig, _rs.GetRandomID(), _rs.GetSubID(), _rs.GetDistribution())
			}
		}
		_rss, ok := tmp_rs[_rs.GetRandomID()]
		if !ok {
			_rss = make([]IRewardStaticOutput, 0, DefaultTmpMapLength)
		}
		_rss = append(_rss, _rs)
		tmp_rs[_rs.GetRandomID()] = _rss
	}
	for id, rss := range tmp_rs {
		_datas[id] = NewOutputStatic(id, rss)
	}

	// RewardWeightOutput
	tmp_rw := make(map[string][]IRewardWeightOutput, 1024)
	for i := range rw {
		_rw := rw[i]
		if _rw.GetDistribution() > 0 {
			if _, ok := _distributions[_rw.GetDistribution()]; !ok {
				return nil, nil, fmt.Errorf("%w RewardWeightOutput表里RandomID为%s，SubID为%d的Distribution在RandomDistribution表中没找到, Distribution:%d",
					ErrInvalidConfig, _rw.GetRandomID(), _rw.GetSubID(), _rw.GetDistribution())
			}
		}
		_rsw, ok := tmp_rw[_rw.GetRandomID()]
		if !ok {
			_rsw = make([]IRewardWeightOutput, 0, DefaultTmpMapLength)
		}
		_rsw = append(_rsw, _rw)
		tmp_rw[_rw.GetRandomID()] = _rsw
	}
	for id, rws := range tmp_rw {
		_datas[id] = NewOutputWeight(id, rws)
	}

	// RewardLink
	tmp_rl := make(map[string]*tmpLink, 1024)
	for i := range rl {
		_rl := rl[i]
		_rls, ok := tmp_rl[_rl.GetRandomID()]
		if !ok {
			_rls = &tmpLink{
				data: make([]IRewardLink, 0, DefaultTmpMapLength),
			}
		}
		_rls.data = append(_rls.data, _rl)
		if _rl.GetSubID() == 1 { // 和策划约定subId为1里，填RewardType
			_typ := RewardType(strings.TrimSpace(strings.ToLower(_rl.GetRewardType())))
			_rls.typ = _typ
			if _rl.GetSaveType() != "" {
				typ, ok := ExpireTypeMap[ExpireTypeStr(_rl.GetSaveType())]
				if !ok {
					return nil, nil, fmt.Errorf("%w ，RewardLink表中RandomId为%s，SubID为%d的配置对应的SaveType是非法的，错误的SaveType为：%s",
						ErrInvalidConfig, _rl.GetRandomID(), _rl.GetSubID(), _rl.GetSaveType())
				}
				_p := _rl.GetSaveParam()
				if typ == ExpireTypeUntil {
					t, err := time.Parse(TimeStampLayOut, fmt.Sprintf("%d", _rl.GetSaveParam()))
					if err != nil {
						return nil, nil, fmt.Errorf("%w RewardLink表中RandomId为%s，SubID为%d的配置对应的的SaveParam解析出现错误，错误的SaveParam：%d，err %s",
							ErrInvalidConfig, _rl.GetRandomID(), _rl.GetSubID(), _rl.GetSaveParam(), err.Error())
					}
					_p = uint64(t.Unix())
				}
				_rls.cep = &counterExpireParam{
					typ:   typ,
					param: _p,
				}
			}
		}
		tmp_rl[_rl.GetRandomID()] = _rls
	}
	// 检查goto
	noDefGoTo := map[string]struct{}{}
	for _, v := range tmp_rl {
		for _, vv := range v.data {
			k := vv.GetGoto()
			if k == "" || k == TimesReset {
				continue
			}
			if _, ok := tmp_rl[k]; ok {
				continue
			}
			if _, ok := tmp_rw[k]; ok {
				continue
			}
			if _, ok := tmp_rs[k]; ok {
				continue
			}
			noDefGoTo[k] = struct{}{}
		}
	}
	if len(noDefGoTo) > 0 {
		sb := strings.Builder{}
		for k := range noDefGoTo {
			sb.WriteString(k + " ")
		}
		return nil, nil, fmt.Errorf("%w ，RewardLink表中有以下GoTo没有定义：%s", ErrInvalidConfig, sb.String())
	}
	for k, v := range tmp_rl {
		if v.typ == "" {
			return nil, nil, fmt.Errorf("%w，RewardLink表中RandomId为%s的配置没有RewardType", ErrInvalidConfig, k)
		}
		switch v.typ {
		case RewardTypeRedo:
			_lr := newLinkRedo(k, v.data)
			_datas[k] = _lr
		case RewardTypeMultiple:
			_lm := newLinkMultiple(k, v.data)
			_datas[k] = _lm
		case RewardTypeRandom:
			_l := newLinkRandom(k, v.data)
			_datas[k] = _l
		case RewardTypeRandomCondition:
			_l := newLinkRandomCondition(k, v.data)
			_datas[k] = _l
		case RewardTypeArrays:
			_l := newLinkArrays(k, v.data, v.cep)
			if err := _l.init(); err != nil {
				return nil, nil, err
			}
			_datas[k] = _l
		case RewardTypeTimes:
			_lt := newLinkTimes(k, v.cep)
			if err := _lt.init(v.data); err != nil {
				return nil, nil, err
			}
			_datas[k] = _lt
		case RewardTypeTimesReset:
			for _, data := range v.data {
				gotoData, ok := tmp_rl[data.GetGoto()]
				if !ok {
					return nil, nil, fmt.Errorf("%w，RewardType为%v,RewardLink表中RandomId为%s的配置对应的goto %s 没有找到", ErrInvalidConfig, RewardTypeTimesReset, k, data.GetGoto())
				}
				if gotoData.typ != RewardTypeTimes {
					return nil, nil, fmt.Errorf("%w，RewardType为%v,RewardLink表中RandomId为%s的配置对应的goto%s，对应的类型不是times", ErrInvalidConfig, RewardTypeTimesReset, k, data.GetGoto())
				}
			}
			_ltr := newLinkTimesReset(k, v.data)
			_datas[k] = _ltr
		case RewardTypeByLevel, RewardTypeByVip, RewardTypeByDate, RewardTypeServerOpenDays:
			_lbv := NewLinkByVar(k, v.typ)
			if err := _lbv.init(v.data); err != nil {
				return nil, nil, err
			}
			_datas[k] = _lbv
		case RewardTypeSeed:
			if len(v.data) != 1 {
				return nil, nil, fmt.Errorf("%w，RewardType为%v,RewardLink表中RandomId为%s的配置，对应的配置数不等于1", ErrInvalidConfig, RewardTypeSeed, k)
			}
			_range := v.data[0].GetPoolRange()
			if len(_range) != 2 {
				return nil, nil, fmt.Errorf("%w，RewardType为%v,RewardLink表中RandomId为%s的配置对应的PoolRange长度不等于2", ErrInvalidConfig, RewardTypeSeed, k)
			}
			if _range[0] == 0 && _range[1] == 0 {
				continue
			}
			_l := newLinkSeed(k, v.data[0], v.cep)
			if err := _l.init(); err != nil {
				return nil, nil, err
			}
			_datas[k] = _l
		case RewardTypeSeedTimes:
			if len(v.data) != 1 {
				return nil, nil, fmt.Errorf("%w RewardType为%v,RewardLink表中RandomId为%s的配置对应的配置数不唯一", ErrInvalidConfig, RewardTypeSeedTimes, k)
			}
			_l := newLinkSeedTimes(k, v.data[0], v.cep)
			if err := _l.init(); err != nil {
				return nil, nil, err
			}
			_datas[k] = _l
		default:
			return nil, nil, fmt.Errorf("%w ，RewardLink表中RandomId为%s的配置对应的配置对应的RewardType是错误的 %v", ErrInvalidConfig, k, v.typ)
		}
	}

	return _datas, _distributions, nil
}

// 抽奖运行的时候调用
// res有可能为nil
func Run(id string, counters ICounterMgr, info IProfileInfo, activityId string) ([]*Loot, error) {
	return _Run(id, counters, info, &recursionCount{}, activityId)
}

func _Run(id string, counters ICounterMgr, info IProfileInfo, recursionC *recursionCount, activityId string) ([]*Loot, error) {
	if id == "" || counters == nil || info == nil {
		return nil, fmt.Errorf("%w param is nil", ErrInvalidParams)
	}

	if err := recursionC.AddRecursionCount(1); err != nil {
		return nil, fmt.Errorf("%w id %s Recursion %d", err, id, recursionC.GetRecursionCount())
	}

	_r := Getter.GetRandomRewardData(id)
	if _r == nil {
		return nil, fmt.Errorf("%w id %s", ErrIDNotExist, id)
	}
	res, err := _r.run(counters, info, recursionC, activityId)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("%w id %s", ErrNoResult, id)
	}
	if res.LinkRandomID != "" {
		return _Run(res.LinkRandomID, counters, info, recursionC, activityId)
	}
	return res.Loots, nil
}

func ClearExpireCounter(id string, mgr ICounterMgr, tNow int64) {
	c, ok := mgr.GetCounter(id)
	if ok {
		// 是否过期
		if c.GetExpireTime() != 0 && c.GetExpireTime() <= tNow {
			mgr.DelCounter(id)
		}
	}
}

type tmpLink struct {
	typ  RewardType
	cep  *counterExpireParam
	data []IRewardLink
}

func checkIdDuplicate(rs []IRewardStaticOutput,
	rw []IRewardWeightOutput,
	rl []IRewardLink) error {
	_ids := make(map[string]struct{}, 1024)
	for _, _r := range rs {
		id := fmt.Sprintf("%s_%d", _r.GetRandomID(), _r.GetSubID())
		_, ok := _ids[id]
		if ok {
			return fmt.Errorf("%w id 重复配置 %s %d", ErrInvalidConfig, _r.GetRandomID(), _r.GetSubID())
		}
		_ids[id] = struct{}{}
	}
	for _, _r := range rw {
		id := fmt.Sprintf("%s_%d", _r.GetRandomID(), _r.GetSubID())
		_, ok := _ids[id]
		if ok {
			return fmt.Errorf("%w 重复配置 %s %d", ErrInvalidConfig, _r.GetRandomID(), _r.GetSubID())
		}
		_ids[id] = struct{}{}
	}
	for _, _r := range rl {
		id := fmt.Sprintf("%s_%d", _r.GetRandomID(), _r.GetSubID())
		_, ok := _ids[id]
		if ok {
			return fmt.Errorf("%w 重复配置 %s %d", ErrInvalidConfig, _r.GetRandomID(), _r.GetSubID())
		}
		_ids[id] = struct{}{}
	}
	return nil
}
