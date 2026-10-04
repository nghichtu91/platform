package math

// Round2N 获取最接近且大于等于num的2的n次幂
// e.g.  15 => 16, 33 => 64
// 暂不处理负数
func Round2N(num int) int {
	if num < 0 {
		return 0
	}
	if num&(num-1) != 0 {
		for num&(num-1) != 0 {
			num &= num - 1
		}
		num = num << 1
	}
	return num
}
