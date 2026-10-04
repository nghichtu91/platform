package chat_util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegCheckFace(t *testing.T) {
	tests := []struct {
		in     string
		result bool
	}{
		{"[q][q]", true},
		{"[0]", true},
		{"[1][1]", true},
		{"[1]", true},
		{"[q]haha[q]", false},
		{"[A][A][A]", true},
		{"test", false},
		{"[test]", false},
		{"[[12]]", false},
		{"[[1]2]]", false},
	}

	for i, test := range tests {
		assert.Equal(t, test.result, IsPureEmoji(test.in), i)
	}
}
