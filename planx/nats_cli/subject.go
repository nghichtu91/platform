package nats_cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nats-io/nats.go"

	"github.com/nghichtu91/platform/share/planx/etcd"
)

const (
	BattleSubj_CreateRoom  = "CreateRoom"
	BattleSubj_BattleCheck = "BattleCheck"
	LogicxSubj_Validate    = "LogicValidate"
)

type BattleCheckType string

const (
	BattleCheckSubjAsync BattleCheckType = "Async"
	BattleCheckSubjSync  BattleCheckType = "Sync"
)

// gamex
func GamexSerSubj(gid, shardId string) []string {
	return []string{etcd.Server_Gamex, gid, shardId}
}

// GenGamexSerTopics 生成gamex的主题
func GenGamexSerTopics(gid uint, shardIds []uint) []string {
	topics := make([]string, 0, len(shardIds))
	gidStr := strconv.Itoa(int(gid))
	for _, sid := range shardIds {
		topics = append(topics, etcd.Server_Gamex+"."+gidStr+"."+strconv.Itoa(int(sid)))
	}
	return topics
}

func GamexFunc(gid string, f string) []string {
	return []string{etcd.Server_Gamex, gid, f}
}

// cross
func CrossSubj(gid string) []string {
	return []string{etcd.Server_Crossx, gid}
}

func CrossSerSubj(gid string, crossSerId string) []string {
	return []string{etcd.Server_Crossx, gid, crossSerId}
}

func CrossModuleGroup(gid, moduleName, groupId string) []string {
	return []string{etcd.Server_Crossx, gid, moduleName, groupId}
}

func CrossFunc(gid string, f string) []string {
	return []string{etcd.Server_Crossx, gid, f}
}

// battle
func BattleFunc(gid, f string) []string {
	return []string{etcd.Server_Battle, gid, f}
}

func BattleZoneFunc(gid, zone, f string) []string {
	return []string{etcd.Server_Battle, gid, zone, f}
}

func BattleSubj(gid, serId string) []string {
	return []string{etcd.Server_Battle, gid, serId}
}

// GM
func GMGidSubj(gid string) []string {
	return []string{etcd.Server_Gmtools, gid}
}
func GMSerSubj(gid string, gmSerId string) []string {
	return []string{etcd.Server_Gmtools, gid, gmSerId}
}

// Chat 内部 logic-》job的主题
func ChatSubj(gid string) []string {
	return []string{etcd.Server_ChatJob, gid}
}

// 游戏服发送到logic的主题
func ChatGamexSubj(gid string) []string {
	return []string{etcd.Server_ChatLogic, gid}
}

// LogicValidSubj 用于验证文本内容的接口
func LogicValidSubj(gid string) []string {
	return []string{etcd.Server_ChatLogic, gid, LogicxSubj_Validate}
}

// stress
func StressSerSubj(gid string, serId string) []string {
	return []string{etcd.Server_Stressx, gid, serId}
}

// friend
func FriendSubj(gid string) []string {
	return []string{etcd.Server_Friendx, gid}
}

// battlecheck
func BattleCheck(gid string, battleCheckType BattleCheckType) []string {
	return []string{etcd.Server_BattleCheck, gid, string(battleCheckType)}
}

// battlecheckSlow
func BattleCheckSlow(gid string, battleCheckType BattleCheckType) []string {
	return []string{etcd.Server_BattleCheck, gid, string(battleCheckType), "slow"}
}

// nats queue模式 主题格式
func BattleCheckFunc(gid string, battleCheckType BattleCheckType, f string) []string {
	return []string{etcd.Server_BattleCheck, gid, string(battleCheckType), f}
}

// nats 指定battlecheck 模式 （目前只支持Sync的房间）
func BattleCheckFuncWithBattleCheckid(gid string, battleCheckType BattleCheckType, f, battleCheckId string) []string {
	return []string{etcd.Server_BattleCheck, gid, string(battleCheckType), f, battleCheckId}
}

func BattleCheckSerSubj(gid string, battleCheckType BattleCheckType, serverId string) []string {
	return []string{etcd.Server_BattleCheck, gid, string(battleCheckType), serverId}
}

// kcprobot
func KcpRobotSerSubj(gid, shardId string) []string {
	return []string{etcd.Server_KcpRobot, gid, shardId}
}

func BattleCheckResultSubj(serverType, gid, ServerId string) []string {
	return []string{etcd.Server_BattleCheck, serverType, gid, ServerId}
}

func OfflineBattleCheckResultSubj(serverType, gid, ServerId string) []string {
	return []string{etcd.Server_BattleCheck, serverType, gid, "Offline", ServerId}
}

func ServerCheckSerSubj(gid, ServerId string) []string {
	return []string{etcd.Server_ServerCheck, gid, ServerId}
}

func CachexSubj(gid string) []string {
	return []string{etcd.Server_Cachex, gid} // cahcex.12
}

func SensitiveWordSubj(gid string) []string {
	return []string{etcd.Server_SensitiveWord, gid}
}

func GamexCachexSubj(gid, ServerId string) []string {
	return []string{etcd.Server_Gamex, etcd.Server_Cachex, gid, ServerId}
}

func GamexPlayerMirrorSubj(gid, ServerId string) []string {
	return []string{etcd.Server_Gamex, "player_mirror", gid, ServerId}
}

func SceneModuleGroup(gid, moduleName, groupId string) []string {
	return []string{etcd.Server_Scene, gid, moduleName, groupId}
}

// OtherCachexSubj cachex支持服务的Subject
func OtherCachexSubj(serTyp, gid, serverId string) []string {
	switch serTyp {
	case etcd.Server_Gamex, etcd.Server_Crossx, etcd.ServerSer_GMServer, etcd.Server_Stressx:
	default:
		return nil
	}

	return []string{etcd.Server_Cachex, serTyp, gid, serverId}
}

// OtherCachexTopic cachex支持服务的Subject
func OtherCachexTopic(serTyp, gid, serverId string) string {
	switch serTyp {
	case etcd.Server_Gamex, etcd.Server_Crossx, etcd.ServerSer_GMServer, etcd.Server_Stressx:
	default:
		return ""
	}

	return etcd.Server_Cachex + "." + serTyp + "." + gid + "." + serverId
}

// CachexSendTopic 向cachex发送请求时的nats topic
func CachexSendTopic(gid string) string {
	return etcd.Server_Cachex + "." + gid
}

func topic2Metric(gid, topic, oper string) (string, string) {
	if len(topic) <= 0 || strings.Contains(topic, nats.InboxPrefix) {
		return "", ""
	}
	tt := strings.Split(topic, ".")
	if len(tt) <= 0 {
		return "", ""
	}
	last := tt[len(tt)-1]
	_, err := strconv.Atoi(last)

	if err == nil && last != gid { // 是数字但不是gid，则去掉
		k4stat := fmt.Sprintf("nats%s%v", oper, strings.Join(tt[:len(tt)-1], ""))
		return fmt.Sprintf("requests.%s.nats.%s", gid, k4stat), k4stat
	}
	k4stat := fmt.Sprintf("nats%s%v", oper, strings.Join(tt, ""))
	return fmt.Sprintf("requests.%s.nats.%s", gid, k4stat), k4stat
}
