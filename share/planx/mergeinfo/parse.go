package mergeinfo

import (
	"errors"
	"strconv"
	"strings"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var (
	ErrInvalidRangeString = errors.New("invalid range config param string")
)

// ParseRangeStr 解析区间类型字符串的通用方法
// 将类似 "1001,1003-1005,1007-2010" 的字符串解析成区间slice
// 格式需求：允许额外的单双引号，分隔符必须为半角逗号，允许中间有空格，必须按升序排列且无重合区间
// 参数为空时不报错，返回nil
func ParseRangeStr(param string) ([][]int, error) {
	str := strings.TrimSpace(param)
	// 去掉运维可能带入的单双引号
	str = strings.Replace(str, "\"", "", -1)
	str = strings.Replace(str, "'", "", -1)
	if len(str) == 0 {
		return nil, nil
	}

	// 进行具体解析并校验
	parts := strings.Split(str, ",")
	results := make([][]int, 0, len(parts))
	lastE := 0

	for _, part := range parts {
		if strings.Contains(part, "-") {
			sections := strings.Split(part, "-")
			if len(sections) != 2 {
				tilogs.L().Errorf("invalid range config param %s", param)
				return nil, ErrInvalidRangeString
			}
			s, err1 := strconv.Atoi(strings.TrimSpace(sections[0]))
			e, err2 := strconv.Atoi(strings.TrimSpace(sections[1]))

			// 校验
			if err1 != nil || err2 != nil || e < 0 || s < 0 || e < s || s <= lastE {
				tilogs.L().Errorf("invalid range config param %s", param)
				return nil, ErrInvalidRangeString
			}
			lastE = e

			results = append(results, []int{s, e})
		} else {
			sid, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || sid < 0 || sid <= lastE {
				tilogs.L().Errorf("invalid range config param %s", param)
				return nil, ErrInvalidRangeString
			}

			lastE = sid
			results = append(results, []int{sid, sid})
		}
	}

	return results, nil
}
