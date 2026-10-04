package util

import (
	"errors"
	"math"
)

var Zero ZeroType

var _ = Zero

var (
	ErrOverflow = errors.New("param overflow")
	ErrBatch    = errors.New("batch not valid")
)

type ZeroType struct{}

func Iter(n int) []ZeroType {
	if n <= 0 {
		return nil
	}
	// zero memory alloc
	return make([]ZeroType, n)
}

// Batch 将 ln 分成 batch 组执行, end - start = groupCnt, 也就是这一组有多少个.
// 返回的区间是[start, end), 也就是左闭右开区间
func Batch(ln, batch int, cb func(start, end int) bool) error {
	if ln <= 0 || cb == nil {
		return nil
	}
	if batch <= 0 {
		return ErrBatch
	}
	if batch > ln {
		batch = ln
	}
	// 如果将 ln 分成 batch 组, 则每一组需要有多少个
	minNum := ln / batch
	remain := ln % batch

	for start := 0; start < ln; {
		end := start + minNum
		if remain > 0 {
			end++
			remain--
		}
		if !cb(start, end) {
			break
		}
		start = end
	}
	return nil
}

// BatchRange  ln: 总长度, cnt: 一批多少人
// 返回的区间是[start, end), 也就是左闭右开区间
func BatchRange(ln, cnt int, cb func(start, end int) bool) error {
	if ln <= 0 || cb == nil {
		return nil
	}
	if err := batchCheck(ln, cnt); err != nil {
		return err
	}
	var start = 0
	for start < ln {
		end := start + cnt
		if end > ln {
			end = ln
		}
		if !cb(start, end) {
			break
		}
		start = end
	}
	return nil
}

func emptyIterator(func(int, int) bool) {}

func batchCheck(ln, groupCnt int) error {
	if groupCnt <= 0 {
		return ErrBatch
	}
	if ln > math.MaxInt32 || groupCnt > math.MaxInt32 {
		return ErrOverflow
	}
	return nil
}

// BatchIterator ln: 总长度, groupCnt: 一批多少人
// 返回分了多少组, 以及对应的迭代器. 或者错误
func BatchIterator(ln, groupCnt int) (batchCnt int, iterator func(func(int, int) bool), err error) {
	if ln == 0 {
		return 0, emptyIterator, nil
	}
	if err = batchCheck(ln, groupCnt); err != nil {
		batchCnt = -1
		return
	}
	return Upper(int32(ln), int32(groupCnt)), func(f func(start int, end int) bool) {
		_ = BatchRange(ln, groupCnt, f)
	}, nil
}

// Upper 将 ln 按照每组 groupCnt 个分批, 最少需要多少批. 向上取整
func Upper(ln, groupCnt int32) int {
	if ln < 0 {
		return 0
	}
	if groupCnt <= 1 {
		return int(ln)
	}
	// 限定参数的形式避免溢出
	return (int(ln) + int(groupCnt) - 1) / int(groupCnt)
}
