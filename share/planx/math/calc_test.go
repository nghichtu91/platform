package math

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeCalc(t *testing.T) {

	type Uint64Case struct {
		Old      uint64
		Num      uint64
		Backup   uint64
		WantNum  uint64
		WantOver bool
	}

	const (
		MU64 = math.MaxUint64
	)

	t.Run("Uint64Add", func(t *testing.T) {

		for _, c := range []Uint64Case{
			{
				Old:      MU64 - 1,
				Num:      2,
				Backup:   10,
				WantNum:  10,
				WantOver: true,
			},
			{
				Old:      MU64 - 10,
				Num:      10,
				Backup:   100,
				WantNum:  MU64,
				WantOver: false,
			},
			{
				Old:      10,
				Num:      20,
				Backup:   0,
				WantNum:  30,
				WantOver: false,
			},
		} {
			num, over := SafeAddUint64(c.Old, c.Num, c.Backup)
			assert.Equal(t, c.WantNum, num)
			assert.Equal(t, over, c.WantOver)
		}

	})

	t.Run("Uint64Sub", func(t *testing.T) {
		for _, c := range []Uint64Case{
			{
				Old:      1,
				Num:      2,
				Backup:   10,
				WantNum:  10,
				WantOver: true,
			},
			{
				Old:      10,
				Num:      10,
				Backup:   0,
				WantNum:  0,
				WantOver: false,
			},
			{
				Old:      20,
				Num:      10,
				Backup:   100,
				WantNum:  10,
				WantOver: false,
			},
		} {
			num, over := SafeSubUint64(c.Old, c.Num, c.Backup)
			assert.Equal(t, c.WantNum, num)
			assert.Equal(t, over, c.WantOver)
		}
	})
}

var (
	g uint64
)

// addUint642 仅用于测试, 用于对比 AddUint64 的性能
// AddUint64 主要是针对0个, 1个, 参数的情况进行了优化, 2个以上的参数没有优化
func addUint642(a uint64, nums ...uint64) uint64 {
	for _, b := range nums {
		a, _ = SafeAddUint64(a, b, math.MaxUint64)
		if a == math.MaxUint64 {
			return a
		}
	}
	return a
}

func BenchmarkAdd(b *testing.B) {
	b.Run("SafeAddUint64", func(b *testing.B) {
		var v uint64
		for i := 0; i < b.N; i++ {
			v, _ = SafeAddUint64(1, 1, 0)
		}
		g = v
	})

	for _, c := range [][]uint64{
		{},
		{1},
		{1, 2},
		{1, 2, 3},
	} {
		ln := len(c)
		b.Run("AddUint64"+strconv.Itoa(ln), func(b *testing.B) {
			var v uint64
			for i := 0; i < b.N; i++ {
				v = AddUint64(1, c...)
			}
			g = v
		})

		b.Run("addUint642"+strconv.Itoa(ln), func(b *testing.B) {
			var v uint64
			for i := 0; i < b.N; i++ {
				v = addUint642(1, c...)
			}
			g = v
		})
	}

	_ = g
}
