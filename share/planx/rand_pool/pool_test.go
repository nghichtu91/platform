package rand_pool

import (
	"math/rand"
	"testing"
)

func BenchmarkRand(b *testing.B) {
	b.SetParallelism(5000)

	b.Run("rand", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				rand.Int31()
			}
		})
	})

	b.Run("rand_pool", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Int31()
			}
		})
	})
}
