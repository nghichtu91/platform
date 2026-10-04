package timysql

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
)

var (
	logInitOnce  sync.Once
	etcdInitOnce sync.Once
)

func InitLog() {
	logInitOnce.Do(func() {
		zaplog.InitZapLog("", nil, "")
	})
}

func InitEtcd() {
	etcdInitOnce.Do(func() {
		etcd.InitEtcd([]string{"http://127.0.0.1:2379/"})
	})
}

func TestMain(m *testing.M) {
	InitLog()
	InitEtcd()
	code := m.Run()
	tilogs.Close()
	os.Exit(code)
}

func TestIncr(t *testing.T) {
	sdb, err := NewMySqlConn(&MySqlDBCfg{
		Driver:     "mysql",
		URL:        "tcp(localhost:3306)",
		DBName:     "jws2_global",
		TableName:  NameTableMisc,
		User:       "root",
		Pwd:        "123456",
		PriKeyName: KeyUIDType,
	})
	assert.Nil(t, err)
	assert.NotNil(t, sdb)

	t.Run("INCR", func(t *testing.T) {
		v, err := sdb.Incr("gamexMerge")
		assert.Nil(t, err)
		assert.NotNil(t, v)
	})

	t.Run("INCR Parallel", func(t *testing.T) {
		var (
			wg      sync.WaitGroup
			rwMutex sync.RWMutex
		)

		sdb.db.SetMaxOpenConns(100)

		n := 100
		counter := make(map[int64]struct{}, n)

		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				v, err := sdb.Incr("gamexMerge")
				assert.Nil(t, err)
				rwMutex.RLock()
				_, ok := counter[v]
				assert.False(t, ok)
				rwMutex.RUnlock()

				rwMutex.Lock()
				counter[v] = struct{}{}
				rwMutex.Unlock()

				wg.Done()
			}()
		}

		wg.Wait()

		assert.Equal(t, n, len(counter))
	})
}
