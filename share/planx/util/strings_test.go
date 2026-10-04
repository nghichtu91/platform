package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUintFromString(t *testing.T) {
	tests := []struct {
		in  string
		out int
	}{
		{"", 0},
		{" ", 0},
		{"1", 1},
		{"121", 121},
		{" 1211 ", 1211},
		{" 22 33", 22},
		{"gmserver", 0},
		{"gamex160001", 160001},
		{"cross0", 0},
		{"cross1", 1},
		{"cross00", 0},
		{"cross01", 1},
		{"test111plus", 111},
		{"test111plus222", 111},
		{"极无双2", 2},
		{"😅", 0},
	}

	for i, test := range tests {
		assert.Equal(t, test.out, SimpleGetUintFromString(test.in), i)
		t.Logf("%v len %v", test, CalcGameStringLen(test.in))
	}
}
