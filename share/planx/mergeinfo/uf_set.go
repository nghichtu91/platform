package mergeinfo

import (
	"sort"
)

// IntUFSet 一个简化的整数并查集. 需求上不需要进行Merge, 所以这里只有简单的检查
// node -> root: 描述的是子节点和根节点之间的关系
type IntUFSet map[int]int

func NewIntUFSet(size int) IntUFSet {
	return make(IntUFSet, size)
}

// SetRoot 设置node的根节点为root
func (set IntUFSet) SetRoot(node, root int) {
	set[node] = root
}

// GetRoot 获取指定node的根, 如果node是新增节点, 那么isNew返回true
func (set IntUFSet) GetRoot(node int) (root int, isNew bool) {
	// 假设原始的 数据是 [1,2,3,4,5]
	// 旧的node -> root是 [1:1, 2:1, 3:2, 4:3, 5:4]
	// 通过 GetRoot(5)
	// 新的node -> root是 [1:1, 2:1, 3:1, 4:1, 5:1]
	// 根节点1次hash, 压平后的并查集查询子节点需要两次hash
	var find bool
	root, find = set[node]
	if !find {
		// 新节点, 根节点就是自己
		set.SetRoot(node, node)
		isNew = true
		root = node
		return
	}
	if root == node {
		// 根节点是自己, 直接返回
		return
	}
	// 根节点不是自己, 此时需要递归的获取根节点
	// 这种情况下, node本身就不可能是新节点了
	root, _ = set.GetRoot(root)
	// 压平并查集, 下次可以快速取出root
	set.SetRoot(node, root)
	return
}

// GetRoots 返回所有所有根节点
func (set IntUFSet) GetRoots() map[int]struct{} {
	var allRoot = make(map[int]struct{}, len(set)/4)
	for node := range set {
		// 这里不会出现new节点
		root, _ := set.GetRoot(node)
		// 判定也需要一次hash, 不如直接赋值
		allRoot[root] = struct{}{}
	}
	return allRoot
}

// GetRootSlice 获取所有的根节点, 按照大小排序
func (set IntUFSet) GetRootSlice() []int {
	rootSet := set.GetRoots()
	ret := make([]int, 0, len(rootSet))
	for root := range rootSet {
		ret = append(ret, root)
	}
	sort.Ints(ret)
	return ret
}

/*

整体检查思路如下:

添加根节点时,
	- 如果是新节点, 更新所有子节点的根节点为自身.
		- 如果某一子节点不是新节点, 需要保证它以及它之前的根节点 都在当前集合中出现
	- 如果不是新节点, 需要保证所有子节点的根节点是相同的, 并且所有的子节点也不应该是 新的

*/
