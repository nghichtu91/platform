package gate

import "testing"

func BenchmarkSelect(b *testing.B) {
	idleShared := make(chan struct{})

	ps := []int{16, 64, 1024, 4096, 16384}

	for _, p := range ps {
		b.SetParallelism(p)

		b.Run("shared", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				ch := make(chan int, 1)
				for pb.Next() {
					select {
					case ch <- 1:
					case <-ch:
					case <-idleShared:
					}
				}
			})
		})

		b.Run("private", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				ch := make(chan int, 1)
				idlePrivate := make(chan struct{})
				for pb.Next() {
					select {
					case ch <- 1:
					case <-ch:
					case <-idlePrivate:
					}
				}
			})
		})
	}
}

func BenchmarkSelectShared(b *testing.B) {
	idleShared := make(chan struct{})
	b.RunParallel(func(pb *testing.PB) {
		ch := make(chan int, 1)
		for pb.Next() {
			select {
			case ch <- 1:
			case <-ch:
			case <-idleShared:
			}
		}
	})
}

func BenchmarkSelectPrivate(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		ch := make(chan int, 1)
		idlePrivate := make(chan struct{})
		for pb.Next() {
			select {
			case ch <- 1:
			case <-ch:
			case <-idlePrivate:
			}
		}
	})
}
