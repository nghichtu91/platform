package etcd

import (
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	etcdInitOnce sync.Once
)

func TestMain(m *testing.M) {
	InitEtcdTest([]string{"http://127.0.0.1:2379"})
	code := m.Run()
	os.Exit(code)
}

// InitEtcdTest 去掉log避免循环引用
func InitEtcdTest(endPoins []string) {
	etcdInitOnce.Do(func() {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:            endPoins,
			DialTimeout:          5 * time.Second,
			DialKeepAliveTime:    1 * time.Second,
			DialKeepAliveTimeout: 500 * time.Millisecond,
		})
		if err != nil {
			panic(err)
		}
		etcdClient = cli
	})
}

func TestPutWithPreRev(t *testing.T) {
	var wg sync.WaitGroup
	k := "root/server/testputwithrev"

	vers := make(map[int64]struct{})
	var mutex sync.RWMutex

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			ver, err := PutWithPreRev(k, strconv.Itoa(i))
			if err != nil {
				t.Logf("failed to put with rev, err %v", err)
			}
			// fmt.Println("shard ", i, " with ver ", ver)

			// 确保原子性
			mutex.RLock()
			_, ok := vers[ver+1]
			assert.False(t, ok)
			mutex.RUnlock()

			mutex.Lock()
			vers[ver+1] = struct{}{}
			mutex.Unlock()

			wg.Done()
		}(i)
	}

	wg.Wait()

	assert.Nil(t, Delete(k))
}

func BenchmarkPutWithPreRevParallel(b *testing.B) {
	k := "root/server/testputwithrev"
	b.SetParallelism(runtime.NumCPU() * 10)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			PutWithPreRev(k, strconv.Itoa(999))
		}
	})
}

func TestKeyNotExist(t *testing.T) {

	InitEtcdTest([]string{"http://127.0.0.1:2379"})

	errKey := "not exist"
	val, err := ExistThenGet(errKey)
	assert.Equal(t, err, ErrNotExist)
	assert.Empty(t, val)

	val, err = Get(errKey)
	assert.Empty(t, val)
	assert.Empty(t, err)
}