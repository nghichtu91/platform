package controllers

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

/*
	用于Auth压测、机器人或Debug
*/

var (
	ErrTarInfoNil   = errors.New("target info is nil")
	ErrSrcToMapFail = errors.New("source to map failed")
)

// SdkDebugLoginInfo
type SdkDebugLoginInfo struct {
	Enc *SdkLoginInfo
	Dec *SdkLoginInfo
}

var (
	// encode func
	encFunc = func(in string) string {
		return secure.DefaultEncode.Encode64ForNet([]byte(in))
	}

	// decode func
	decFunc = func(in string) string {
		d, err := secure.DefaultEncode.Decode64FromNet(in)
		if err != nil {
			tilogs.L().Errorf("SdkDebugLoginInfo Decode failed, err %v", err)
			return ""
		}
		return string(d)
	}
)

// toMap 将结构转为map
func (s *SdkLoginInfo) toMap() map[string]string {
	if s == nil {
		return nil
	}

	tmpRaw, err := json.Marshal(s)
	if err != nil {
		return nil
	}

	tmp := make(map[string]string, 16)
	err = json.Unmarshal(tmpRaw, &tmp)
	if err != nil {
		return nil
	}
	return tmp
}

// trans 将src的字段进行encode/decode
func trans(src, tar *SdkLoginInfo, codeFunc func(in string) string) error {
	if tar == nil {
		return ErrTarInfoNil
	}

	tmp := src.toMap()
	if tmp == nil {
		return ErrSrcToMapFail
	}

	for k, v := range tmp {
		if v == "" {
			continue
		}
		if codeFunc != nil {
			if cv := codeFunc(tmp[k]); cv != "" {
				tmp[k] = cv
			}
		}
	}

	tmpRaw, err := json.Marshal(tmp)
	if err != nil {
		return err
	}

	return json.Unmarshal(tmpRaw, tar)
}

// Encode
// 将Dec里的内容encode后写入enc
// Debug不追求性能，用json处理中间过程
func (info *SdkDebugLoginInfo) Encode() {
	err := trans(info.Dec, info.Enc, encFunc)
	if err != nil {
		tilogs.L().Errorf("SdkDebugLoginInfo Encode failed, err %v", err)
	}
}

// Decode
// 将Enc里的内容decode后写入Dec
// Debug不追求性能，用json处理中间过程
func (info *SdkDebugLoginInfo) Decode() {
	err := trans(info.Enc, info.Dec, decFunc)
	if err != nil {
		tilogs.L().Errorf("SdkDebugLoginInfo Decode failed, err %v", err)
	}
}

// GenParams 生成http请求params用于调试
// 懒得排序了，反正都一样
func (s *SdkLoginInfo) GenParams(codeFunc func(in string) string) string {
	tmp := s.toMap()
	if tmp == nil {
		return ""
	}

	params := make([]string, 0, len(tmp))

	for k, v := range tmp {
		if v != "" {
			if codeFunc != nil {
				v = codeFunc(v)
			}
			field, ok := reflect.TypeOf(s).Elem().FieldByName(k)
			if ok {
				if tag, ok := field.Tag.Lookup("form"); ok && k != tag {
					k = tag
				}
			}
			params = append(params, k+"="+v)
		}
	}

	return strings.Join(params, "&")
}
