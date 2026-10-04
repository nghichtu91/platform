package config

import (
	"sync"

	"github.com/nghichtu91/platform/share/x/chat/web/logic/chatSystem"

	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	common_config "github.com/nghichtu91/platform/share/x/common/config"
)

const (
	ChatLogicServerOSSFolder       = "chat/logic/"
	DefaultRoomSaveHistoryMsgCount = 50 // 如果etcd没有读取到保存历史消息的数量配置，使用这个默认值
	DefaultActivityChatStoreTime   = 3600
	PetBreedChannel                = "PetBornChannel"
)

var (

	// Cfg Chat配置
	Cfg ChatConfig

	// etcd config
	EtcdConf common_config.GidConfig

	// EtcdConfig KV配置此数据会热更
	EtcdConfig      = make(map[chatSystem.EtcdInfoType]map[string]int, 4)
	EtcdConfigMutex sync.RWMutex

	// run mode
	RunMode = "dev"

	// 是否允许机器人 以非正常流程登陆
	IsRobotNeedLogin = false

	// labelCode 对应的 描述
	LabelCode2Desc = make(map[int64]string, 10)
)

func IsDevelopMode() bool {
	return RunMode == "dev"
}

func LoadConfig(configName string) bool {
	config.NewConfigToml(configName, &Cfg)
	tilogs.SetTiLogger(tilogs.L().With(tilogs.TagServerId, Cfg.ServerId).With(tilogs.TagGid, Cfg.Gid))
	tilogs.L().Infof("chat config: %s", Cfg.String())

	if err := etcd.InitEtcd(Cfg.EtcdEndPoint); err != nil {
		tilogs.L().Errorf("etcd.InitEtcd err %s", err.Error())
		return false
	}

	// code2desc
	for _, ele := range Cfg.YiDun.Labels {
		LabelCode2Desc[ele.Code] = ele.Desc
	}
	return loadConfigFromEtcd()
}

func loadConfigFromEtcd() bool {
	pConfig := common_config.InitLoadGidConfig(Cfg.EtcdDevops, Cfg.Gid)
	if pConfig == nil {
		return false
	}
	EtcdConf = *pConfig
	return true
}

func GetChatConfigV(infoType chatSystem.EtcdInfoType, k string) int {
	EtcdConfigMutex.RLock()
	defer EtcdConfigMutex.RUnlock()
	return EtcdConfig[infoType][k]
}

func GetRoomSaveHistoryMsgCount(roomType string) int {
	if roomType == PetBreedChannel {
		return 0
	}

	if GetChatConfigV(chatSystem.Common, roomType) > 0 {
		return GetChatConfigV(chatSystem.Common, roomType)
	}
	return DefaultRoomSaveHistoryMsgCount
}

func GetRoomStoreMsgTime(roomType string) int {
	if GetChatConfigV(chatSystem.Time, roomType) > 0 {
		return GetChatConfigV(chatSystem.Time, roomType)
	}
	return Cfg.Room.MaxRoomKeySaveTime
}

func GetActivityRoomStoreMsgTime(roomType string) int {
	if GetChatConfigV(chatSystem.Time, roomType) > 0 {
		return GetChatConfigV(chatSystem.Time, roomType)
	}
	return DefaultActivityChatStoreTime
}
