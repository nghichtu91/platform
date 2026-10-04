package gift_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/gift_client/gift_api"
	"github.com/nghichtu91/platform/share/x/gift/model"
)

func PostGiftCodeServer(apiUrl string, req, resp interface{}) (error, string) {
	respR := &model.CommonResult{
		Data: resp,
	}

	// 从ETCD中获取GiftCode的礼包码服务器地址。
	urlAPI := giftCodeUrl + gift_api.RootAPI + apiUrl
	if client == nil {
		return fmt.Errorf("PostGiftCodeServer client is nil"), ""
	}

	reqBytes, errEn := json.Marshal(req)
	if errEn != nil {
		return fmt.Errorf("PostGiftCodeServer json.Marshal(req) err %v", errEn), ""
	}

	tilogs.L().Debugf("PostGiftCodeServer urlAPI: %v", urlAPI)
	respInfo, errResp := client.Post(urlAPI, "application/json", bytes.NewBuffer(reqBytes))
	defer func() {
		if respInfo != nil && respInfo.Body != nil {
			if err := respInfo.Body.Close(); err != nil {
				tilogs.L().Errorf("PostGiftCodeServer respInfo.Body.Close err: %v", err)
				return
			}
		}
	}()

	if errResp != nil {
		return fmt.Errorf("PostGiftCodeServer client.Post err %v", errResp), ""
	}

	result, _ := ioutil.ReadAll(respInfo.Body)

	errDe := json.Unmarshal(result, &respR)
	if errDe != nil {
		return fmt.Errorf("PostGiftCodeServer json.Unmarshal(result, &resp) err %v", errDe), ""
	}

	tilogs.L().Debugf("Message %v", respR.Message)
	tilogs.L().Debugf("Code %v", respR.Code)

	if respR.Code != 200 {
		return errors.New(respR.Message), respR.Tag
	}
	return nil, ""
}
