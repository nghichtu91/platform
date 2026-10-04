package gate

import (
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/astaxie/beego/cache"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"

	"github.com/nghichtu91/platform/share/planx/safecache"
)

var randPool = sync.Pool{
	New: func() interface{} {
		return rand.New(rand.NewSource(time.Now().UnixNano()))
	},
}

func get() *rand.Rand {
	return randPool.Get().(*rand.Rand)
}

func put(r *rand.Rand) {
	if r == nil {
		return
	}
	randPool.Put(r)
}

const CNT = 1000

var randString []string

func randKey(r *rand.Rand) string {
	return randString[r.Intn(CNT)]
}

func randRange(r *rand.Rand) []string {
	const Range = 100
	start := r.Intn(CNT - 100)
	return randString[start : start+Range]
}

func init() {

	r := get()

	randString = make([]string, 0, CNT)
	for i := 0; i < CNT; i++ {
		var buf [24]byte
		r.Read(buf[:])
		randString = append(randString, string(buf[:]))
	}
}

func runTest(c cache.Cache) {
	r := get()
	defer put(r)
	switch n := r.Intn(50); {
	case n < 15:
		_ = c.Put(randKey(r), 100, time.Duration(0))
	case n < 30:
		c.Get(randKey(r))
	case n < 40:
		c.GetMulti(randRange(r))
	case n < 45:
		_ = c.ClearAll()
	default:
		c.IsExist(randKey(r))
	}
}

func TestCache(t *testing.T) {
	c, _ := safecache.NewSafeCache("213123131", "")
	c2 := NewSessionCache()

	runG := func(n int) {
		t.Logf("Run: %v", n)
		var wait sync.WaitGroup
		for i := 0; i < n; i++ {
			wait.Add(1)
			go func() {
				defer wait.Done()
				r := get()
				defer put(r)
				for i2 := 0; i2 < 1000; i2++ {
					key := randKey(r)
					assert.Equal(t, c.IsExist(key), c2.IsExist(key))
					key = randKey(r)
					assert.Equal(t, c.Put(key, 1, 0), c2.Put(key, 1, 0))
					key = randKey(r)
					assert.Equal(t, c.Delete(key), c2.Delete(key))
				}
			}()
		}
		wait.Wait()
	}

	runG(1)

	runG(runtime.GOMAXPROCS(0) * 2)

	// 并发执行可能会出问题, 但是最终结果应该是一致的(?)

	t.Logf("after")

	for _, key := range randString {
		assert.Equal(t, c.IsExist(key), c2.IsExist(key))
	}

}

func BenchmarkSession(b *testing.B) {
	var cnt atomic.Int32
	b.Run("lock1", func(b *testing.B) {
		c, _ := safecache.NewSafeCache("lock1"+strconv.Itoa(int(cnt.Add(1))), "")
		for i := 0; i < b.N; i++ {
			runTest(c)
		}
	})

	b.Run("atomic1", func(b *testing.B) {
		c2 := NewSessionCache()
		for i := 0; i < b.N; i++ {
			runTest(c2)
		}
	})

	b.Run("lock2", func(b *testing.B) {
		c3, _ := safecache.NewSafeCache("lock2"+strconv.Itoa(int(cnt.Add(1))), "")
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				runTest(c3)
			}
		})
	})

	b.Run("atomic2", func(b *testing.B) {
		c4 := NewSessionCache()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				runTest(c4)
			}
		})
	})
}
