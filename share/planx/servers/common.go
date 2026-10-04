package servers

import (
	"github.com/opentracing/opentracing-go"

	"github.com/nghichtu91/platform/share/planx/nats_cli"
	"github.com/nghichtu91/platform/share/planx/servers/planxprotogen"
)

// serType是etcd下的服务器类型枚举, 如：etcd.Server_CrossService
type OnServStopFunc func(serId, serType string)

// 给战斗服发消息
func SendBattleMsg(gid, servId string, msg *planxprotogen.BattleMsg) error {
	return SendBattleMsgWithCtx(nil, gid, servId, msg)
}

// 给战斗服发消息
func SendBattleMsgWithCtx(fatherSpan opentracing.Span, gid, servId string, msg *planxprotogen.BattleMsg) error {
	if fatherSpan != nil {
		return nats_cli.SendMsgWithCtx(fatherSpan, nats_cli.BattleSubj(gid, servId), msg)
	}
	return nats_cli.SendMsg(nats_cli.BattleSubj(gid, servId), msg)
}
