package util

// Int32Merge2Int64 将两个int32拼接成一个int64
func Int32Merge2Int64(high, low int32) int64 {
	return int64(high)<<32 + int64(low)
}

// Int64Split 将int64按高低位拆成两个int32
func Int64Split(in int64) (int32, int32) {
	return int32(in >> 32), int32(in)
}

// UInt32Merge2UInt64 将两个uint32拼接成一个uint64
func UInt32Merge2UInt64(high, low uint32) uint64 {
	return uint64(high)<<32 + uint64(low)
}

// UInt64Split 将int64按高低位拆成两个int32
func UInt64Split(in uint64) (uint32, uint32) {
	return uint32(in >> 32), uint32(in)
}

// UInt32Merge3 将三个uint32按指定偏移拼接成一个uint64
// uint32数字不能超过 1<<24，即 16777216
func UInt32Merge3(high, mid, low uint32) uint64 {
	return uint64(high)<<48 + uint64(mid)<<24 + uint64(low)
}

// UInt64Split3 将int64按指定偏移拆成三个uint32
// 需要自行注意偏移量是否合理
func UInt64Split3(in uint64) (high, mid, low uint32) {
	high = uint32(in >> 48)
	mid = uint32((in - uint64(high)<<48) >> 24)
	low = uint32(in - uint64(high)<<48 - uint64((mid)<<24))
	return
}
