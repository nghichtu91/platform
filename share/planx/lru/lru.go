// Package lru 通用的lru缓存以及对应cache
package lru

type ICache interface {
	// Get 查询key对应的缓存
	// 同时将命中的缓存放回顶部
	Get(k interface{}) interface{}

	// Put 将cache放入LRU cache
	// 同时会执行淘汰
	Put(k, v interface{})

	// Prune 整理cache内部数据
	Prune()
}
