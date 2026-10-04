package auth

import (
	"errors"
	"time"

	"github.com/astaxie/beego/httplib"

	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/x/auth/errorctl"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/common/service_proto/common"
)

var (
	ErrUnknown     = errors.New("unknown")          // 100 未知错误
	ErrNotGate     = errors.New("no gate")          // 402 没有可用的gate
	ErrMaintenance = errors.New("server maintains") // 100001 维护或阻挡中
	ErrFull        = errors.New("server full")      // 100003 爆满，不允许注册新账号
	ErrAbnormal    = errors.New("server abnormal")  // 100004 锁服、未启动、未开启等状态，不应该被看到
)

type GetGateResp struct {
	EncIP     string `json:"ip"`
	EncToken  string `json:"logintoken"`
	Result    string `json:"result"`
	RecallID  string `json:"recallid"`
	Team      string `json:"team"`
	Error     string `json:"error,omitempty"`
	ForClient int    `json:"forclient,omitempty"`
}

// GetGate 获取gate登录ip和token
func GetGate(url, authToken, sn string) (ip, loginToken string, err error) {
	return GetGateRequireHost(url, authToken, sn, false)
}

// GetGateRequireHost 获取gate登录ip和token. 尝试获取host代替ip
func GetGateRequireHost(url, authToken, sn string, requireHost bool) (ip, loginToken string, err error) {
	req := common.NewReq(url+getGatePath, httplib.Get).
		Param("at", authToken).
		Param("sn", sn).
		Param("channelId", consts.RobotChannel)
	if requireHost {
		req = req.Param("requireHost", "1")
	}

	resp := new(GetGateResp)
	err = req.ToJSON(resp)
	if err != nil {
		return
	}

	switch resp.Result {
	case "ok":
		ip = common.Dec(resp.EncIP)
		loginToken = common.Dec(resp.EncToken)
		return
	case "no":
		switch resp.ForClient {
		case errorctl.ClientErrorGetGateNotExist:
			err = ErrNotGate
		case errorctl.ClientErrorUnknown:
			err = ErrUnknown
		case errorctl.ClientErrorGetGateMaintenance:
			err = ErrMaintenance
		case errorctl.ClientErrorGetGateServerIsFull:
			err = ErrFull
		case errorctl.ClientErrorGetGateServerIsAbnormality:
			err = ErrAbnormal
		}
		return
	case "retry":
		<-timeutil.TimerSec.After(time.Second * 2)
		return GetGateRequireHost(url, authToken, sn, requireHost)
	}

	return
}
