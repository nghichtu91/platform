package status

import (
	"fmt"
	"time"

	"github.com/nghichtu91/platform/share/x/common/authdiscovery"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/astaxie/beego/httplib"

	gateconfig "github.com/nghichtu91/platform/share/x/gatex/config"
)

var (
	PublicIP string
	RpcIP    string

	_login *loginConnector
)

type loginConnector struct {
	loginStatusChan chan gateconfig.LoginStatus

	stopSign chan struct{}
}

func newLoginConnector() *loginConnector {
	l := &loginConnector{
		loginStatusChan: make(chan gateconfig.LoginStatus, 1024),
		stopSign:        make(chan struct{}),
	}
	return l
}

func Start(publicip, rpcip string) {
	tilogs.L().Infof("gate ccumetric started.")

	PublicIP = publicip
	RpcIP = rpcip

	_login = newLoginConnector()
	_login.start(rpcip)
}

func Stop() {
	_login.stop()
}

func NotifyLogInOff(s gateconfig.LoginStatus) {
	select {
	case _login.loginStatusChan <- s:
	default:
		tilogs.L().Errorf("NotifyLogInOff _login.loginStatusChan full")
	}

}

func httppost(url, gameip, rpcip string, ccu int64, gateId uint) (string, error) {
	req := httplib.Post(url)
	req.Param("gameipaddrport", gameip)
	req.Param("rpcipaddrport", rpcip)
	req.Param("ccu", fmt.Sprintf("%d", ccu))
	req.Param("gateid", fmt.Sprintf("%d", gateId))

	var str string
	err := req.ToJSON(&str)
	if err != nil {
		return "", fmt.Errorf("login server update error %s", err.Error())
	}

	rsp, _ := req.Response()
	defer rsp.Body.Close()

	if str != "ok" {
		return "", fmt.Errorf("login server update error %s", str)
	}
	//tilogs.L().Infof("login server update success")
	return str, nil
}

func postLoginStatus(url, loginToken, accountId, RPCAddrPort string) {
	tilogs.L().Debugf("postLoginStatus: %s", url)
	req := httplib.Post(url).SetTimeout(2*time.Second, 2*time.Second)
	req.Param("logintoken", loginToken)
	req.Param("accountid", accountId)
	req.Param("rpcaddrport", RPCAddrPort)

	tilogs.L().Debugf("<Gate> postLoginStatus %s %s %s %s",
		url, loginToken, accountId, RPCAddrPort)

	var str string
	err := req.ToJSON(&str)
	if err != nil {
		tilogs.L().Errorf("<Gate> postLoginStatus failed err with %v", err)
		return
	}
	if str != "ok" {
		tilogs.L().Errorf("<Gate> postLoginStatus failed with %v", str)
		return
	}
}

func (l *loginConnector) notifyLogInOff() {
	defer tilogs.PanicCatcher("NotifyAuthUserInfo")

	c := l.loginStatusChan
	rpc := RpcIP
	for i := range c {
		etcd_info := authdiscovery.GetAuthInfo()
		if etcd_info == nil {
			continue
		}
		var url string
		if i.LogInOff {
			url = etcd_info.LoginLoginUrl
		} else {
			url = etcd_info.LoginLogoutUrl
		}

		go postLoginStatus(url,
			i.LoginToken,
			i.AccountID,
			rpc,
		)
	}
}

func (l *loginConnector) start(rpcip string) {
	go l.notifyLogInOff()
}

func (l *loginConnector) stop() {
	close(l.stopSign)
	close(l.loginStatusChan)
}
