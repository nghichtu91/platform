package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringCompare(t *testing.T) {
	tests := []struct {
		a, b string
		less bool
	}{
		{"1234", "1240", true},
		{"2230", "2400", true},
		{"1024", "1124", true},
	}

	for i, test := range tests {
		assert.Equal(t, test.less, test.a < test.b, i)
	}
}
