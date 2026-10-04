package util

import (
	"testing"
)

func BenchmarkInt32ArrayToStringArray(b *testing.B) {
	w := "Is Unicorn"

	b.Run("is Ucfirst", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			IsUcfirst(w)
		}
	})

	b.Run("is Lcfirst", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			IsLcfirst(w)
		}
	})

	type logiclogStruct struct {
		StrSlice []string
	}

	lls := &logiclogStruct{}
	intArray := []int{8, 3, 11, 23312, 51132, 112321, 11, 1, 1, 1, 534543, 3232, 2, 223, 2232, 1123, 141}
	int32Array := []int32{8, 3, 11, 23312, 51132, 112321, 11, 1, 1, 1, 534543, 3232, 2, 223, 2232, 1123, 141}
	uint32Array := []uint32{8, 3, 11, 23312, 51132, 112321, 11, 1, 1, 1, 534543, 3232, 2, 223, 2232, 1123, 141}

	b.Run("int array to string", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = IntArrayToString(intArray, ",")
		}
	})

	b.Run("int32 array to string", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = Int32ArrayToString(int32Array, ",")
		}
	})

	b.Run("uint32 array to string", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = Int32ArrayToString(int32Array, ",")
		}
	})

	b.Run("int32 array to string slice", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			lls.StrSlice = Int32ArrayToStringArray(int32Array)
		}
	})

	b.Run("uint32 array to string slice", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			lls.StrSlice = Uint32ArrayToStringArray(uint32Array)
		}
	})

	b.Run("slice int32 to string", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SliceInt32ToString(int32Array, &lls.StrSlice)
		}
	})

	b.Run("slice uint32 to string", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			SliceUint32ToString(uint32Array, &lls.StrSlice)
		}
	})
}
