// Package lru 通用的lru缓存以及对应cache
package lru

// IntCache 指定key为int64的cache，对比通用cache能带来约20%的性能提升
// 无锁，用于modules内部，
// 并发不安全
type IntCache struct {
	// 缓存顶和底
	head, tail *IntNode

	// 缓存最大容量
	cap int

	// 存储内容
	data map[int64]*IntNode
}

// IntNode 使用int64作为key的节点，对应IntCache
type IntNode struct {
	// 用于cache map中查询，需要唯一
	key int64

	// 链表指针
	pre, next *IntNode

	// 缓存内容
	v interface{}
}

// NewIntCache 生成容量为cap的cache
func NewIntCache(cap int) *IntCache {
	return &IntCache{
		cap:  cap,
		data: make(map[int64]*IntNode, cap),
	}
}

// push 放入节点
func (c *IntCache) push(node *IntNode) {
	node.pre = nil
	node.next = c.head
	if c.head != nil {
		c.head.pre = node
	}
	c.head = node
	if c.tail == nil {
		c.tail = node
		c.tail.next = nil
	}
}

// pop 弹出节点
func (c *IntCache) pop(node *IntNode) {
	if c.head == node {
		c.head = node.next
		if node.next != nil {
			node.next.pre = nil
		}
		node.next = nil
		return
	}

	if c.tail == node {
		c.tail = node.pre
		node.pre.next = nil
		node.pre = nil
		return
	}

	node.pre.next = node.next
	node.next.pre = node.pre
}

// Get 查询key对应的缓存
// 同时将命中的缓存放回顶部
func (c *IntCache) Get(k interface{}) interface{} {
	key, ok := k.(int64)
	if !ok {
		return nil
	}
	if node, ok := c.data[key]; ok {
		c.pop(node)
		c.push(node)
		return node.v
	}
	return nil
}

// Put 将cache放入LRU cache
// 同时会执行淘汰
func (c *IntCache) Put(k, v interface{}) {
	key, ok := k.(int64)
	if !ok {
		return
	}

	// 命中放回缓存顶
	if node, ok := c.data[key]; ok {
		node.v = v
		c.pop(node)
		c.push(node)
		return
	}

	node := &IntNode{key: key, v: v}
	c.data[key] = node
	c.push(node)

	// 淘汰
	if len(c.data) > c.cap {
		delete(c.data, c.tail.key)
		c.pop(c.tail)
	}
}

// Prune 整理缓存
// 由于Cache内部使用map
// 只增不减特性可能影响内存，提供这个接口定期整理
func (c *IntCache) Prune() {
	newData := make(map[int64]*IntNode, len(c.data))

	for k := range c.data {
		newData[k] = c.data[k]
	}

	c.data = newData
}
