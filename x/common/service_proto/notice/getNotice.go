package notice

import (
	"strings"

	"github.com/astaxie/beego/httplib"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/common/service_proto/common"
)

const (
	noticeSuffix = "notice/v1/getnotice"
)

func GetNotice(uri, gid, version string) error {
	if !strings.HasSuffix(uri, "/") {
		uri += "/"
	}

	req := common.NewReq(uri+noticeSuffix, httplib.Get).
		Header(consts.Spec_Header, consts.Spec_Header_Content).
		Param("gid", gid).
		Param("version", version).
		Param("language", "zh-CN")

	ret, err := req.String()
	if err != nil {
		return err
	}

	// TODO 后续考虑增加公告校验
	tilogs.L().Debugf("get notice ret: %s", ret)

	return nil
}
