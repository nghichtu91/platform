package models

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

func BenchmarkMongoAuth(b *testing.B) {
	tilogs.Close()

	mdb := &DBByMongoDB{}
	assert.Nil(b, mdb.Init(DBConfig{
		MongoDBName: "AuthDB",
		MongoDBUrl:  "127.0.0.1:27017",
	}))

	mdbv2 := &DBByMongoDBV2{}
	assert.Nil(b, mdbv2.Init(DBConfig{
		MongoDBName: "AuthDB",
		MongoDBUrl:  "127.0.0.1:27017",
	}))

	b.Run("mgo copy GetDeviceInfo", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			mdb.GetDeviceInfo("E51AB9B0-43D3-5FE0-8037-6441319D974E@07151522156391@apple.com", true)
		}
	})

	b.Run("mgo clone GetDeviceInfo", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			mdb.GetDeviceInfoClone("E51AB9B0-43D3-5FE0-8037-6441319D974E@07151522156391@apple.com", true)
		}
	})

	b.Run("mongo v2 official GetDeviceInfo", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			mdbv2.GetDeviceInfo("E51AB9B0-43D3-5FE0-8037-6441319D974E@07151522156391@apple.com", true)
		}
	})

	b.SetParallelism(4000)

	b.Run("mgo copy para GetDeviceInfo", func(b *testing.B) {
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				mdb.GetDeviceInfo("E51AB9B0-43D3-5FE0-8037-6441319D974E@07151522156391@apple.com", true)
			}
		})
	})

	b.Run("mgo clone para GetDeviceInfo", func(b *testing.B) {
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				mdb.GetDeviceInfoClone("E51AB9B0-43D3-5FE0-8037-6441319D974E@07151522156391@apple.com", true)
			}
		})
	})

	b.Run("mongo v2 official GetDeviceInfo", func(b *testing.B) {
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				mdbv2.GetDeviceInfo("E51AB9B0-43D3-5FE0-8037-6441319D974E@07151522156391@apple.com", true)
			}
		})
	})
}
