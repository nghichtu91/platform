package randpool

import (
	"math/rand"
	"time"
)

/*
	param 用于传入各类不包含于counter内的参数
*/

// IRandomParams 随机参数，包括需要从外部获取的参数，时间戳，随机种子，递归次数等等
type IRandomParams interface {
	GetValue(vType RangeVariantType) int
	GetTimestamp() int64
	GetRand() *rand.Rand
	AddRecursionCount(n int) error
	GetRecursionCount() int
	GetDailyRefreshTime() string // 5:00
}

type DefaultRandomParam struct {
	SavedParams map[RangeVariantType]int

	Rand             *rand.Rand
	RecursionCount   int // 递归计数
	DailyRefreshTime string
}

func DefaultParam(r *rand.Rand) IRandomParams {
	return &DefaultRandomParam{
		SavedParams: make(map[RangeVariantType]int),
		Rand:        r,
	}
}

func (p *DefaultRandomParam) GetValue(vType RangeVariantType) int {
	return p.SavedParams[vType]
}

func (p *DefaultRandomParam) GetTimestamp() int64 {
	return time.Now().Unix()
}

func (p *DefaultRandomParam) GetRand() *rand.Rand {
	return p.Rand
}

func (p *DefaultRandomParam) AddRecursionCount(n int) error {
	if p.RecursionCount+1 > MaxRecursionNum {
		return ErrMaxRecursionExceed
	}
	p.RecursionCount += n
	return nil
}

func (p *DefaultRandomParam) GetRecursionCount() int {
	return p.RecursionCount
}

func (p *DefaultRandomParam) GetDailyRefreshTime() string {
	return p.DailyRefreshTime
}
