package lru

import (
	"strconv"
	"testing"
)

func BenchmarkCache(b *testing.B) {
	n := 1024

	cases := make([]*struct{ k, v interface{} }, 0, n)

	for i := 0; i < n; i++ {
		cases = append(cases, &struct{ k, v interface{} }{k: int64(i), v: strconv.Itoa(i)})
	}

	c := NewCache(n)
	ic := NewIntCache(n)
	mc := NewMuxCache(n)

	b.Run("cache put", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			c.Put(cases[i%n].k, cases[i%n].v)
		}
	})

	b.Run("cache get", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			c.Get(cases[i%n].k)
		}
	})

	b.Run("cache prune", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			c.Prune()
		}
	})

	b.Run("int cache put", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			ic.Put(cases[i%n].k, cases[i%n].v)
		}
	})

	b.Run("int cache get", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			ic.Get(cases[i%n].k)
		}
	})

	b.Run("int cache prune", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			ic.Prune()
		}
	})

	b.Run("mux cache put", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			mc.Put(cases[i%n].k, cases[i%n].v)
		}
	})

	b.Run("mux cache get", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			mc.Get(cases[i%n].k)
		}
	})

	b.Run("mux cache prune", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			mc.Prune()
		}
	})
}
