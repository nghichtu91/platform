package randpool2

import "errors"

// RewardType 策划表定义的逻辑类型
type RewardType string

const (
	RewardTypeRedo            RewardType = "redo"
	RewardTypeMultiple                   = "multiple"
	RewardTypeArrays                     = "arrays"
	RewardTypeRandom                     = "random"
	RewardTypeRandomCondition            = "randomcondition"
	RewardTypeTimes                      = "times"
	RewardTypeTimesReset                 = "timesreset"
	RewardTypeByLevel                    = "bylevel"
	RewardTypeByVip                      = "byvip"
	RewardTypeByDate                     = "bydate"
	RewardTypeSeed                       = "seed"
	RewardTypeSeedTimes                  = "seedtimes"
	RewardTypeServerOpenDays             = "severopenday"
)

// ExpireTypeStr 策划表定义的过期类型
type ExpireTypeStr string

const (
	ExpireTypeStrNever        ExpireTypeStr = "static"
	ExpireTypeStrUtil                       = "until"
	ExpireTypeStrActivate                   = "activate"
	ExpireTypeStrDaily                      = "Daily"
	ExpireTypeStrDateAssign                 = "DateAssign"
	ExpireTypeStrNoActivityId               = "NoActivityId"
)

// ExpireType 配置过期时间
type ExpireType int

const (
	ExpireTypeNever        ExpireType = iota // 永久
	ExpireTypeUntil                          // 达到时过期
	ExpireTypeActivate                       // 每次激活后延长
	ExpireTypeDaily                          // 每天指定时间(05:00)重置
	ExpireTypeDateAssign                     // 从激活当天的(05:00)为起点，保留多少天
	ExpireTypeNoActivityId                   // 不添加ActivityId后缀
	ExpireTypeCount
)

// ExpireTypeMap 策划表"SaveType"字段对应的类型定义
var ExpireTypeMap = map[ExpireTypeStr]ExpireType{
	ExpireTypeStrNever:        ExpireTypeNever,
	ExpireTypeStrUtil:         ExpireTypeUntil,
	ExpireTypeStrActivate:     ExpireTypeActivate,
	ExpireTypeStrDaily:        ExpireTypeDaily,
	ExpireTypeStrDateAssign:   ExpireTypeDateAssign,
	ExpireTypeStrNoActivityId: ExpireTypeNoActivityId,
}

const (
	DefaultTmpMapLength = 8        // 建立的map默认长度，一般数据长度不会超过这个长度
	TimesReset          = "#reset" // Times重置关键字
	MaxRecursionNum     = 64       // 最大递归数
	TimeStampLayOut     = "20060102150405"
)

var (
	ErrInvalidConfig      = errors.New("invalid config")
	ErrIDNotExist         = errors.New("random id not exist")
	ErrInvalidParams      = errors.New("invalid params")
	ErrNoResult           = errors.New("no result")
	ErrMaxRecursionExceed = errors.New("max recursion exceed")
)
