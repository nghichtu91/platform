package collection

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntSet(t *testing.T) {
	set := NewIntSet()
	set.Range(func(key int) bool { return true })
	set.Range(nil)
	require.True(t, set.Len() == 0)
	set.Adds(1, 2, 3)
	if set.Len() != 3 {
		t.Errorf("Expected set.Len() to be 3, but got %d", set.Len())
	}
	require.True(t, set.Add(5))
	require.False(t, set.Add(5))
	require.True(t, set.Remove(5))
	if !set.Contain(1) {
		t.Errorf("Expected set to contain 1, but it did not")
	}
	if set.Remove(4) {
		t.Errorf("Expected set.Remove(4) to return false, but it returned true")
	}
	if !set.Remove(2) {
		t.Errorf("Expected set.Remove(2) to return true, but it returned false")
	}
	if set.Len() != 2 {
		t.Errorf("Expected set.Len() to be 2, but got %d", set.Len())
	}
	if set.Intersection(NewIntSet(2, 3, 4)).Len() != 1 {
		t.Errorf("Expected set.Intersection(NewIntSet(2, 3, 4)) to contain 3, but it did not")
	}
	if set.Union(NewIntSet(3, 4, 5)).Len() != 4 {
		t.Errorf("Expected set.Union(NewIntSet(3, 4, 5)) to contain 1, 3, 4, and 5, but it did not")
	}
	if set.Difference(NewIntSet(1, 3)).Len() != 0 {
		t.Errorf("Expected set.Difference(NewIntSet(1, 3)) to be empty, but it was not")
	}
	if set.Difference(NewIntSet(5, 6)).Len() == 0 {
		t.Errorf("Expected set.Difference(NewIntSet(5, 6)) to be not empty, but it was not")
	}

	set.Removes(1, 2, 3)
	require.False(t, set.Contain(1))
	require.False(t, set.Contain(2))
	require.False(t, set.Contain(3))

	set.Adds(1, 2, 3, 4, 5, 6)
	set.Range(func(key int) bool {
		return false
	})
	require.Equal(t, NewIntSet(set.Slice()...), set)
	set.Clear()
	require.Equal(t, set.Len(), 0)
	set.Clear()
	require.Equal(t, set.Len(), 0)
	set.Removes()
	require.Equal(t, set.Len(), 0)

	set.Adds(1, 2, 3, 4)
	require.Equal(t, set.Intersection(NewIntSet(2, 3)), NewIntSet(2, 3))
}
