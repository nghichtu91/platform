// Package lru 通用的lru缓存以及对应cache
package lru

// StrCache 指定key为string的cache
// 无锁，用于modules内部，
// 并发不安全
type StrCache struct {
	// 缓存顶和底
	head, tail *StrNode

	// 缓存最大容量
	cap int

	// 存储内容
	data map[string]*StrNode
}

// StrNode 使用int64作为key的节点，对应StrCache
type StrNode struct {
	// 用于cache map中查询，需要唯一
	key string

	// 链表指针
	pre, next *StrNode

	// 缓存内容
	v interface{}
}

// NewStrCache 生成容量为cap的cache
func NewStrCache(cap int) *StrCache {
	return &StrCache{
		cap:  cap,
		data: make(map[string]*StrNode, cap),
	}
}

// push 放入节点
func (c *StrCache) push(node *StrNode) {
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
func (c *StrCache) pop(node *StrNode) {
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
func (c *StrCache) Get(k interface{}) interface{} {
	key, ok := k.(string)
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
func (c *StrCache) Put(k, v interface{}) {
	key, ok := k.(string)
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

	node := &StrNode{key: key, v: v}
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
func (c *StrCache) Prune() {
	newData := make(map[string]*StrNode, len(c.data))

	for k := range c.data {
		newData[k] = c.data[k]
	}

	c.data = newData
}
