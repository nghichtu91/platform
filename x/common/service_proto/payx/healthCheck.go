package payx

import (
	"strings"

	"github.com/astaxie/beego/httplib"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/service_proto/common"
)

const (
	healthCheckSuffix = "/sc_pay_callback"
)

func HealthCheck(uri string) error {
	if !strings.HasSuffix(uri, "/") {
		uri += "/"
	}

	req := common.NewReq(uri+healthCheckSuffix, httplib.Post)

	ret, err := req.String()
	if err != nil {
		return err
	}

	tilogs.L().Debugf("get payx health check resp: %s", ret)

	return nil
}
