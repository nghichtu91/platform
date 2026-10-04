package timeutil

import (
	"testing"
	"time"
)

func TestTimeCache(t *testing.T) {
	tc := NewTimeCache(100*time.Millisecond, 10)

	for i := 0; i < 30; i++ {
		t.Logf("timecache: %d, time.Now: %d", tc.GetCachedTimeNow(), time.Now().Unix())
		t.Logf("timecache nano: %d, time.Now nano: %d", tc.GetCachedTimeNowNano(), time.Now().UnixNano())
		time.Sleep(400 * time.Millisecond)
	}
}

func BenchmarkTimeCache(b *testing.B) {
	tc := NewTimeCache(100*time.Millisecond, 10)

	b.SetParallelism(5000)

	b.Run("time.Now", func(b *testing.B) {
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				time.Now().Unix()
			}
		})
	})

	b.Run("time cache", func(b *testing.B) {
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tc.GetCachedTimeNowNano()
			}
		})
	})
}
