package rand_pool

import (
	"errors"
	"sort"
)

var (
	ErrTotalWeightZero     = errors.New("total weight is zero")  // 权重和为0
	ErrTotalWeightOverflow = errors.New("total weight overflow") // 权重和溢出
)

// IArrayRander 基于数组的随机生成器
type IArrayRander interface {
	// Init 使用一个uint32 slice进行初始化
	// slice的值为对应权重
	// 譬如 [0, 0, 50, 150]
	Init([]uint32) error

	// Rand 按权重进行一次随机，返回对应的权重的索引
	// 如果Rander未初始化，或初始化数据错误，返回-1
	// 譬如 [0, 0, 50, 150] 只会返回2或3
	Rand() int
}

// RealArrayRander 真随机器
// 每次随机按权重进行，没有保底
type RealArrayRander struct {
	total int

	// 记录对应的随机节点
	// 譬如初始化为 [1, 0, 2, 3, 4]
	// 这里对应的节点为 [1, 3, 6, 10]
	sections []int

	// 记录对应的索引
	// 譬如初始化为 [1, 0, 2, 3, 4]
	// 这里对应的节点为 [0, 2, 3, 4]
	indexes []int
}

func (rd *RealArrayRander) Init(weights []uint32) error {
	rd.total = 0
	rd.sections = make([]int, 0, len(weights))
	rd.indexes = make([]int, 0, len(weights))

	for i := 0; i < len(weights); i++ {
		if weights[i] != 0 {
			pre := rd.total
			rd.total += int(weights[i])
			if rd.total < pre {
				return ErrTotalWeightOverflow
			}
			rd.sections = append(rd.sections, rd.total)
			rd.indexes = append(rd.indexes, i)
		}
	}

	if rd.total == 0 {
		return ErrTotalWeightZero
	}

	return nil
}

func (rd *RealArrayRander) Rand() int {
	if rd.total == 0 {
		return -1
	}

	// 产生 [0, n) 区间随机数
	n := Intn(rd.total)

	// 使用go自己的二分查找
	idx := sort.Search(len(rd.indexes), func(i int) bool {
		return rd.sections[i] > n
	})

	// 返回原始数据中的索引
	return rd.indexes[idx]
}
