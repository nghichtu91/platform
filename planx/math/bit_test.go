package math

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRound2N(t *testing.T) {
	tests := []struct {
		in, out int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{15, 16},
		{65536, 65536},
		{1024*1024 - 1, 1024 * 1024},
	}

	for i, test := range tests {
		assert.Equal(t, test.out, Round2N(test.in), i)
	}
}
