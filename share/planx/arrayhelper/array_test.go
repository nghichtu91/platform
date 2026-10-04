package arrayhelper

import (
	"strconv"
	"testing"
)

func BenchmarkMaps(b *testing.B) {
	n := 1000000

	keysI := make([]int, n)
	keysS := make([]string, n)
	for i := 0; i < n; i++ {
		keysI[i] = i
		keysS[i] = strconv.Itoa(i)
	}

	i64Map := make(map[int]struct{}, n)
	strMap := make(map[string]struct{}, n)

	b.Run("int64 write", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			i64Map[keysI[i%n]] = struct{}{}
		}
	})

	b.Run("int64 read", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, ok := i64Map[keysI[i%n]]
			_ = ok
		}
	})

	b.Run("string write", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			strMap[keysS[i%n]] = struct{}{}
		}
	})

	b.Run("string read", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, ok := strMap[keysS[i%n]]
			_ = ok
		}
	})
}

func BenchmarkMaps2(b *testing.B) {
	iMap := make(map[int]struct{})
	sMap := make(map[string]struct{})

	b.Run("int64 write", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			iMap[1024] = struct{}{}
		}
	})

	b.Run("int64 read", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, ok := iMap[1024]
			_ = ok
		}
	})

	b.Run("string write", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			sMap["smap"] = struct{}{}
		}
	})

	b.Run("string read", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, ok := sMap["smap"]
			_ = ok
		}
	})
}
