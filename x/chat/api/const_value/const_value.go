package const_value

import (
	"strings"

	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	cometConfig "github.com/nghichtu91/platform/share/x/chat/cometx/config"
)

const (
	ForbiddenForever = -1
	RobotToken       = "robot"
)

func IsIgnoreHealthIp(addr string) bool {
	tilogs.L().Debugf("Check HealthIp %s", addr)
	return util.NeedFilterElbAddr(addr, cometConfig.Cfg.ElbAddr)
}

func GetRoomType(roomId string) string {
	index := strings.Index(roomId, "_")
	if index == -1 {
		return ""
	}
	return roomId[:index]
}
