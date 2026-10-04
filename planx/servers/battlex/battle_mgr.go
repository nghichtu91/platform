package battlex

import (
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/nats_cli"
	"github.com/nghichtu91/platform/share/planx/servers/planxprotogen"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type GenBattleSerHandler func(battleSerId string) IBattleSerHandler

type IBattleSerHandler interface {
	OnRevBattleServMsg(fatherSpan opentracing.Span, msg *planxprotogen.BattleMsg)
}

func StartBattleMgr(gid uint, servType etcd.ServiceTypeId, serverId string,
	genHandler GenBattleSerHandler) {
	gBattleServsMgr = &battleServsMgr{
		serverId: serverId,
		handler:  genHandler(serverId),
	}
	gBattleServsMgr.regRecBattlexHandlers(gid, servType, serverId)
}

var gBattleServsMgr *battleServsMgr

type battleServsMgr struct {
	serverId string
	handler  IBattleSerHandler
}

// 来自battle的消息
func (manager *battleServsMgr) regRecBattlexHandlers(gid uint, servType etcd.ServiceTypeId, serverId string) {
	var scopeFn func(string, string) []string
	switch servType {
	case etcd.Ser_Crossx:
		scopeFn = nats_cli.CrossSerSubj
	case etcd.Ser_Gamex:
		scopeFn = nats_cli.GamexSerSubj
	case etcd.Ser_KcpRobot:
		scopeFn = nats_cli.KcpRobotSerSubj
	case etcd.Ser_ServerCheck:
		scopeFn = nats_cli.ServerCheckSerSubj
	default:
		tilogs.L().Errorf("servType %s not supported", servType)
		return
	}

	nats_cli.GetHandlersMgr().RegWithLock(scopeFn(strconv.Itoa(int(gid)), serverId),
		&recBattlexBattleRoomMsg{handler: manager.handler})
}

type recBattlexBattleRoomMsg struct {
	handler IBattleSerHandler
	msg     *planxprotogen.BattleMsg
}

func (req *recBattlexBattleRoomMsg) NewProtoMsg() proto.Message {
	req.msg = &planxprotogen.BattleMsg{}
	return req.msg
}

func (req *recBattlexBattleRoomMsg) GetMsg() proto.Message {
	return req.msg
}

func (req *recBattlexBattleRoomMsg) Handle(fatherSpan opentracing.Span) proto.Message {
	req.handler.OnRevBattleServMsg(fatherSpan, req.msg)
	return nil
}

// SendCreateBattleRoomMsg 其他需要和战斗服通信的服务，需要调用，如gamex或cross
// 大区战斗服池子
func SendCreateBattleRoomMsg(fatherSpan opentracing.Span, gid string, msg *planxprotogen.BattleMsg) error {
	return sendCreateBattleRoomMsgWithScope(fatherSpan, nats_cli.BattleFunc(gid, nats_cli.BattleSubj_CreateRoom), msg)
}

// SendCreateBattleRoomMsgWithZone 其他需要和战斗服通信的服务，需要调用，如gamex或cross
// 大区下地区划分的池子
func SendCreateBattleRoomMsgWithZone(fatherSpan opentracing.Span, gid, zone string, msg *planxprotogen.BattleMsg) error {
	return sendCreateBattleRoomMsgWithScope(fatherSpan, nats_cli.BattleZoneFunc(gid, zone, nats_cli.BattleSubj_CreateRoom), msg)
}

func sendCreateBattleRoomMsgWithScope(fatherSpan opentracing.Span, scope []string, msg *planxprotogen.BattleMsg) error {
	if fatherSpan != nil {
		return nats_cli.SendMsgWithCtx(fatherSpan, scope, msg)
	}
	return nats_cli.SendMsg(scope, msg)
}
