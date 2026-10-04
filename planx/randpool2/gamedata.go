package randpool2

type IRewardConfig interface {
	GetRandomID() string
	GetSubID() uint32
}

type IRewardStaticOutput interface {
	IRewardConfig
	GetOutputID() uint32     // 输出ID
	GetRangeDown() uint32    // 数量下限
	GetRangeUp() uint32      // 数量上限
	GetDistribution() uint32 // 分布类
}

type IRewardWeightOutput interface {
	IRewardStaticOutput
	GetWeight() uint32 // 权重
}

type IRewardLink interface {
	IRewardConfig
	GetSaveType() string
	GetSaveParam() uint64
	GetRewardType() string
	GetParam() uint64
	GetGoto() string
	GetGotoPage() string
	GetFactor() uint32
	GetPoolRange() []uint32
	GetWeightRange() []uint32
	GetHitLink() string
	GetMissLink() string
	GetConditionID() uint32
}

type IDistribution interface {
	GetID() uint32
	GetDistribution() []float32
}
