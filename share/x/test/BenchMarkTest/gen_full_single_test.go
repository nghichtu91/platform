package BenchMarkTest

import (
	"testing"

	"github.com/nghichtu91/platform/share/x/test/logic"
)

const (
	ProjectID = uint64(0)
	BatchID   = uint64(0)
	GroupID   = uint64(0)
	Count     = uint64(100)
)

func Benchmark_GenFullSingle(b *testing.B) {
	b.N = 100
	for i := 0; i < b.N; i++ {
		logic.Gen_Gift_Code(ProjectID, BatchID, GroupID, Count)
	}
}
