package savedbwrapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoolList(t *testing.T) {
	t.Run("V1", func(t *testing.T) {
		l := NewboolListOld(func(v []bool) {}, []bool{}, nil)

		l.Append(true)
		assert.Equal(t, []bool{true}, l.GetAllReadOnly())

		l.Append(true)
		assert.Equal(t, []bool{true, true}, l.GetAllReadOnly())

		l.Append(true)
		assert.Equal(t, []bool{true, true, true}, l.GetAllReadOnly())

		l.Insert(1, false)
		assert.Equal(t, []bool{true, false, true, true}, l.GetAllReadOnly())

		l.Insert(4, false)
		assert.Equal(t, []bool{true, false, true, true, false}, l.GetAllReadOnly())
	})

	t.Run("V2", func(t *testing.T) {
		l := NewboolList(func(v []bool) {}, []bool{}, nil)

		l.Append(true)
		assert.Equal(t, []bool{true}, l.GetAllReadOnly())

		l.Append(true)
		assert.Equal(t, []bool{true, true}, l.GetAllReadOnly())

		l.Append(true)
		assert.Equal(t, []bool{true, true, true}, l.GetAllReadOnly())

		l.Insert(1, false)
		assert.Equal(t, []bool{true, false, true, true}, l.GetAllReadOnly())

		l.Insert(4, false)
		assert.Equal(t, []bool{true, false, true, true, false}, l.GetAllReadOnly())
	})

}

func BenchmarkBoolList(b *testing.B) {
	n := 8192

	b.Run("V1", func(b *testing.B) {
		l := NewboolListOld(func(v []bool) {}, []bool{}, nil)
		b.Run("Append", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l.Append(true)
				if l.Len() > n {
					l.Clear()
				}
			}
		})

		l = NewboolListOld(func(v []bool) {}, []bool{}, nil)
		b.Run("Insert", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l.Insert(i/2, true)
				if l.Len() > n {
					l.Clear()
				}
			}
		})
	})

	b.Run("V2", func(b *testing.B) {
		// V2
		l2 := NewboolList(func(v []bool) {}, []bool{}, nil)

		b.Run("Append V2", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l2.Append(true)
				if l2.Len() > n {
					l2.Clear()
				}
			}
		})

		l2 = NewboolList(func(v []bool) {}, []bool{}, nil)
		b.Run("Insert V2", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l2.Insert(i/2, false)
				if l2.Len() > n {
					l2.Clear()
				}
			}
		})
	})
}
