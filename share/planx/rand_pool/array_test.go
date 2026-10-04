package rand_pool

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearch(t *testing.T) {
	sections := []int{1, 4, 6, 10, 15}

	assert.Equal(t, 0, sort.Search(len(sections), func(i int) bool {
		return sections[i] > 0
	}))

	assert.Equal(t, 1, sort.Search(len(sections), func(i int) bool {
		return sections[i] > 1
	}))

	assert.Equal(t, 1, sort.Search(len(sections), func(i int) bool {
		return sections[i] > 3
	}))

	assert.Equal(t, 2, sort.Search(len(sections), func(i int) bool {
		return sections[i] > 4
	}))

	assert.Equal(t, 4, sort.Search(len(sections), func(i int) bool {
		return sections[i] > 14
	}))

	assert.Equal(t, 5, sort.Search(len(sections), func(i int) bool {
		return sections[i] > 15
	}))

	assert.Equal(t, 5, sort.Search(len(sections), func(i int) bool {
		return sections[i] > 32
	}))
}

func TestArrayRander(t *testing.T) {
	rd := &RealArrayRander{}

	assert.Equal(t, ErrTotalWeightZero, rd.Init(nil))
	assert.Equal(t, ErrTotalWeightZero, rd.Init([]uint32{0, 0, 0, 0, 0}))

	assert.Equal(t, -1, rd.Rand())

	assert.Nil(t, rd.Init([]uint32{0, 0, 0, 100}))
	for i := 0; i < 10; i++ {
		assert.Equal(t, 3, rd.Rand())
	}

	assert.Nil(t, rd.Init([]uint32{0, 0, 5, 15, 25}))

	counter := make(map[int]int)

	for i := 0; i < 1000; i++ {
		counter[rd.Rand()]++
	}

	t.Logf("total: %v", counter)
}

func BenchmarkArrayRander(b *testing.B) {
	rd := &RealArrayRander{}

	weights := []uint32{
		0, 1, 5, 12, 17, 22, 25, 32,
		22, 18, 512, 333, 256, 7, 9, 8,
		24, 8, 1024, 7, 27, 16, 16, 16,
		128, 256, 666, 32, 16, 8, 4, 2,
		618, 222, 111, 333, 444, 555, 666, 777,
		1, 2, 3, 4, 5, 6, 7, 8,
		27, 72, 28, 82, 25, 52, 37, 73,
	}

	b.Run("Init", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			rd.Init(weights)
		}
	})

	b.Run("Rand", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			rd.Rand()
		}
	})
}
