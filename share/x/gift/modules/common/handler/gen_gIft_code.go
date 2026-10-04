package handler

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/nghichtu91/platform/share/planx/rand_pool"
	"github.com/nghichtu91/platform/share/x/gift/config"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
)

// GenGiftCode 生成礼包码
//
// 1位		1位		2位			7位				5位			1位		2位		1位
//
// 1		0~7		0~99		0~999.9999万	0~99999		0~9		0~99	0~9
//
// 固定值1	随机位	项目代号位	生成序号位		批次位		随机		组号位	随机
//
// 首位固定值1，保证了字符码长为14，首随机为（0~7）因为uint64最大值前三位为184。
//
// Param-batchID: 批次号; Param-groupID: 组号; Param-count: 本次生成数量; Param-ownedCount: 已生成的数量。
//
// Return-[]string: 生成的兑换码; Return-error: 内部显示异常信息; Return-string: 反馈异常信息。
func GenGiftCode(projectID, batchID, groupID, count, ownedCount uint64) ([]string, error, string) {
	// 判断生成兑换码的各组成部分是否超出上限。
	if errLimit, errLimitMsg := judgeGenCodeLimit(projectID, batchID, groupID, count); errLimit != nil {
		return nil, errLimit, errLimitMsg
	}

	// 生成指定数量和信息的兑换码
	result := make([]string, 0, count)
	rand.Seed(time.Now().Unix())
	for index := ownedCount; index < ownedCount+count; index++ {
		if charCoding, errCoding, errCodingMsg := num2CharCoding(genOneGiftCode(projectID, batchID, groupID, index)); errCoding != nil {
			return nil, errCoding, errCodingMsg
		} else {
			result = append(result, charCoding)
		}
	}
	return result, nil, errs.Success
}

// genOneGiftCode 生成一条礼包兑换码。
//
// Param-batchID: 批次号; Param-groupID: 组号; Param-index: 当前批次组生成的第index个兑换码。
//
// Return-string: 生成的由定长十四位字符组成的兑换码字符串。
func genOneGiftCode(projectID, batchID, groupID, index uint64) uint64 {
	var numCoding uint64

	// 1 固定值 （1位）（【防止溢出】【固定码长】）
	numCoding = numCoding*config.CodeInherentCoding + config.CodeInherent

	// 4 随机生成（1位）（【防止爆破】）
	numCoding = numCoding*config.CodeRandomCoding + genRandomCoding(config.CodeRandomFirstRange)

	// 2 项目代号（2位）
	numCoding = numCoding*config.CodeProjectCoding + projectID

	// 3 对应序号（7位）
	numCoding = numCoding*config.CodeIndexCoding + config.CodeIndexCoding - index - 1

	// 5 对应批次（5位）
	numCoding = numCoding*config.CodeBatchCoding + batchID

	// 6 随机生成（1位）（【防止爆破】）
	numCoding = numCoding*config.CodeRandomCoding + genRandomCoding(config.CodeRandomCommonRange)

	// 7 对应组号（2位）
	numCoding = numCoding*config.CodeGroupCoding + groupID

	// 8 随机生成（1位）（【防止爆破】）
	numCoding = numCoding*config.CodeRandomCoding + genRandomCoding(config.CodeRandomCommonRange)

	return numCoding
}

// judgeGenCodeLimit 判断生成兑换码的要求是否超出上限。
//
// Param-batchID: 批次号; Param-groupID: 组号; Param-count: 生成数量。
//
// Return-error: 判断结果，用于内部显示; Return-string: 判断结果，用于反馈显示。
func judgeGenCodeLimit(projectID, batchID, groupID, count uint64) (error, string) {
	// 判断项目上限。
	if projectID > config.CodeMaxProject {
		return fmt.Errorf("over projectID limit：Max: %v, Req: %v", config.CodeMaxProject, projectID), errs.ErrCodeOverLimit
	}
	// 判断批次上限。
	if batchID > config.CodeMaxBatchNum {
		return fmt.Errorf("over batchID limit： Max：%v, Req: %v", config.CodeMaxBatchNum, batchID), errs.ErrCodeOverLimit
	}
	// 判断组上限。
	if groupID > config.CodeMaxGroupNum {
		return fmt.Errorf("over groupID limit： Max： %v， Req： %v", config.CodeMaxGroupNum, groupID), errs.ErrCodeOverLimit
	}
	// 判断数量上限。
	if count > config.CodeMaxGenNum {
		return fmt.Errorf("over count limit：Max：%v, Req: %v", config.CodeMaxGenNum, count), errs.ErrCodeOverLimit
	}

	return nil, errs.Success
}

// genRandomCoding 生成随机部分的数值。
//
// Param-randomRange：随机生成的上限（不包含）。
//
// Return-uint64: 生成的[0~randomRange-1]随机数。
func genRandomCoding(randomRange int32) uint64 {
	return uint64(rand_pool.Int31n(randomRange))
}

// num2CharCoding 数字兑换码转换为字符串兑换码。
//
// Param-numCoding: 以数字表示的兑换码。
//
// Return-string: 以字符串表示的兑换码; Return-error: 内部显示异常信息; Return-string: 反馈异常信息。
func num2CharCoding(numCoding uint64) (string, error, string) {
	charBase := uint64(len(config.Num2Char))
	if charBase == 0 {
		return "", fmt.Errorf("num2CharCoding charBase is Empty"), errs.ErrCodeEmptyCharBase
	}

	charCoding := ""

	for numCoding > 0 {
		currentNum := numCoding % charBase
		numCoding = numCoding / charBase

		charCoding = string(config.Num2Char[currentNum]) + charCoding
	}

	return charCoding, nil, errs.Success
}

// GetBatchGroupIDByCharCoding 通过生成字符码获取批次组号ID
//
// Param-charCoding: 以字符串表示的生成兑换码。
//
// Return-int: 批次号; Return-int: 组号。
func GetBatchGroupIDByCharCoding(charCoding string) (int, int, error, string) {
	numCoding, err, errMsg := char2NumCoding(charCoding)
	if err != nil {
		return -1, -1, err, errMsg
	}

	// 获取生成兑换码的批次号和组号。
	batchID := numCoding / config.CodeRandomCoding / config.CodeGroupCoding / config.CodeRandomCoding % config.CodeBatchCoding
	groupID := numCoding / config.CodeRandomCoding % config.CodeGroupCoding

	return int(batchID), int(groupID), nil, errs.Success
}

// char2NumCoding 字符串生成兑换码转换为数字兑换码。
//
// Param-charCoding: 以字符串表示的生成兑换码。
//
// Return-uint64: 以数字表示的兑换码; Return-error: 内部显示异常信息; Return-string: 反馈异常信息。
func char2NumCoding(charCoding string) (uint64, error, string) {
	charBase := uint64(len(config.Num2Char))
	if charBase == 0 {
		return 0, fmt.Errorf("char2NumCoding charBase is Empty"), errs.ErrCodeEmptyCharBase
	}

	numCoding := uint64(1)

	// 自高位到低位还原数字码。
	for index := 0; index < len(charCoding); index++ {
		if num, ok := config.Char2Num[string(charCoding[index])]; ok {
			if index == 0 {
				numCoding = num
			} else {
				numCoding = numCoding*charBase + num
			}
		} else {
			return 0, fmt.Errorf("cannot distinguish char %v in %v", string(charCoding[index]), charCoding), errs.ErrCodeCanNotDistinguish
		}
	}

	return numCoding, nil, errs.Success
}
