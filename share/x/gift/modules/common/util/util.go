package util

import (
	"bytes"
	"fmt"
	"strings"
)

// GetBatchGroupFileName 获取批次、组号的反馈文件名
func GetBatchGroupFileName(batchID, groupID int) string {
	return fmt.Sprintf("%d_%d.txt", batchID, groupID)
}

// Codes2bytes 将兑换码集合转换为字节码。
func Codes2bytes(codes []string) []byte {
	var resultBytes []byte
	buffer := bytes.NewBuffer(resultBytes)
	for _, code := range codes {
		buffer.WriteString(code)
		buffer.WriteString("\r\n")
	}
	return buffer.Bytes()
}

// Bytes2Codes 将字节码转换为兑换码合集。
func Bytes2Codes(content []byte) []string {
	return strings.Split(string(content), "\r\n")
}

// MergeSliceNoRepetition 以元素不重复的方式合并int类型切片。
func MergeSliceNoRepetition(originSlice, mergedSlice []int) []int {
	result := originSlice
	for _, mergedCode := range mergedSlice {
		find := false
		for _, originCode := range result {
			if mergedCode == originCode {
				find = true
				break
			}
		}

		if find {
			continue
		} else {
			result = append(result, mergedCode)
		}
	}

	return result
}
