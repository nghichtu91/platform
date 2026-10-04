package arrayhelper

import (
	"math/rand"
	"strconv"
	"time"
)

func ContainsInt(array []int, des int) bool {
	for _, id := range array {
		if id == des {
			return true
		}
	}
	return false
}

func ContainsInt32(array []int32, des int32) bool {
	for _, id := range array {
		if id == des {
			return true
		}
	}
	return false
}

func ConvertInt32Array(array []int) []int32 {
	ret := make([]int32, len(array))
	for i := range array {
		ret[i] = int32(array[i])
	}
	return ret
}

func ContainsString(array []string, des string) bool {
	for _, id := range array {
		if id == des {
			return true
		}
	}
	return false
}
func RemoveString(array []string, des string) []string {
	for i := 0; i < len(array); i++ {
		if array[i] == des {
			array = append(array[:i], array[i+1:]...)
			break
		}
	}
	return array
}

func BuildIntArrayFromInt32(array []int32) []int {
	ret := make([]int, len(array))
	for idx := range array {
		ret[idx] = int(array[idx])
	}
	return ret
}

func BuildInt32ArrayFromUint32(array []uint32) []int32 {
	ret := make([]int32, len(array))
	for idx := range array {
		ret[idx] = int32(array[idx])
	}
	return ret
}

func BuildUint32ArrayFromInt32(array []int32) []uint32 {
	ret := make([]uint32, len(array))
	for idx := range array {
		ret[idx] = uint32(array[idx])
	}
	return ret
}

func BuildUint32ArrayFromInt(array []int) []uint32 {
	ret := make([]uint32, len(array))
	for idx := range array {
		ret[idx] = uint32(array[idx])
	}
	return ret
}

func BuildFloatArrayFromString(array []string) []float64 {
	ret := make([]float64, len(array))
	for idx := range array {
		v, _ := strconv.ParseFloat(array[idx], 64)
		ret[idx] = v
	}
	return ret
}

func BuildUint32ArrayFromString(array []string) []uint32 {
	ret := make([]uint32, len(array))
	for idx := range array {
		v, err := strconv.Atoi(array[idx])
		if err != nil {
			return []uint32{}
		}
		ret[idx] = uint32(v)
	}
	return ret
}

// IsSame 判断两个int32数组是否相同
func IsSame(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func ContainsUint(array []uint, des uint) bool {
	for _, id := range array {
		if id == des {
			return true
		}
	}
	return false
}

func ContainsUint32(array []uint32, des uint32) bool {
	for _, id := range array {
		if id == des {
			return true
		}
	}
	return false
}

func ContainsInt64(array []int64, des int64) bool {
	for _, id := range array {
		if id == des {
			return true
		}
	}
	return false
}

// ShuffleByte 打乱array，并返回新切片，会改变array的内容
func ShuffleByte(array []byte) []byte {
	newArray := make([]byte, 0)
	rs := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
	Loop:
		length := len(array)

		if length == 0 {
			break
		}
		for i := 0; i <= length; i++ {
			p := rs.Intn(length)
			newArray = append(newArray, array[p])
			array = append(array[0:p], array[p+1:]...)
			goto Loop
		}
	}
	return newArray
}

// GenUint32ShuffleCopy 生成一个重排后的uint32 slice
// 不会修改原slice
func GenUint32ShuffleCopy(in []uint32) []uint32 {
	out := make([]uint32, len(in))
	copy(out, in)
	rand.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}
