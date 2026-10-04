package battlex

import (
	"fmt"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"

	"github.com/nats-io/nats.go"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	discovery "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/nats_cli"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/servers/planxprotogen"

	"github.com/golang/protobuf/proto"
)

// 战斗服回调
type ILogicBattle interface {
	HandleGameBattleMsg(fatherSpan opentracing.Span, msg *planxprotogen.BattleMsg)
	OnServStop(serId, serType string)
}

type battle_server struct {
	etcdServer             string
	gid                    string
	battleSerId            string
	privateIp              string
	logicBattle            ILogicBattle
	serReg                 *etcd_ser_discovery.ServiceMgr
	disGamex               *etcd_ser_discovery.DiscoveryMgr
	disCross               *etcd_ser_discovery.DiscoveryMgr
	disBattleAgent         *etcd_ser_discovery.DiscoveryMgr
	createRoomSubscription []*nats.Subscription
	battleAgentIpPort      string //战斗代理连接的地址
	battleZone             string //战斗服所在地区
}

func NewBattleSer(etcdServer string, gid, battleSerId, privateIp, battleAgentIpPort, battleZone string, logicBattle ILogicBattle) *battle_server {
	return &battle_server{
		etcdServer:        etcdServer,
		gid:               gid,
		battleSerId:       battleSerId,
		privateIp:         privateIp,
		logicBattle:       logicBattle,
		battleAgentIpPort: battleAgentIpPort,
		battleZone:        battleZone,
	}
}

func (bSer *battle_server) Run() error {
	// 注册创建战斗房间消息处理
	if err := bSer.handlerCreateBattleRoom(); err != nil {
		return err
	}
	// 注册房间内消息处理
	bSer.handlerBattleRoom()
	// cross、gamex和battle之间能感知关服，做服务发现
	if err := bSer.regServ(); err != nil {
		return err
	}
	if err := bSer.discoverGamex(); err != nil {
		return err
	}
	if err := bSer.discoverCross(); err != nil {
		return err
	}
	if err := bSer.discoverBattleAgent(); err != nil {
		return err
	}
	return nil
}

func (bSer *battle_server) Stop() {
	bSer.serReg.Stop()
	bSer.disGamex.Stop()
	bSer.disCross.Stop()
	bSer.disBattleAgent.Stop()
}

func (bSer *battle_server) Downline() {
	for _, subscription := range bSer.createRoomSubscription {
		err := subscription.Unsubscribe()
		if err != nil {
			tilogs.L().Errorf("battlex Downline createRoomSubscription.Unsubscribe err %v", err)
		}
	}
	tilogs.L().Infof("battlex Downline createRoomSubscription.Unsubscribe")
}

// 处理创建房间请求
// 同主题，多服务注册，用nats实现同主题负载均衡
func (bSer *battle_server) handlerCreateBattleRoom() error {
	var (
		subscriptions []*nats.Subscription
		err           error
	)
	// 主题订阅失败，取消之前的订阅
	defer func() {
		if err != nil {
			for _, subscription := range subscriptions {
				if err := subscription.Unsubscribe(); err != nil {
					tilogs.L().Errorf("battlex handlerCreateBattleRoom subscription.Unsubscribe err %v", err)
				}
			}
		}
	}()

	var natsSubs []string
	// 大区维度的主题组
	gidSub := nats_cli.BattleFunc(bSer.gid, nats_cli.BattleSubj_CreateRoom)
	subscriptionGid, err := nats_cli.GetHandlersMgr().RegQueueNoSubWithLock(gidSub,
		&createRoomHandler{
			battle_server: bSer,
		})
	if err != nil {
		return err
	}
	subscriptions = append(subscriptions, subscriptionGid)
	natsSubs = append(natsSubs, strings.Join(gidSub, "."))

	// 地区维度主题组
	var zone = bSer.battleZone
	if zone != "" {
		zoneSub := nats_cli.BattleZoneFunc(bSer.gid, zone, nats_cli.BattleSubj_CreateRoom)
		subscriptionZone, err := nats_cli.GetHandlersMgr().RegQueueNoSubWithLock(zoneSub,
			&createRoomHandler{
				battle_server: bSer,
			})
		if err != nil {
			return err
		}
		subscriptions = append(subscriptions, subscriptionZone)
		natsSubs = append(natsSubs, strings.Join(zoneSub, "."))
	}
	tilogs.L().Infof("battlex handlerCreateBattleRoom natsSubs %v", natsSubs)

	bSer.createRoomSubscription = subscriptions
	return nil
}

type createRoomHandler struct {
	*battle_server
	msg *planxprotogen.BattleMsg
}

func (h *createRoomHandler) NewProtoMsg() proto.Message {
	h.msg = &planxprotogen.BattleMsg{}
	return h.msg
}

func (h *createRoomHandler) GetMsg() proto.Message {
	return h.msg
}

func (h *createRoomHandler) Handle(fatherSpan opentracing.Span) proto.Message {
	h.battle_server.logicBattle.HandleGameBattleMsg(fatherSpan, h.msg)
	return nil
}

// 处理战斗房间内其他消息
func (bSer *battle_server) handlerBattleRoom() {
	nats_cli.GetHandlersMgr().RegWithLock(nats_cli.BattleSubj(bSer.gid, bSer.battleSerId),
		&roomHandler{
			battle_server: bSer,
		})
}

type roomHandler struct {
	*battle_server
	msg *planxprotogen.BattleMsg
}

func (h *roomHandler) NewProtoMsg() proto.Message {
	h.msg = &planxprotogen.BattleMsg{}
	return h.msg
}

func (h *roomHandler) GetMsg() proto.Message {
	return h.msg
}

func (h *roomHandler) Handle(fatherSpan opentracing.Span) proto.Message {
	h.battle_server.logicBattle.HandleGameBattleMsg(fatherSpan, h.msg)
	return nil
}
func SendMsg(gid, servId, servType string, msg *planxprotogen.BattleMsg) error {
	return SendMsgWithCtx(nil, gid, servId, servType, msg)
}
func SendMsgWithCtx(fatherSpan opentracing.Span, gid, servId, servType string, msg *planxprotogen.BattleMsg) error {
	timeStart := time.Now()
	var scope []string

	switch servType {
	case etcd.Server_Gamex:
		scope = nats_cli.GamexSerSubj(gid, servId)
	case etcd.Server_Crossx:
		scope = nats_cli.CrossSerSubj(gid, servId)
	case etcd.Server_KcpRobot:
		scope = nats_cli.KcpRobotSerSubj(gid, servId)
	case etcd.Server_ServerCheck:
		scope = nats_cli.ServerCheckSerSubj(gid, servId)
	default:
		err := fmt.Errorf("battlex SendMsg servType not found %s", servType)
		tilogs.L().Errorf(err.Error())
		return err
	}

	if err := nats_cli.SendMsgWithCtx(fatherSpan, scope, msg); err != nil {
		tilogs.L().Errorf("battlex SendMsg to %s err %v", servType, err)
		return err
	}

	timeEnd := time.Now()
	timeSpend := timeEnd.Sub(timeStart)
	if timeSpend.Milliseconds() > 50 {
		tilogs.L().Errorf("nats slow time Spend %v msgId:%v bRoomId:%v serId:%v", timeSpend, msg.GetMsgId(), msg.GetBRoomId(), msg.GetSerId())
	}
	return nil
}

// 注册服务器发现，注册战斗服
func (bSer *battle_server) regServ() error {
	bSer.serReg = discovery.NewSeriveMgr(discovery.ServiceConfig{
		EtcdRoot: bSer.etcdServer,
		SerId: etcd_ser_discovery.ServiceId{
			SerTyp: etcd_ser_discovery.ServiceType{
				Gid:    bSer.gid,
				SerTyp: etcd.Ser_Battle,
			},
			SerId: bSer.battleSerId,
		},
		HasPrivateAddr: true,
		CheckHealth:    true,
		HasExtraInfo:   true,
	}, bSer)
	return bSer.serReg.Start()
}

func (bSer *battle_server) discoverGamex() error {
	bSer.disGamex = discovery.NewDiscoveryMgr(discovery.DiscoveryConfig{
		EtcdRoot: bSer.etcdServer,
		BeDiscoverySerTyp: discovery.ServiceType{
			Gid:    bSer.gid,
			SerTyp: etcd.Ser_Gamex,
		},
		HasPrivateAddr: true,
	}, bSer, nil)
	if bSer.disGamex == nil {
		return fmt.Errorf("battle_server discoverGamex NewDiscoveryMgr fail")
	}
	return bSer.disGamex.Start()
}
func (bSer *battle_server) discoverCross() error {
	bSer.disCross = discovery.NewDiscoveryMgr(discovery.DiscoveryConfig{
		EtcdRoot: bSer.etcdServer,
		BeDiscoverySerTyp: discovery.ServiceType{
			Gid:    bSer.gid,
			SerTyp: etcd.Ser_Crossx,
		},
		HasPrivateAddr: true,
	}, bSer, nil)
	if bSer.disCross == nil {
		return fmt.Errorf("battle_server discoverCross NewDiscoveryMgr fail")
	}
	return bSer.disCross.Start()
}

func (bSer *battle_server) discoverBattleAgent() error {
	bSer.disBattleAgent = discovery.NewDiscoveryMgr(discovery.DiscoveryConfig{
		EtcdRoot: bSer.etcdServer,
		BeDiscoverySerTyp: discovery.ServiceType{
			Gid:    bSer.gid,
			SerTyp: etcd.Ser_BattleAgent,
		},
		HasPrivateAddr: true,
	}, bSer, nil)
	if bSer.disBattleAgent == nil {
		return fmt.Errorf("battle_server disBattleAgent NewDiscoveryMgr fail")
	}
	return bSer.disBattleAgent.Start()
}

func (bSer *battle_server) GetPublicAddr() string {
	return ""
}
func (bSer *battle_server) GetPrivateAddr() string {
	return fmt.Sprintf("%s:%s", bSer.privateIp, bSer.battleSerId) // 这里只用这个机制，让game和cross能感知battle的关闭，所以并不关心private的实际值，就用serverid了
}

// 返回代理的ipport
func (bSer *battle_server) GetExtraInfo() string {
	return bSer.battleAgentIpPort
}
func (bSer *battle_server) GetTickExtraInfo() string {
	return ""
}

func (bSer *battle_server) OnAddService(info discovery.ServiceInfo) {

}
func (bSer *battle_server) OnDelService(serId discovery.ServiceId) {
	if serId.SerTyp.SerTyp == etcd.Ser_Gamex {
		bSer.logicBattle.OnServStop(serId.SerId, etcd.Server_Gamex)
	} else if serId.SerTyp.SerTyp == etcd.Ser_Crossx {
		bSer.logicBattle.OnServStop(serId.SerId, etcd.Server_Crossx)
	} else if serId.SerTyp.SerTyp == etcd.Ser_BattleAgent {
		// todo 代理的链接关闭依赖于 实际链接的连通性, 这里先临时屏蔽一下
		//bSer.logicBattle.OnServStop(serId.SerId, etcd.Server_BattleAgent)
	}
}

func (bSer *battle_server) OnChangeExtraInfo(info discovery.ServiceInfo) {

}
