package savedbwrapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFloat32List(t *testing.T) {
	t.Run("V1", func(t *testing.T) {
		l := Newfloat32ListOld(func(v []float32) {}, []float32{}, nil)

		l.Append(3)
		assert.Equal(t, []float32{3}, l.GetAllReadOnly())

		l.Append(2)
		assert.Equal(t, []float32{3, 2}, l.GetAllReadOnly())

		l.Append(1)
		assert.Equal(t, []float32{3, 2, 1}, l.GetAllReadOnly())

		l.Insert(1, 4)
		assert.Equal(t, []float32{3, 4, 2, 1}, l.GetAllReadOnly())

		l.Insert(4, 5)
		assert.Equal(t, []float32{3, 4, 2, 1, 5}, l.GetAllReadOnly())
	})

	t.Run("V2", func(t *testing.T) {
		l := Newfloat32List(func(v []float32) {}, []float32{}, nil)

		l.Append(3)
		assert.Equal(t, []float32{3}, l.GetAllReadOnly())

		l.Append(2)
		assert.Equal(t, []float32{3, 2}, l.GetAllReadOnly())

		l.Append(1)
		assert.Equal(t, []float32{3, 2, 1}, l.GetAllReadOnly())

		l.Insert(1, 4)
		assert.Equal(t, []float32{3, 4, 2, 1}, l.GetAllReadOnly())

		l.Insert(4, 5)
		assert.Equal(t, []float32{3, 4, 2, 1, 5}, l.GetAllReadOnly())
	})

}

func BenchmarkFloat32List(b *testing.B) {
	n := 8192

	b.Run("V1", func(b *testing.B) {
		l := Newfloat32ListOld(func(v []float32) {}, []float32{}, nil)
		b.Run("Append", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l.Append(128)
				if l.Len() > n {
					l.Clear()
				}
			}
		})

		l = Newfloat32ListOld(func(v []float32) {}, []float32{}, nil)
		b.Run("Insert", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l.Insert(i/2, 128)
				if l.Len() > n {
					l.Clear()
				}
			}
		})
	})

	b.Run("V2", func(b *testing.B) {
		// V2
		l2 := Newfloat32List(func(v []float32) {}, []float32{}, nil)

		b.Run("Append V2", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l2.Append(128)
				if l2.Len() > n {
					l2.Clear()
				}
			}
		})

		l2 = Newfloat32List(func(v []float32) {}, []float32{}, nil)
		b.Run("Insert V2", func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				l2.Insert(i/2, 128)
				if l2.Len() > n {
					l2.Clear()
				}
			}
		})
	})
}
