package util

import (
	"reflect"
	"runtime"
)

// GetTypeName 返回对象类型名称
// 如果输入为nil，返回""
func GetTypeName(st interface{}) string {
	if st == nil {
		return ""
	}
	switch t := reflect.TypeOf(st); t.Kind() {
	case reflect.Ptr:
		return t.Elem().Name()
	default:
		return t.Name()
	}
}

// GetFunctionRuntimeName 返回函数runtime名称
// 主要用于跟踪waitGroup执行情况
func GetFunctionRuntimeName(fn interface{}) string {
	if fn == nil {
		return ""
	}
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}
