package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTypeName(t *testing.T) {
	type TestStruct struct {
		TS int64
	}

	tests := []struct {
		in  interface{}
		out string
	}{
		{nil, ""},
		{TestStruct{}, "TestStruct"},
		{&TestStruct{}, "TestStruct"},
	}

	for i, test := range tests {
		assert.Equal(t, test.out, GetTypeName(test.in), i)
	}
}

func FunctionNameFn() {}

func funcNameP() {}

func TestGetFunctionRuntimeName(t *testing.T) {
	testFn := func() {}

	tests := []struct {
		in  interface{}
		out string
	}{
		{nil, ""},
		{FunctionNameFn, "github.com/nghichtu91/platform/share/planx/util.FunctionNameFn"},
		{funcNameP, "github.com/nghichtu91/platform/share/planx/util.funcNameP"},
		{testFn, "github.com/nghichtu91/platform/share/planx/util.TestGetFunctionRuntimeName.func1"},
		{func() {}, "github.com/nghichtu91/platform/share/planx/util.TestGetFunctionRuntimeName.func2"},
	}

	for i, test := range tests {
		assert.Equal(t, test.out, GetFunctionRuntimeName(test.in), i)
	}
}
