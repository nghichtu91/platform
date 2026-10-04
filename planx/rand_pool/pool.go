package rand_pool

import (
	"math/rand"
	"sync"
	"time"
)

// 全局rand有锁，单个rand占用内存较多
// 使用对象池均衡性能和内存问题
// 封装了一些常用的全局随机函数

var rPool = sync.Pool{New: func() interface{} {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}}

func Get() *rand.Rand {
	return rPool.Get().(*rand.Rand)
}

func Put(r *rand.Rand) {
	rPool.Put(r)
}

func Intn(n int) int {
	rd := Get()
	defer Put(rd)
	return rd.Intn(n)
}

func Int31n(n int32) int32 {
	rd := Get()
	defer Put(rd)
	return rd.Int31n(n)
}

func Int31() int32 {
	rd := Get()
	defer Put(rd)
	return rd.Int31()
}

func Int63n(n int64) int64 {
	rd := Get()
	defer Put(rd)
	return rd.Int63n(n)
}

func Float32() float32 {
	rd := Get()
	defer Put(rd)
	return rd.Float32()
}

func Float64() float64 {
	rd := Get()
	defer Put(rd)
	return rd.Float64()
}
