package metrics

import (
	rand2 "crypto/rand"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

var msgPool = sync.Pool{
	New: func() interface{} {
		return new(msgInfo)
	},
}

func loop(ch chan msgInfo) {
	for {
		select {
		case info := <-ch:
			_ = info
		}
	}
}

func send(ch chan msgInfo) {
	select {
	case ch <- msgInfo{
		metricsName:   "metricsName",
		clientMsgName: "clientMsgName.clientMsgName.clientMsgName.clientMsgName",
	}:
	}
}

func loopPtr(ch chan *msgInfo) {
	for {
		select {
		case info := <-ch:
			_ = info
		}
	}
}

func sendPtr(ch chan *msgInfo) {
	select {
	case ch <- &msgInfo{
		metricsName:   "metricsName",
		clientMsgName: "clientMsgName.clientMsgName.clientMsgName.clientMsgName",
	}:
	}
}

func loopPool(ch chan *msgInfo) {
	for {
		select {
		case info := <-ch:
			_ = info
			msgPool.Put(info)
		}
	}
}

func sendPool(ch chan *msgInfo) {
	msg := msgPool.Get().(*msgInfo)
	msg.metricsName = "metricsName"
	msg.clientMsgName = "clientMsgName.clientMsgName.clientMsgName.clientMsgName"
	select {
	case ch <- msg:
	}
}

func BenchmarkSent(b *testing.B) {
	b.Run("ptr", func(b *testing.B) {
		b.ReportAllocs()

		ch := make(chan *msgInfo, 10240)
		go loopPtr(ch)

		for i := 0; i < b.N; i++ {
			sendPtr(ch)
		}
	})

	b.Run("struct", func(b *testing.B) {
		b.ReportAllocs()

		ch := make(chan msgInfo, 10240)
		go loop(ch)

		for i := 0; i < b.N; i++ {
			send(ch)
		}
	})

	b.Run("pool", func(b *testing.B) {
		b.ReportAllocs()

		ch := make(chan *msgInfo, 10240)
		go loopPool(ch)

		for i := 0; i < b.N; i++ {
			sendPool(ch)
		}
	})
}

func BenchmarkIterMap(b *testing.B) {
	m := make(map[string]int, 512)
	for i := 0; i < 512; i++ {
		m[strconv.Itoa(i)] = i
	}

	b.Run("iter", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for k, v := range m {
				_ = k
				_ = v
			}
		}
	})

	b.Run("remake", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			m = make(map[string]int, 512)
		}
	})
}

func BenchmarkCommandMetric(b *testing.B) {

	const Count = 10000
	const BufSize = 24
	var buf [BufSize]byte

	var allKeys []string
	for i := 0; i < Count; i++ {
		_, _ = rand2.Read(buf[:])
		allKeys = append(allKeys, string(buf[:]))
	}

	genKey := func(r *rand.Rand) string {
		return allKeys[r.Intn(len(allKeys))]
	}

	genParallelKey := func() string {
		return allKeys[rand.Intn(len(allKeys))]

	}

	metric := func() *CommandMetrics {
		var m CommandMetrics
		m.SetPrefix("123123")
		return &m
	}

	b.ResetTimer()
	var now = time.Now()
	b.Run("Single", func(b *testing.B) {
		var m = metric()
		var source = rand.NewSource(time.Now().UnixNano())
		r := rand.New(source)
		for i := 0; i < b.N; i++ {
			m.CountTime(genKey(r), now)
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		var m = metric()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.CountTime(genParallelKey(), now)
			}
		})
	})
}
