package mergeinfo

import (
	"reflect"
	"testing"
)

func TestParseRangeStr(t *testing.T) {
	type args struct {
		param string
	}
	tests := []struct {
		name    string
		args    args
		want    [][]int
		wantErr bool
	}{
		{"nil1", args{""}, nil, false},
		{"nil2", args{"  "}, nil, false},

		{"err1", args{"aaa"}, nil, true},

		{"correct1", args{"1001"}, [][]int{{1001, 1001}}, false},
		{"correct2", args{"1001, 1003-1005"}, [][]int{{1001, 1001}, {1003, 1005}}, false},
		{"correct3", args{"1001, 1003-1005, 1009, 2001-2005"}, [][]int{{1001, 1001}, {1003, 1005}, {1009, 1009}, {2001, 2005}}, false},
		{"correct4", args{"10,999"}, [][]int{{10, 10}, {999, 999}}, false},
		{"correct4", args{"\"10,999\""}, [][]int{{10, 10}, {999, 999}}, false},

		{"invalid1", args{"-1"}, nil, true},
		{"invalid2", args{"1, 1-1"}, nil, true},
		{"invalid3", args{"1, 3-2, 5"}, nil, true},
		{"invalid4", args{"1001, 998-999"}, nil, true},
		{"invalid5", args{"1001, 1003--1005"}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRangeStr(tt.args.param)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRangeStr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseRangeStr() got = %v, want %v", got, tt.want)
			}
		})
	}
}
