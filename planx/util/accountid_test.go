package util

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseShardIDFromAcid(t *testing.T) {
	tests := []struct {
		acid string
		sid  int
	}{
		{"", 0},
		{":", 0},
		{"::", 0},
		{":::", 0},
		{":25:", 25},
		{"10:160001:acid", 160001},
	}

	for i, test := range tests {
		assert.Equal(t, test.sid, ParseShardIDFromAcid(test.acid), i)
	}
}

func OldParseShardIDFromAcid(acid string) int {
	strs := strings.Split(acid, ":")
	sid, _ := strconv.Atoi(strs[0])
	return sid
}

func BenchmarkParseShardIDFromAcid(b *testing.B) {
	acid := "10:160001:acid"

	b.Run("Old", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			OldParseShardIDFromAcid(acid)
		}
	})

	b.Run("New", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			ParseShardIDFromAcid(acid)
		}
	})
}
