package util

import (
	"testing"
)

func TestUInt32Merge3(t *testing.T) {
	type args struct {
		high uint32
		mid  uint32
		low  uint32
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "test",
			args: args{
				high: 1,
				mid:  2,
				low:  3,
			},
			want: 281475010265091,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UInt32Merge3(tt.args.high, tt.args.mid, tt.args.low); got != tt.want {
				t.Errorf("UInt32Merge3() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUInt64Split3(t *testing.T) {
	type args struct {
		in         uint64
		offsetHigh int
		offsetMid  int
	}
	tests := []struct {
		name     string
		args     args
		wantHigh uint32
		wantMid  uint32
		wantLow  uint32
	}{
		{
			name: "test",
			args: args{
				in:         281475010265091,
				offsetHigh: 48,
				offsetMid:  24,
			},
			wantHigh: 1,
			wantMid:  2,
			wantLow:  3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHigh, gotMid, gotLow := UInt64Split3(tt.args.in)
			if gotHigh != tt.wantHigh {
				t.Errorf("UInt64Split3() gotHigh = %v, want %v", gotHigh, tt.wantHigh)
			}
			if gotMid != tt.wantMid {
				t.Errorf("UInt64Split3() gotMid = %v, want %v", gotMid, tt.wantMid)
			}
			if gotLow != tt.wantLow {
				t.Errorf("UInt64Split3() gotLow = %v, want %v", gotLow, tt.wantLow)
			}
		})
	}

	t.Logf("%d", 1<<24)
}
