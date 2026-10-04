package randpool2

import "math/rand"

type IProfileInfo interface {
	GetLevel() uint32
	GetVip() uint32
	GetTimeStamp() int64 // 时间戳
	GetRand() *rand.Rand
	FitCond(rewardCondId uint32) bool
	GetServerOpenDaysOrWarZoneLongest() uint32
}
