package savedbwrapper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInt64List(t *testing.T) {
	t.Run("V1", func(t *testing.T) {
		l := Newint64ListOld(func(v []int64) {}, []int64{}, nil)

		l.Append(3)
		assert.Equal(t, []int64{3}, l.GetAllReadOnly())

		l.Append(2)
		assert.Equal(t, []int64{3, 2}, l.GetAllReadOnly())

		l.Append(1)
		assert.Equal(t, []int64{3, 2, 1}, l.GetAllReadOnly())

		l.Insert(1, 4)
		assert.Equal(t, []int64{3, 4, 2, 1}, l.GetAllReadOnly())

		l.Insert(4, 5)
		assert.Equal(t, []int64{3, 4, 2, 1, 5}, l.GetAllReadOnly())
	})

	t.Run("V2", func(t *testing.T) {
		l := Newint64List(func(v []int64) {}, []int64{}, nil)

		l.Append(3)
		assert.Equal(t, []int64{3}, l.GetAllReadOnly())

		l.Append(2)
		assert.Equal(t, []int64{3, 2}, l.GetAllReadOnly())

		l.Append(1)
		assert.Equal(t, []int64{3, 2, 1}, l.GetAllReadOnly())

		l.Insert(1, 4)
		assert.Equal(t, []int64{3, 4, 2, 1}, l.GetAllReadOnly())

		l.Insert(4, 5)
		assert.Equal(t, []int64{3, 4, 2, 1, 5}, l.GetAllReadOnly())
	})

}

func BenchmarkInt64List(b *testing.B) {
	n := 8192

	b.Run("V1", func(b *testing.B) {
		l := Newint64ListOld(func(v []int64) {}, []int64{}, nil)
		b.Run("Append", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l.Append(128)
				if l.Len() > n {
					l.Clear()
				}
			}
		})

		l = Newint64ListOld(func(v []int64) {}, []int64{}, nil)
		b.Run("Insert", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l.Insert(i/2, 128)
				if l.Len() > n {
					l.Clear()
				}
			}
		})

		b.Run("GetReadOnly", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l.GetAllReadOnly()
			}
		})
	})

	b.Run("V2", func(b *testing.B) {
		// V2
		l2 := Newint64List(func(v []int64) {}, []int64{}, nil)

		b.Run("Append V2", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l2.Append(128)
				if l2.Len() > n {
					l2.Clear()
				}
			}
		})

		l2 = Newint64List(func(v []int64) {}, []int64{}, nil)
		b.Run("Insert V2", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l2.Insert(i/2, 128)
				if l2.Len() > n {
					l2.Clear()
				}
			}
		})

		b.Run("GetReadOnly", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l2.GetAllReadOnly()
			}
		})
	})
}

func TestListConcurrent(t *testing.T) {
	l2 := Newint64List(func(v []int64) {}, []int64{}, nil)

	l2.Append(233)

	go func(l []int64) {
		<-time.After(100 * time.Millisecond)
		t.Logf("l: %p %v", l, l)
	}(l2.GetAllReadOnly())

	l2.Append(256)

	<-time.After(500 * time.Millisecond)

	t.Logf("l2: %p %v", l2.GetAllReadOnly(), l2.GetAllReadOnly())
}
