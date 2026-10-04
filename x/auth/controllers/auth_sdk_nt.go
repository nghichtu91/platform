package controllers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sony/sonyflake"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/x/auth/models"

	"github.com/nghichtu91/platform/share/planx/ntsdk"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var reqIdGenerator *sonyflake.Sonyflake

func InitReqIdGenerator(serverId uint) bool {
	if uint32(serverId) > uint32(math.MaxUint16) {
		tilogs.L().Errorf("InitReqIdGenerator serverId %d > MaxUint16", serverId)
		return false
	}
	var st sonyflake.Settings
	st.MachineID = func() (uint16, error) {
		return uint16(serverId), nil
	}
	reqIdGenerator = sonyflake.NewSonyflake(st)
	return true
}

type NtSdk struct {
}

func (h *NtSdk) VerifyLogin(loginInfo *SdkLoginInfo) (SdkUserInterface, error) {
	baseData := make(map[string]string, 3)
	baseData["appId"] = ntsdk.GetNtSdkAppId()
	baseData["uid"] = loginInfo.uidDecoded
	baseData["token"] = loginInfo.tokenDecoded

	response, err := NtSdkPost(baseData, ntsdk.GetSecretKey(), ntsdk.GetLoginUrl(loginInfo.environmentDecoded), ntsdk.GetContentType())
	if err != nil {
		return nil, fmt.Errorf("NtSdkPost response is error, code = %d, reason = %s, err %v",
			response.Code, response.Reason, err)
	}

	userInfo := &NtSdkUser{
		Code: response.Code,
		Uid:  loginInfo.uidDecoded,
	}
	return userInfo, nil
}

func (*NtSdk) Flag() string {
	return "taihe.com"
}

func (*NtSdk) VerifyPay() {
}

type NtSdkResp struct {
	Code   int    `json:"code"`
	Reason string `json:"reason"`
}

type NtSdkUser struct {
	Code int    `json:"code"`
	Uid  string `json:"cUid"`
}

func (hs *NtSdkUser) IsValid() bool {
	return hs.Uid != "" && hs.Code == 0
}

func (hs *NtSdkUser) UserId() string {
	return hs.Uid
}

func (hs *NtSdkUser) GetAccountName() string {
	return ""
}

func DictionaryUpOrder(sortData map[string]string) string {
	// 去除value为空的键值对
	var keyList []string
	for k, v := range sortData {
		if v == "" {
			continue
		}

		keyList = append(keyList, k)
	}

	sort.Strings(keyList)
	var unsignedData string
	for i, k := range keyList {
		unsignedData = unsignedData + k + "=" + sortData[k]
		if i < len(keyList)-1 {
			unsignedData = unsignedData + "&"
		}
	}

	return unsignedData
}

// NtSdk通用Post接口，gamexshared中推送大区及服务器信息也会调用此接口
// param: baseData 除sign的表单
func NtSdkPost(baseData map[string]string, secretKey string, ipAddress string, contentType string) (response NtSdkResp, err error) {
	unsigned := DictionaryUpOrder(baseData)
	unsigned = unsigned + secretKey
	unsignedBytes := []byte(unsigned)
	signed := fmt.Sprintf("%x", md5.Sum(unsignedBytes))

	var r http.Request
	r.ParseForm()
	for k, v := range baseData {
		r.Form.Add(k, v)
	}

	r.Form.Add("sign", signed)
	body := strings.NewReader(r.Form.Encode())

	clt := http.Client{
		Timeout: 3 * time.Second,
	}

	reqId, err := reqIdGenerator.NextID()
	if err != nil {
		tilogs.L().Errorf("NtSdkPost reqIdGenerator.NextID err %v", err)
	}
	// 增加日志，方便查询错误
	tilogs.L().Infof("NtSdk Post Log:{ baseData = %v, secretKey = %s, ipAddress = %s, sign = %s, reqId = %d}",
		baseData, secretKey, ipAddress, signed, reqId)

	ipAddress = fmt.Sprintf("%s?requestId=%d", ipAddress, reqId)
	req, err := http.NewRequest("POST", ipAddress, body)
	if err != nil {
		tilogs.L().Errorf(fmt.Sprintf("NtSdkPost new request is error, %v, reqId %d", err, reqId))
		return response, err
	}

	req.Header.Add("Content-Type", contentType)
	resp, err := clt.Do(req)
	if err != nil {
		tilogs.L().Errorf(fmt.Sprintf("NtSdkPost post req is error, %v, reqId %d", err, reqId))
		return response, err
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		tilogs.L().Errorf(fmt.Sprintf("NtSdkPost readAll is error, %v, reqId %d", err, reqId))
		return response, err
	}

	err = json.Unmarshal(content, &response)
	if err != nil {
		tilogs.L().Errorf(fmt.Sprintf("NtSdkPost Unmarshal is error, %v, reqId %d", err, reqId))
		return response, err
	}

	if response.Code == 0 { // 中台返回成功
		return response, nil
	}

	return response, fmt.Errorf("NtSdkPost response is error, reason: %s, reqId %d", response.Reason, reqId)
}

// QueryNTID 从中台请求的订单号返回中台ID
// 订单号格式：acid:时间戳
func (dic *AuthController) QueryNTID(c *gin.Context) {
	oid := c.Param("id")

	idx := strings.LastIndex(oid, ":")
	if idx == -1 {
		tilogs.L().Errorf("QueryNTID invalid order id %s", oid)
		return
	}

	acc, err := db.ParseAccount(oid[:idx])
	if err != nil {
		tilogs.L().Errorf("QueryNTID invalid order id %s", oid)
		return
	}

	sdkDeviceID, err := models.GetSdkDeviceIdByUid(acc.UserId.String())
	if err != nil {
		tilogs.L().Errorf("QueryNTID get device id by uid %s from db failed, err %s", acc.UserId.String(), err.Error())
		return
	}

	// sdk device为xxxx@xx@xx.xx的样式，返回第一部分即可
	idx = strings.Index(sdkDeviceID, "@")
	if idx == -1 {
		tilogs.L().Errorf("QueryNTID invalid device id %s", sdkDeviceID)
		return
	}

	c.String(http.StatusOK, sdkDeviceID[:idx])
}
