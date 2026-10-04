package battlex

import (
	"strings"
	"testing"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/nats_cli"
	"github.com/nghichtu91/platform/share/planx/servers/planxprotogen"
)

func BenchmarkSendMsg(b *testing.B) {
	// zaplog.InitZapLog("", nil, "")
	nats_cli.InitNats("12", "nats://127.0.0.1:4222", false)

	msg1 := &planxprotogen.BattleMsg{
		MsgData: []byte(strings.Repeat("strWithLength_16", 256)),
	}

	b.Run("SendMsg", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = SendMsg("12", "battlex1", etcd.Server_Gamex, msg1)
		}
	})

	b.Run("SendMsg Parallel 8", func(b *testing.B) {
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = SendMsg("12", "battlex1", etcd.Server_Gamex, msg1)
			}
		})
	})

	b.Run("SendMsg Parallel 256", func(b *testing.B) {
		b.ReportAllocs()

		b.SetParallelism(256)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = SendMsg("12", "battlex1", etcd.Server_Gamex, msg1)
			}
		})
	})

	b.Run("SendMsg Parallel 2048", func(b *testing.B) {
		b.ReportAllocs()

		b.SetParallelism(4096)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = SendMsg("12", "battlex1", etcd.Server_Gamex, msg1)
			}
		})
	})
}
