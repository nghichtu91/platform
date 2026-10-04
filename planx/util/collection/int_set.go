package collection

var empty struct{}

// TODO 后续其他类型的set

// TODO 如果有泛型?
//		type Set[T comparable] map[T]struct{}

type IntSet map[int]struct{}

// NewIntSet 指定初始值的创建.
func NewIntSet(values ...int) IntSet {
	var set = NewIntSetSize(len(values))
	set.Adds(values...)
	return set
}

// NewIntSetSize 指定cap的创建
func NewIntSetSize(size int) IntSet {
	return make(IntSet, size)
}

// AddNoCheck 直接添加
func (set IntSet) AddNoCheck(value int) {
	set[value] = empty
}

// Add 添加一个元素, 返回是否之前不存在于集合中
// 性能会稍微差一点.
func (set IntSet) Add(value int) bool {
	if set.Contain(value) {
		return false
	}
	set.AddNoCheck(value)
	return true
}

// Adds 批量添加元素, 返回新增的元素的个数
func (set IntSet) Adds(values ...int) {
	if len(values) == 0 {
		return
	}
	for _, value := range values {
		set.AddNoCheck(value)
	}
	return
}

// Contain 返回是否包含某个元素
func (set IntSet) Contain(value int) bool {
	_, ok := set[value]
	return ok
}

// RemoveNoCheck 直接删除
func (set IntSet) RemoveNoCheck(value int) {
	delete(set, value)
}

// Remove 删除一个元素, 返回之前是否包含
func (set IntSet) Remove(value int) bool {
	_, ok := set[value]
	if ok {
		set.RemoveNoCheck(value)
	}
	return ok
}

// Removes 批量删除
func (set IntSet) Removes(values ...int) {
	if len(values) == 0 {
		return
	}
	for _, value := range values {
		set.RemoveNoCheck(value)
	}
	return
}

// Clear 重置所有的元素, 注意: 这里不会释放空间! 仅仅是重置所有的映射关系
func (set IntSet) Clear() {
	if set.Len() == 0 {
		return
	}
	// 这里不封装, 是为了利用go编译器优化
	for key := range set {
		delete(set, key)
	}
}

// Len 集合的长度
func (set IntSet) Len() int {
	return len(set)
}

// Intersection 交集. set, other都有. 所以迭代的时候以小的集合为准
func (set IntSet) Intersection(other IntSet) IntSet {
	small, big := judgementIntSetLen(set, other)
	var newSet = NewIntSetSize(small.Len())
	small.Range(func(key int) bool {
		if big.Contain(key) {
			newSet.AddNoCheck(key)
		}
		return true
	})
	return newSet
}

// Union 并集. set, other 加一块
func (set IntSet) Union(other IntSet) IntSet {
	var newSet = NewIntSetSize(set.Len() + other.Len())
	add := func(set IntSet) {
		set.Range(func(value int) bool {
			newSet.AddNoCheck(value)
			return true
		})
	}
	add(set)
	add(other)
	return newSet
}

// Difference 差集. set有 other没有
func (set IntSet) Difference(other IntSet) IntSet {
	var newSet = NewIntSetSize(set.Len())
	set.Range(func(key int) bool {
		if !other.Contain(key) {
			newSet.AddNoCheck(key)
		}
		return true
	})
	return newSet
}

func (set IntSet) Slice() []int {
	ret := make([]int, 0, set.Len())
	set.Range(func(key int) bool {
		ret = append(ret, key)
		return true
	})
	return ret
}

func (set IntSet) Range(f func(key int) bool) {
	if set.Len() == 0 || f == nil {
		return
	}
	for key := range set {
		if !f(key) {
			break
		}
	}
}

// judgementIntSetLen small 是二者中长度较短的集合, big 是二者中长度较长的集合
func judgementIntSetLen(set1, set2 IntSet) (small IntSet, big IntSet) {
	small, big = set1, set2
	if small.Len() > big.Len() {
		small, big = big, small
	}
	return
}
