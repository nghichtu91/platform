package randpool2

import (
	"strings"

	"github.com/nghichtu91/platform/share/planx/savedbwrapper"

	"github.com/nghichtu91/platform/share/planx/timeutil"
)

type ICounterData interface {
	GetExpireTime() int64
	SetExpireTime(t int64)
	GetCount() uint64 // 抽了多少次
	SetCount(n uint64)
	// arrays counter
	GetTotalWeight() int64                    // 用于取随机数
	SetTotalWeight(tw int64)                  // 用于取随机数
	GetCopies() savedbwrapper.IUint32List     // 原始份数
	GetWeights() savedbwrapper.IInt64List     // 原始权重
	GetWWeights() savedbwrapper.IInt64List    // 加权后的权重
	GetWeightIndex() savedbwrapper.IInt32List // 掉落组的序号
	// seed counter
	GetSumCount() uint32     // 抽奖总次数
	SetSumCount(sc uint32)   // 抽奖总次数
	GetPrizeCount() uint32   // 奖品次数，为0则没有奖品了
	SetPrizeCount(pc uint32) // 奖品次数，为0则没有奖品了
	GetPrizeIndex() uint32   // 第几次是奖品
	SetPrizeIndex(pi uint32) // 第几次是奖品
	GetCurIndex() uint32     // 当前抽到第几次了, 等于sumCount时重置
	SetCurIndex(ci uint32)   // 当前抽到第几次了, 等于sumCount时重置
}

type ICounterMgr interface {
	NewCounter(id string) ICounterData
	GetCounter(id string) (ICounterData, bool)
	DelCounter(id string)
}

type counterExpireParam struct {
	typ   ExpireType
	param uint64
}

func (cep *counterExpireParam) getExpireTime(tNow int64) int64 {
	switch cep.typ {
	case ExpireTypeUntil:
		return int64(cep.param)
	case ExpireTypeActivate:
		return tNow + int64(cep.param)
	case ExpireTypeDaily:
		tNow := timeutil.Now().Unix()
		t := timeutil.DailyBeginUnix(tNow) + Getter.GetDailyResetTime()
		if tNow >= t {
			return t + timeutil.DaySec
		}
		return t
	case ExpireTypeDateAssign:
		tNow := timeutil.Now().Unix()
		t := timeutil.DailyBeginUnix(tNow) + Getter.GetDailyResetTime()
		if tNow < t {
			t -= timeutil.DaySec
		}
		return t + int64(cep.param)*timeutil.DaySec
	}
	return 0
}

func (cep *counterExpireParam) refresh(id string, mgr ICounterMgr, tNow int64) ICounterData {
	// 实现可继承抽卡逻辑
	if cep.typ == ExpireTypeNoActivityId {
		id = strings.Split(id, ":")[0]
	}

	c, ok := mgr.GetCounter(id)
	if ok {
		// 是否过期
		// 当永久时 ExpireTime 记为 0
		if c.GetExpireTime() == 0 {
			return c
		} else {
			// 当前时间超过 ExpireTime 证明过期
			if c.GetExpireTime() <= tNow {
				mgr.DelCounter(id)
			} else {
				return c
			}
		}
	}
	c = mgr.NewCounter(id)
	c.SetExpireTime(cep.getExpireTime(tNow))
	return c
}

type recursionCount struct {
	n int
}

func (p *recursionCount) AddRecursionCount(n int) error {
	if p.n+1 > MaxRecursionNum {
		return ErrMaxRecursionExceed
	}
	p.n += n
	return nil
}

func (p *recursionCount) GetRecursionCount() int {
	return p.n
}
