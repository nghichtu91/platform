package util

import (
	"bytes"
	"strconv"
	"unicode"
	"unsafe"
)

// 首字母大写
func Ucfirst(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return ""
}

// 首字母小写
func Lcfirst(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}

// 是否首字母大写
func IsUcfirst(str string) bool {
	for _, v := range str {
		return unicode.ToUpper(v) == v
	}
	return false
}

// 是否首字母小写
func IsLcfirst(str string) bool {
	for _, v := range str {
		return unicode.ToLower(v) == v
	}
	return false
}

func Str2Bytes(s string) []byte {
	var data = struct {
		Str string
		Cap int
	}{
		Str: s,
		Cap: len(s),
	}
	return *(*[]byte)(unsafe.Pointer(&data))
}

func Bytes2Str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// 将int数组转为delim分割的字符串
// func IntArrayToString(a []int, delim string) string {
// 	return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
// }
// func Int32ArrayToString(a []int32, delim string) string {
// 	return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
// }
// func UInt32ArrayToString(a []uint32, delim string) string {
// 	return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
// }

// 将int数组转为delim分割的字符串
func IntArrayToString(a []int, delim string) string {
	if len(a) == 0 {
		return ""
	}

	buff := new(bytes.Buffer)
	for i := 0; i < len(a)-1; i++ {
		buff.WriteString(strconv.Itoa(a[i]))
		buff.WriteString(delim)
	}
	buff.WriteString(strconv.Itoa(a[len(a)-1]))

	return buff.String()
}

// 将int32数组转为delim分割的字符串
func Int32ArrayToString(a []int32, delim string) string {
	if len(a) == 0 {
		return ""
	}

	buff := new(bytes.Buffer)
	for i := 0; i < len(a)-1; i++ {
		buff.WriteString(strconv.Itoa(int(a[i])))
		buff.WriteString(delim)
	}
	buff.WriteString(strconv.Itoa(int(a[len(a)-1])))

	return buff.String()
}

// 将uint32数组转为delim分割的字符串
func UInt32ArrayToString(a []uint32, delim string) string {
	if len(a) == 0 {
		return ""
	}

	buff := new(bytes.Buffer)
	for i := 0; i < len(a)-1; i++ {
		buff.WriteString(strconv.Itoa(int(a[i])))
		buff.WriteString(delim)
	}
	buff.WriteString(strconv.Itoa(int(a[len(a)-1])))

	return buff.String()
}

func Int32ArrayToStringArray(int32Array []int32) []string {
	ret := make([]string, 0, len(int32Array))
	for _, elem := range int32Array {
		ret = append(ret, strconv.Itoa(int(elem)))
	}
	return ret
}

func Uint32ArrayToStringArray(uint32Array []uint32) []string {
	ret := make([]string, 0, len(uint32Array))
	for _, elem := range uint32Array {
		ret = append(ret, strconv.Itoa(int(elem)))
	}
	return ret
}

// SliceInt32ToString 复用之前的slice结构，比上面性能稍好一些
// slice长度为6时，大约提升30%
func SliceInt32ToString(int32Slice []int32, strSlice *[]string) {
	if cap(*strSlice) < len(int32Slice) {
		s := make([]string, 0, len(int32Slice))
		strSlice = &s
	} else {
		*strSlice = (*strSlice)[:0]
	}

	for i := range int32Slice {
		*strSlice = append(*strSlice, strconv.Itoa(int(int32Slice[i])))
	}
}

// SliceUint32ToString 复用之前的slice结构，比上面性能稍好一些
// slice长度为6时，大约提升30%
func SliceUint32ToString(uint32Slice []uint32, strSlice *[]string) {
	if cap(*strSlice) < len(uint32Slice) {
		s := make([]string, 0, len(uint32Slice))
		strSlice = &s
	} else {
		*strSlice = (*strSlice)[:0]
	}

	for i := range uint32Slice {
		*strSlice = append(*strSlice, strconv.Itoa(int(uint32Slice[i])))
	}
}
