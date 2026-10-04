package clientbilog

import (
	"fmt"
	"strconv"
	"time"

	"github.com/astaxie/beego/httplib"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/common/service_proto/common"
)

// BiLogResp 回复
type BiLogResp struct {
	Result   string `json:"result"`
	ClientIP string `json:"ip"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	SerTime  int64  `json:"ser_time"`
}

func SendRobotClientBI(uri string) error {
	req := common.NewReq(uri+clientBIV2, httplib.Post).
		Param("channel", common.Enc(consts.RobotChannel)). // 使用这个渠道号确保测试请求不会进bilog
		Param("clienttime", common.Enc(strconv.Itoa(int(time.Now().Unix())))).
		Param("device", common.Enc(consts.RobotDevice))

	resp := new(BiLogResp)
	err := req.ToJSON(resp)
	if err != nil {
		return err
	}

	if resp.Result != "ok" {
		return fmt.Errorf("clientbilog response failed")
	}

	return nil
}
