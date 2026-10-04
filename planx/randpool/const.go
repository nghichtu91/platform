package randpool

import (
	"errors"
)

const (
	MaxRecursionNum      = 64       // 最大递归数
	MaxFloatRange        = 1e8      // 用于模拟8位随机浮点数
	DefaultTmpMapLength  = 8        // 建立的map默认长度，一般数据长度不会超过这个长度
	ExpireTimeNever      = -1       // 永不过期的时间值
	GetCountNotAvailable = -1       // GetCount无效值
	TimesReset           = "#reset" // Times重置关键字
)

var (
	ErrInvalidID          = errors.New("invalid id")
	ErrInvalidConfig      = errors.New("invalid config")
	ErrInvalidProfile     = errors.New("invalid profile")
	ErrInvalidParams      = errors.New("invalid params")
	ErrMaxRecursionExceed = errors.New("max recursion exceed")
	ErrInvalidRatios      = errors.New("invalid ratios") // 概率分布函数有误
	ErrInvalidStep        = errors.New("invalid step")   // 分布表步长非法
	ErrLootFail           = errors.New("loot fail")      // loot失败
)

// RewardOutputType 奖励类型
type RewardOutputType int

const (
	RewardOutputStatic RewardOutputType = iota
	RewardOutputWeight
)

// RandomType 随机类型
type RandomType int

const (
	RandomTypeReward    RandomType = iota // 随机奖励
	RandomTypeRepeat                      // 放大器
	RandomTypeRandom                      // 真随机
	RandomTypeShuffle                     // 乱序
	RandomTypeRange                       // 区间，可以重置
	RandomTypeSeed                        // Seed
	RandomTypeSeedTimes                   // SeedTimes
	RandomTypeCount
)

// ExpireType 配置过期时间
type ExpireType int

const (
	ExpireTypeNever      ExpireType = iota // 永久
	ExpireTypeUntil                        // 达到时过期
	ExpireTypeActivate                     // 每次激活后延长
	ExpireTypeDaily                        // 每天指定时间(HHMMSS)重置
	ExpireTypeDateAssign                   // 从激活当天的05:00为起点，保留多少天
	ExpireTypeCount
)

// RepeatType 重复类型
type RepeatType int

const (
	RepeatTypeNone     RepeatType = iota // 给予指定物品ID和数量
	RepeatTypeMultiple                   // 指定ID奖励乘n
	RepeatTypeRepeat                     // 指定ID奖励重复n次
	RepeatTypeCount
)

// RangeVariantType 区间掉落传入变量类型
type RangeVariantType int

const (
	None  RangeVariantType = iota
	Times                  // 次数
	Level                  // 等级
	VIP                    // VIP
	Date                   // Date
	RangeVariantTypeCount
)

type DistributionType int

const (
	Liner   DistributionType = iota // 线性
	Density                         // 概率密度分布
)

const (
	HitLink  int = iota // 0
	MissLink            // 1
)
