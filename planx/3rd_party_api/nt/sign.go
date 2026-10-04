package nt

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

// Sign NT平台验证签名方法
// 文档链接: http://confluence.taiyouxi.net/pages/viewpage.action?pageId=35987917
func Sign(secretKey string, in interface{}) string {
	tmp, err := json.Marshal(in)
	if err != nil {
		tilogs.L().Errorf("nt sign marshal err: %s", err.Error())
		return ""
	}

	data := make(map[string]interface{}, 16)
	// 用number避免json将数字改为float64
	d := json.NewDecoder(bytes.NewReader(tmp))
	d.UseNumber()
	d.Decode(&data)

	// 获得key并排序
	keys := make([]string, 0, len(data)-1)
	for k := range data {
		if k == KeySign {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	values := make([]string, 0, len(data))
	for _, k := range keys {
		if data[k] == "" {
			continue
		}
		switch reflect.TypeOf(data[k]).Kind() {
		case reflect.Slice, reflect.Map, reflect.Struct: // 展开嵌套结构为json
			v, _ := json.Marshal(data[k])
			values = append(values, fmt.Sprintf("%s=%v", k, string(v)))
		default:
			values = append(values, fmt.Sprintf("%s=%v", k, data[k]))
		}
	}

	h := md5.New()
	h.Write([]byte(strings.Join(values, "&") + secretKey))

	return hex.EncodeToString(h.Sum(nil))
}
