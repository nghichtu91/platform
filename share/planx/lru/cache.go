// Package lru 通用的lru缓存以及对应cache
package lru

// Cache 基于Node实现的LRU缓存
// 无锁，用于modules内部，
// 并发不安全
type Cache struct {
	// 缓存顶和底
	head, tail *Node

	// 缓存最大容量
	cap int

	// 存储内容
	data map[interface{}]*Node
}

// Node LRU节点
type Node struct {
	k, v interface{}

	// 链表指针
	pre, next *Node
}

// NewCache 生成容量为cap的cache
func NewCache(cap int) *Cache {
	return &Cache{
		cap:  cap,
		data: make(map[interface{}]*Node, cap),
	}
}

// push 放入节点
func (c *Cache) push(node *Node) {
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
func (c *Cache) pop(node *Node) {
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
func (c *Cache) Get(k interface{}) interface{} {
	if node, ok := c.data[k]; ok {
		c.pop(node)
		c.push(node)
		return node.v
	}
	return nil
}

// Put 将cache放入LRU cache
// 同时会执行淘汰
func (c *Cache) Put(k, v interface{}) {
	// 命中放回缓存顶
	if node, ok := c.data[k]; ok {
		node.v = v
		c.pop(node)
		c.push(node)
		return
	}

	node := &Node{k: k, v: v}
	c.data[k] = node
	c.push(node)

	// 淘汰
	if len(c.data) > c.cap {
		delete(c.data, c.tail.k)
		c.pop(c.tail)
	}
}

// Prune 整理缓存
// 由于Cache内部使用map
// 只增不减特性可能影响内存，提供这个接口定期整理
func (c *Cache) Prune() {
	newData := make(map[interface{}]*Node, len(c.data))

	for k, v := range c.data {
		newData[k] = v
	}

	c.data = newData
}
