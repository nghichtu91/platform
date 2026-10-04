package randpool

import (
	"errors"

	jsoniter "github.com/json-iterator/go"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

/*
	parse处理策划源数据到模块数据的转换过程
*/

var (
	ErrUnknownRewardType    = errors.New("unknown reward type")
	ErrUnknownRandomType    = errors.New("unknown random type")
	ErrUnknownSaveType      = errors.New("unknown save type")
	ErrInvalidRepeatType    = errors.New("invalid repeat type")
	ErrInvalidLootData      = errors.New("invalid loot data")
	ErrInvalidRangeData     = errors.New("invalid range data")
	ErrInvalidRandomData    = errors.New("invalid random data")
	ErrInvalidArraysData    = errors.New("invalid arrays data")
	ErrInvalidResetData     = errors.New("invalid reset data")
	ErrInvalidRatioSequence = errors.New("invalid ratio sequence") // 概率序列非法
)

// 两种奖励输出方式的映射
var RewardOutputMap = map[RewardType]RewardOutputType{
	RewardTypeRewardStatic: RewardOutputStatic,
	RewardTypeRewardWeight: RewardOutputWeight,
}

var RewardRepeatMap = map[RewardType]RepeatType{
	RewardTypeRedo:     RepeatTypeRepeat,
	RewardTypeMultiple: RepeatTypeMultiple,
}

// RewardType 策划表定义的逻辑类型
type RewardType string

const (
	RewardTypeRewardStatic RewardType = "RewardStatic"
	RewardTypeRewardWeight            = "RewardWeight"
	RewardTypeRedo                    = "Redo"
	RewardTypeMultiple                = "Multiple"
	RewardTypeArrays                  = "Arrays"
	RewardTypeRandom                  = "Random"
	RewardTypeTimes                   = "Times"
	RewardTypeByLevel                 = "ByLevel"
	RewardTypeByVip                   = "ByVip"
	RewardTypeByDate                  = "ByDate"
	RewardTypeSeed                    = "Seed"
	RewardTypeSeedTimes               = "SeedTimes"
)

// RandomTypeMap 策划表"RewardType"字段对应的类型定义
var RandomTypeMap = map[RewardType]RandomType{
	RewardTypeRewardStatic: RandomTypeReward,
	RewardTypeRewardWeight: RandomTypeReward,
	RewardTypeRedo:         RandomTypeRepeat,
	RewardTypeMultiple:     RandomTypeRepeat,
	RewardTypeArrays:       RandomTypeShuffle,
	RewardTypeRandom:       RandomTypeRandom,
	RewardTypeTimes:        RandomTypeRange,
	RewardTypeByVip:        RandomTypeRange,
	RewardTypeByDate:       RandomTypeRange,
	RewardTypeByLevel:      RandomTypeRange,
	RewardTypeSeed:         RandomTypeSeed,
	RewardTypeSeedTimes:    RandomTypeSeedTimes,
}

// RangeTypeMap
var RangeTypeMap = map[RewardType]RangeVariantType{
	RewardTypeTimes:   Times,
	RewardTypeByVip:   VIP,
	RewardTypeByLevel: Level,
	RewardTypeByDate:  Date,
}

// ExpireTypeStr 策划表定义的过期类型
type ExpireTypeStr string

const (
	ExpireTypeStrNever      ExpireTypeStr = "static"
	ExpireTypeStrUtil                     = "until"
	ExpireTypeStrActivate                 = "activate"
	ExpireTypeStrDaily                    = "Daily"
	ExpireTypeStrDateAssign               = "DateAssign"
)

// ExpireTypeMap 策划表"SaveType"字段对应的类型定义
var ExpireTypeMap = map[ExpireTypeStr]ExpireType{
	ExpireTypeStrNever:      ExpireTypeNever,
	ExpireTypeStrUtil:       ExpireTypeUntil,
	ExpireTypeStrActivate:   ExpireTypeActivate,
	ExpireTypeStrDaily:      ExpireTypeDaily,
	ExpireTypeStrDateAssign: ExpireTypeDateAssign,
}

type IRewardParam interface {
	GetSubID() uint32
}

type OutputRewardParam struct {
	SubID        uint32 // 副键
	OutputID     uint32 // 输出ID
	Weight       uint32 // 权重
	RangeDown    uint32 // 数量下限
	RangeUp      uint32 // 数量上限
	Distribution uint32 // 分布类
}

func (p *OutputRewardParam) GetSubID() uint32 {
	return p.SubID
}

type LinkRewardParam struct {
	SubID       uint32   // 副键
	Param       int64    // 条件参数
	Goto        string   // 跳转帧
	GotoPage    string   // 跳转页
	Factor      uint32   // 权数
	PoolRange   []uint32 // 总池区间
	WeightRange []uint32 // 权数区间
	HitLink     string   // 抽中跳转
	MissLink    string   // 未中跳转, 可能为空，则什么也不做
}

func (p *LinkRewardParam) GetSubID() uint32 {
	return p.SubID
}

// RandomSource 提供配置的接口，目前的pb文件可以直接使用接口
type RandomSource interface {
	GetID() string
	GetSaveType() string
	GetSaveParam() int64
	GetRewardType() string
	GetParam() []IRewardParam
}

// marshalBaseInfo 解析基础信息
func marshalBaseInfo(s RandomSource) (*BaseInfo, error) {
	rType, ok := RandomTypeMap[RewardType(s.GetRewardType())]
	if !ok {
		return nil, ErrUnknownRewardType
	}

	eType, ok := ExpireTypeMap[ExpireTypeStr(s.GetSaveType())]
	if !ok {
		return nil, ErrUnknownSaveType
	}

	return &BaseInfo{
		ID:               s.GetID(),
		RandomType:       rType,
		RangeVariantType: RangeTypeMap[RewardType(s.GetRewardType())], // 可选，默认0
		ExpireType:       eType,
		ExpiredTimeValue: s.GetSaveParam(),
	}, nil
}

// IRandomMiddleWare 原始json -> IRandomMiddleWare -> RandomData过程的中间件
type IRandomMiddleWare interface {
	// ToData 生成不带BaseInfo的RandomData结构
	ToData(RandomSource) (RandomData, error)
}

// MarshalSource 将json字符串按随机类型解析
func MarshalSource(s RandomSource) (RandomData, error) {
	baseInfo, err := marshalBaseInfo(s)
	if err != nil {
		return nil, err
	}

	var mw IRandomMiddleWare

	switch baseInfo.RandomType {
	case RandomTypeReward: // 奖励
		mw = new(RewardMiddleWare)
	case RandomTypeRepeat: // 放大组
		mw = new(RepeatMiddleWare)
	case RandomTypeShuffle: // 乱序组
		mw = new(ShuffleMiddleWare)
	case RandomTypeRandom: // 真随机
		mw = new(RandomMiddleWare)
	case RandomTypeRange: // 区间组
		mw = new(RangeMiddleWare)
	case RandomTypeSeed:
		mw = new(SeedMiddleWare)
	case RandomTypeSeedTimes:
		mw = new(SeedTimesMiddleWare)
	default: // 这里只有新加了类型，两边不匹配才会走到这里，覆盖不到了
		return nil, ErrUnknownRandomType
	}

	data, err := mw.ToData(s)
	if err != nil {
		return nil, err
	}

	data.SetBaseInfo(baseInfo)

	return data, nil
}
