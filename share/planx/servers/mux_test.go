package servers

import (
	"sync"
	"testing"

	"github.com/nghichtu91/platform/share/x/common/msg/scene/pb"
)

func BenchmarkRequest(b *testing.B) {
	// 验证Request能否放到堆上

	// req := Request{
	// 	Code:     32,
	// 	RawBytes: []byte("go tool pprof -http=127.0.0.1:8089 profile.pb\ngo tool pprof -svg game_profile.pb > gamex_cpu.svg\n"),
	// }

	sMap := make(map[int]Request, 128)

	pMap := make(map[int]*Request, 128)

	b.Run("struct", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			sMap[i%100] = Request{
				Code:     32,
				RawBytes: []byte("go tool pprof -http=127.0.0.1:8089 profile.pb\ngo tool pprof -svg game_profile.pb > gamex_cpu.svg\n"),
			}
		}
	})

	b.Run("ptr", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			pMap[i%100] = &Request{
				Code:     32,
				RawBytes: []byte("go tool pprof -http=127.0.0.1:8089 profile.pb\ngo tool pprof -svg game_profile.pb > gamex_cpu.svg\n"),
			}
		}
	})
}

func BenchmarkPoolPut(b *testing.B) {
	// 验证pb.SceneMsg能否放到堆上

	pp := sync.Pool{New: func() interface{} {
		return new(pb.SceneMsg)
	}}

	b.Run("struct", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = pb.SceneMsg{
				Code:       32,
				MsgData:    []byte("go tool pprof -http=127.0.0.1:8089 profile.pb\ngo tool pprof -svg game_profile.pb > gamex_cpu.svg\n"),
				AllLineNpc: &pb.AllLineNpcInfo{},
			}
		}
	})

	b.Run("ptr", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = &pb.SceneMsg{
				Code:       32,
				MsgData:    []byte("go tool pprof -http=127.0.0.1:8089 profile.pb\ngo tool pprof -svg game_profile.pb > gamex_cpu.svg\n"),
				AllLineNpc: &pb.AllLineNpcInfo{},
			}
		}
	})

	b.Run("ptr pool fully get", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			msg := pp.Get().(*pb.SceneMsg)
			msg.Code = 32
			msg.MsgData = []byte("go tool pprof -http=127.0.0.1:8089 profile.pb\ngo tool pprof -svg game_profile.pb > gamex_cpu.svg\n")
			msg.AllLineNpc = &pb.AllLineNpcInfo{}
			msg.Reset()
			pp.Put(msg)
		}
	})

	b.Run("ptr pool more put than get", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			msg := pp.Get().(*pb.SceneMsg)
			msg.Code = 32
			msg.MsgData = []byte("go tool pprof -http=127.0.0.1:8089 profile.pb\ngo tool pprof -svg game_profile.pb > gamex_cpu.svg\n")
			msg.AllLineNpc = &pb.AllLineNpcInfo{}
			msg.Reset()
			if i%2 == 0 {
				pp.Put(msg)
			}
		}
	})
}
