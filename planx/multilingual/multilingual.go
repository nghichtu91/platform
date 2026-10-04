package multilingual

import (
	"encoding/json"
	"strings"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

const (
	Common             = "common" // 默认文本。
	Chinese            = "zh-CN"  // 简体中文。
	TraditionalChinese = "zh-TW"  // 繁体中文（台湾）。
	English            = "en-US"  // 英文（美式）。
	Indonesia          = "id-ID"  // 印尼
	Japan              = "ja-JP"  // 日本
	Vietnam            = "vi-VN"  // 越南
	Thailand           = "th-TH"  // 泰国
)

const IDSFixedPrefix = "IDS_" // IDS固定前缀。

const (
	TypeIDS          = iota // 内容为IDS。
	TypeCommonString        // 内容为普通字符串（如：玩家昵称）。
	TypeMulLanguage         // 内容为多语言。
)

type UnMarshalMultilingual struct {
	Common string `json:"common,omitempty"` // 默认文本。
	ZH_CN  string `json:"zh-CN,omitempty"`  // 内容的简体中文部分。
	ZH_TW  string `json:"zh-TW,omitempty"`  // 内容的繁体中文（台湾）部分。
	EN_US  string `json:"en-US,omitempty"`  // 内容的英文（美式）部分。
	ID_ID  string `json:"id-ID,omitempty"`  // 内容的印尼部分。
	JA_JP  string `json:"ja-JP,omitempty"`  // 内容的日文部分。
	VI_VN  string `json:"vi-VN,omitempty"`  // 内容的越南部分。
	TH_TH  string `json:"th-TH,omitempty"`  // 内容的泰国部分。
}

// 多语言语种。
var multilingual string

// 获取指定的多语言语种。
func GetMultilingual() string {
	return multilingual
}

// 设置指定的多语言语种。
func SetMultilingual(language string) {
	if language == "" {
		multilingual = Chinese
		return
	}
	multilingual = language
}

// 解析并返回多语言json中指定的语言
//
// Param-content: 多语言文本、纯字符串（举例：玩家昵称）或IDS。
//
// Note: 运营部门-李梓烨（2021/10/08 上午） 要求增加All语言类型，当其余语言类型均为空时，使用All类型所含内容。
func UnMarshalByLanguage(language string, content string) string {
	// 如果内容为IDS则不做解析，直接返回。
	if strings.HasPrefix(content, IDSFixedPrefix) {
		return content
	}

	mul := &UnMarshalMultilingual{}
	errUnmarshal := json.Unmarshal([]byte(content), &mul)
	if errUnmarshal != nil {
		// 这里说明content是纯字符串。
		return content
	}

	if mul.ZH_CN == "" && mul.ZH_TW == "" && mul.EN_US == "" && mul.ID_ID == "" && mul.JA_JP == "" && mul.VI_VN == "" && mul.TH_TH == "" {
		return mul.Common
	} else {
		switch language {
		case Chinese:
			return mul.ZH_CN
		case TraditionalChinese:
			return mul.ZH_TW
		case English:
			return mul.EN_US
		case Indonesia:
			return mul.ID_ID
		case Japan:
			return mul.JA_JP
		case Vietnam:
			return mul.VI_VN
		case Thailand:
			return mul.TH_TH
		default:
			tilogs.L().Errorf("UnMarshalByLanguage switch language unknown, language: %v, content: %v", language, content)
			return content
		}
	}
}

func JudgeContentType(content string) int {
	if strings.HasPrefix(content, IDSFixedPrefix) {
		return TypeIDS
	}

	mul := &UnMarshalMultilingual{}
	errUnmarshal := json.Unmarshal([]byte(content), &mul)
	if errUnmarshal != nil {
		// 这里说明content是纯字符串。
		return TypeCommonString
	}

	return TypeMulLanguage
}

// Unmarshal 将多语言文本转换为UnMarshalMultilingual
// 如果内容为IDS或纯字符串，放到Common中并直接返回
// 其他情况返回解码完毕的结构
func Unmarshal(content string) *UnMarshalMultilingual {
	ml := new(UnMarshalMultilingual)
	if strings.HasPrefix(content, IDSFixedPrefix) {
		ml.Common = content
		return ml
	}

	if err := json.Unmarshal([]byte(content), ml); err != nil {
		ml.Common = content
	}

	return ml
}

// TrimBeforeDot 去掉所有内容第一个点之前的文本
func (ml *UnMarshalMultilingual) TrimBeforeDot() {
	ml.Common = trimSrvIndex(ml.Common)
	ml.ZH_CN = trimSrvIndex(ml.ZH_CN)
	ml.ZH_TW = trimSrvIndex(ml.ZH_TW)
	ml.EN_US = trimSrvIndex(ml.EN_US)
	ml.ID_ID = trimSrvIndex(ml.ID_ID)
	ml.JA_JP = trimSrvIndex(ml.JA_JP)
	ml.VI_VN = trimSrvIndex(ml.VI_VN)
	ml.TH_TH = trimSrvIndex(ml.TH_TH)
}

func trimSrvIndex(name string) string {
	if name == "" {
		return ""
	}

	idx := strings.IndexByte(name, '.')
	if idx != -1 {
		// 最后一个字符是.
		if idx == len(name)-1 {
			return ""
		}
		return name[idx+1:]
	}
	return name
}

// RemoveJsonSrvIdx 将类似 {"common": "1.S1-桃源村"} 的服务器名称转为 {"common": "S1-桃源村"}
// 只处理正式的json格式
// 非json格式返回原值
func RemoveJsonSrvIdx(content string) string {
	ml := new(UnMarshalMultilingual)
	if err := json.Unmarshal([]byte(content), ml); err != nil {
		return content
	}

	ml.TrimBeforeDot()

	bs, _ := json.Marshal(ml)
	return string(bs)
}
