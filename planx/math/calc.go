package math

import (
	"math"
)

// SafeAddUint64 old+add,
// v 表示返回值, 如果溢出了(>math.MaxUint64)则返回backup.
// overflow 表示是否溢出
func SafeAddUint64(old, add, backup uint64) (v uint64, overflow bool) {
	if math.MaxUint64-old < add {
		return backup, true
	}
	return old + add, false
}

// SafeSubUint64 old-add
// v 表示返回值,  如果溢出了(<0)则返回backup.
// overflow 表示是否溢出
func SafeSubUint64(old, sub, backup uint64) (v uint64, overflow bool) {
	if old < sub {
		return backup, true
	}
	return old - sub, false
}

// AddUint64 a+nums, 默认溢出返回 math.MaxUint64
func AddUint64(a uint64, nums ...uint64) uint64 {
	if len(nums) == 0 {
		return a
	}

	if len(nums) == 1 {
		a, _ = SafeAddUint64(a, nums[0], math.MaxUint64)
		return a
	}

	for _, b := range nums {
		a, _ = SafeAddUint64(a, b, math.MaxUint64)
		if a == math.MaxUint64 {
			return a
		}
	}
	return a
}

// SubUint64 a-nums, 默认溢出返回 0
func SubUint64(a uint64, nums ...uint64) uint64 {
	if len(nums) == 0 {
		return a
	}

	if len(nums) == 1 {
		a, _ = SafeSubUint64(a, nums[0], 0)
		return a
	}

	for _, b := range nums {
		a, _ = SafeSubUint64(a, b, 0)
		if a == 0 {
			return a
		}
	}
	return a
}

// SafeAddUint32 old+add,
// v 表示返回值, 如果溢出了(>math.MaxUint32)则返回backup.
// overflow 表示是否溢出
func SafeAddUint32(old, add, backup uint32) (v uint32, overflow bool) {
	if math.MaxUint32-old < add {
		return backup, true
	}
	return old + add, false
}

// SafeSubUint32 old-add
// v 表示返回值,  如果溢出了(<0)则返回backup.
// overflow 表示是否溢出
func SafeSubUint32(old, sub, backup uint32) (v uint32, overflow bool) {
	if old < sub {
		return backup, true
	}
	return old - sub, false
}

// AddUint32 a+nums, 默认溢出返回 math.MaxUint32
func AddUint32(a uint32, nums ...uint32) uint32 {
	if len(nums) == 0 {
		return a
	}
	if len(nums) == 1 {
		a, _ = SafeAddUint32(a, nums[0], math.MaxUint32)
		return a
	}
	for _, b := range nums {
		a, _ = SafeAddUint32(a, b, math.MaxUint32)
		if a == math.MaxUint32 {
			return a
		}
	}
	return a
}

// SubUint32 a-nums, 默认溢出返回 0
func SubUint32(a uint32, nums ...uint32) uint32 {
	if len(nums) == 0 {
		return a
	}

	if len(nums) == 1 {
		a, _ = SafeSubUint32(a, nums[0], 0)
		return a
	}

	for _, b := range nums {
		a, _ = SafeSubUint32(a, b, 0)
		if a == 0 {
			return a
		}
	}
	return a
}

// SafeAddUint16 old+add,
// v 表示返回值, 如果溢出了(>math.MaxUint16)则返回backup.
// overflow 表示是否溢出
func SafeAddUint16(old, add, backup uint16) (v uint16, overflow bool) {
	if math.MaxUint16-old < add {
		return backup, true
	}
	return old + add, false
}

// SafeSubUint16 old-add
// v 表示返回值,  如果溢出了(<0)则返回backup.
// overflow 表示是否溢出
func SafeSubUint16(old, sub, backup uint16) (v uint16, overflow bool) {
	if old < sub {
		return backup, true
	}
	return old - sub, false
}

// AddUint16 a+nums, 默认溢出返回 math.MaxUint16
func AddUint16(a uint16, nums ...uint16) uint16 {
	if len(nums) == 0 {
		return a
	}
	if len(nums) == 1 {
		a, _ = SafeAddUint16(a, nums[0], math.MaxUint16)
		return a
	}
	for _, b := range nums {
		a, _ = SafeAddUint16(a, b, math.MaxUint16)
		if a == math.MaxUint16 {
			return a
		}
	}
	return a
}

// SubUint16 a-nums, 默认溢出返回 0
func SubUint16(a uint16, nums ...uint16) uint16 {
	if len(nums) == 0 {
		return a
	}

	if len(nums) == 1 {
		a, _ = SafeSubUint16(a, nums[0], 0)
		return a
	}

	for _, b := range nums {
		a, _ = SafeSubUint16(a, b, 0)
		if a == 0 {
			return a
		}
	}
	return a
}
