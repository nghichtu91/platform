package mergeinfo

import (
	"os"
	"sync"
	"testing"

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
