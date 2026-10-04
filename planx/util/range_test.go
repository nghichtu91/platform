package util

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkIntIter(b *testing.B) {
	b.ReportAllocs()
	var c int
	for i := 0; i < b.N; i++ {
		for idx := range Iter(1000) {
			c = idx
		}
	}
	_ = c
}

func TestBatchRange(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	assert.Nil(t, BatchRange(1000, 45, func(start, end int) bool {
		t.Logf("start:%v, end: %v", start, end)
		if r.Intn(2) == 1 {
			t.Logf("break")
			return false
		}
		return true
	}))

	assert.Equal(t, ErrOverflow, BatchRange(math.MaxUint32, 1, func(start, end int) bool {
		return true
	}))
}

func TestBatchGroup(t *testing.T) {

	testCases := []struct {
		name   string
		length int
		batch  int
	}{
		{
			name:   "normal",
			length: 1000,
			batch:  41,
		},
		{
			name:   "small1",
			length: 5,
			batch:  8,
		},
		{
			name:   "small2",
			length: 8,
			batch:  7,
		},
		{
			name:   "equal",
			length: 8,
			batch:  8,
		},
		{
			name:   "zero",
			length: 0,
			batch:  8,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var total int
			var batchCnt int
			err := Batch(tc.length, tc.batch, func(start, end int) bool {
				t.Logf("start:%v, end: %v, count: %v", start, end, end-start)
				total += end - start
				batchCnt++
				return true
			})
			assert.Nil(t, err)
			assert.Equal(t, tc.length, total)
			assert.Equal(t, batchCnt, func() int {
				if tc.batch > tc.length {
					return tc.length
				}
				return tc.batch
			}())
		})
	}
}

func TestBatchIter(t *testing.T) {
	batch, iter, err := BatchIterator(1000, 41)
	assert.Nil(t, err)
	var tmp = make([][]int, 0, batch)
	var pre int
	iter(func(start int, end int) bool {
		require.Equal(t, pre, start)
		tmp = append(tmp, []int{start, end - 1})
		pre = end
		return true
	})
	t.Logf("%v", tmp)
}

func TestUpper(t *testing.T) {
	require.Equal(t, Upper(math.MaxInt32, math.MaxInt32-1), 2)
	require.Equal(t, Upper(math.MaxInt32, math.MaxInt32), 1)
	require.Equal(t, Upper(-1, math.MaxInt32), 0)
	require.Equal(t, Upper(0, math.MaxInt32), 0)
	require.Equal(t, Upper(10, -1), 10)
}
