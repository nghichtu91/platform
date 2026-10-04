package controllers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"encoding/json"

	"sort"

	"bytes"

	"crypto/md5"

	"encoding/base64"

	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

type HeroSdk struct {
	platformType string
}

func (h *HeroSdk) VerifyLogin(loginInfo *SdkLoginInfo) (SdkUserInterface, error) {
	timestamp := fmt.Sprint(time.Now().Unix())
	dataStr := h.data(loginInfo.uidDecoded, loginInfo.tokenDecoded)
	sign := h.sign(dataStr, timestamp)
	var r http.Request
	r.ParseForm()
	r.Form.Add("pcode", h.productId())
	r.Form.Add("timestamp", timestamp)
	r.Form.Add("data", dataStr)
	r.Form.Add("sign", sign)
	body := strings.NewReader(r.Form.Encode())

	clt := http.Client{
		Timeout: 3 * time.Second,
	}
	req, err := http.NewRequest("POST", h.url(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := clt.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	userInfo := &HeroSdkUser{}
	err = json.Unmarshal(content, userInfo)
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

func (*HeroSdk) Flag() string {
	return "hero.com"
}

func (*HeroSdk) VerifyPay() {
}

func (h *HeroSdk) url() string {
	if h.platformType == consts.PlatformTypeAndroid {
		return config.Cfg.HeroSdkAndroidCfg.Url + config.Cfg.HeroSdkAndroidCfg.UrlLogin
	} else {
		return config.Cfg.HeroSdkIosCfg.Url + config.Cfg.HeroSdkIosCfg.UrlLogin
	}
}

func (h *HeroSdk) productId() string {
	if h.platformType == consts.PlatformTypeAndroid {
		return config.Cfg.HeroSdkAndroidCfg.ProductId
	} else {
		return config.Cfg.HeroSdkIosCfg.ProductId
	}
}

func (h *HeroSdk) appKey() string {
	if h.platformType == consts.PlatformTypeAndroid {
		return config.Cfg.HeroSdkAndroidCfg.AppKey
	} else {
		return config.Cfg.HeroSdkIosCfg.AppKey
	}
}

func (h *HeroSdk) data(uid, token string) string {
	return makeData(map[string]interface{}{
		"cUid":        uid,
		"accessToken": token,
	})
}

func (h *HeroSdk) sign(data string, timestamp string) string {
	return makeSign(h.appKey(), map[string]string{
		"pcode":     h.productId(),
		"data":      data,
		"timestamp": timestamp,
	})
}

func makeData(dataMap map[string]interface{}) string {
	dataBytes, err := json.Marshal(dataMap)
	if err != nil {
		tilogs.L().Errorf("json marshal error %v", err)
		return ""
	}

	data64 := base64.StdEncoding.EncodeToString(dataBytes)

	if len(data64) > 51 {
		return swapIndex(data64, 1, 33, 10, 42, 18, 50, 19, 51)
	}

	return data64
}

func swapIndex(src string, indexs ...int) string {
	srcBytes := append([]byte{}, []byte(src)...)
	for i := 1; i < len(indexs); i += 2 {
		srcBytes[indexs[i-1]], srcBytes[indexs[i]] = srcBytes[indexs[i]], srcBytes[indexs[i-1]]
	}
	return string(srcBytes)
}

type Entry struct {
	Key   string
	Value string
}

func makeSign(appKey string, dataMap map[string]string) string {
	entryList := make([]Entry, 0, len(dataMap))
	for k, v := range dataMap {
		entryList = append(entryList, Entry{Key: k, Value: v})
	}
	sort.Slice(entryList, func(i, j int) bool {
		return entryList[i].Key < entryList[j].Key
	})
	buf := bytes.NewBuffer([]byte{})
	for _, entry := range entryList {
		buf.WriteString(entry.Key)
		buf.WriteString("=")
		buf.WriteString(entry.Value)
		buf.WriteString("&")
	}
	buf.WriteString(appKey)
	return fmt.Sprintf("%x", md5.Sum(buf.Bytes()))
}

type HeroSdkUser struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Uid  string `json:"cUid"`
	Name string `json:"cName"`
}

func (hs *HeroSdkUser) IsValid() bool {
	return hs.Uid != "" && hs.Code == 0
}

func (hs *HeroSdkUser) UserId() string {
	return hs.Uid
}

func (hs *HeroSdkUser) GetAccountName() string {
	return hs.Name
}
