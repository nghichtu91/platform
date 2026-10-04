package file_hash_cache

import (
	"testing"
)

func BenchmarkGenHash(b *testing.B) {
	b.ReportAllocs()
	path := "/Users/jiangyi/share/planx/file_hash_cache/README.md"

	for i := 0; i < b.N; i++ {
		GenHash(path)
	}
}
