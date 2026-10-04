package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/k8s"
	"github.com/nghichtu91/platform/share/planx/signalhandler"

	"go.uber.org/atomic"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/planx/etcd"

	"github.com/nghichtu91/platform/share/planx/cloud_db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type GidConfig struct {
	Proj         string `etcd3:"Proj"`
	RunMode      string `etcd3:"RunMode"`
	ServerType   string `etcd3:"ServerType"`
	MirrorEnable string `etcd3:"MirrorEnable"`
	CheatEnable  string `etcd3:"CheatEnable"`
	TimeLocal    string `etcd3:"TimeLocal"`
	DBDriver     string `etcd3:"DBDriver"`
	// 基于不同的实现， MailDBName的意义也不同
	// MailDBName 在mongo中是database， 在dynamo中是table， 在redis里面没用
	MailDBName string `etcd3:"MailDBName"`

	AwsRegion          string `etcd3:"AwsRegion"`
	AwsAccessKey       string `etcd3:"AwsAccessKey"`
	AwsSecretKey       string `etcd3:"AwsSecretKey"`
	AwsInitialInterval int    `etcd3:"AwsInitialInterval"`
	AwsMultiplier      int    `etcd3:"AwsMultiplier"`
	AwsMaxElapsedTime  int    `etcd3:"AwsMaxElapsedTime"`
	MongoUrl           string `etcd3:"MongoUrl"`
	MongoAuthDbName    string `etcd3:"MongoAuthDbName"`

	// account数据库
	AccountPikaDBUrl string `etcd3:"AccountPikaDBUrl"`
	AccountPikaDBPwd string `etcd3:"AccountPikaDBPwd"`
	AccountDb        int    `etcd3:"account_db"`
	// cross db
	CrossPikaDbAddr string `etcd3:"CrossPikaDbAddr"`
	CrossPikaDbNum  int    `etcd3:"CrossPikaDbNum"`
	CrossPikaDbPwd  string `etcd3:"CrossPikaDbPwd"`
	// 排行榜数据库
	BattleCheckRedisAddr  string `etcd3:"BattleCheckRedisAddr"`
	BattleCheckRedisDb    int    `etcd3:"BattleCheckRedisDb"`
	BattleCheckRedisDbPwd string `etcd3:"BattleCheckRedisDbPwd"`
	HotDataBucket         string `etcd3:"HotDataBucket"`
	BattleDataBucket      string `etcd3:"BattleDataBucket"`
	ChatDataBucket        string `etcd3:"ChatDataBucket"`
	PprofDataBucket       string `etcd3:"PprofDataBucket"`
	GiftCodeBatchTable    string `etcd3:"GiftCodeBatchTable"`
	GiftCodeListTable     string `etcd3:"GiftCodeListTable"`
	MysqlUrl              string `etcd3:"MysqlGmUrl"`
	ChatxRedisAddr        string `etcd3:"ChatxRedisAddr"`
	ChatxRedisDb          string `etcd3:"ChatxRedisDb"`
	ChatxRedisDbPwd       string `etcd3:"ChatxRedisDbPwd"`
	CachexRedisAddr       string `etcd3:"CachexRedisAddr"`
	CachexRedisDb         string `etcd3:"CachexRedisDb"`
	CachexRedisDbPwd      string `etcd3:"CachexRedisDbPwd"`
	FriendPikaAddr        string `etcd3:"FriendPikaAddr"`
	FriendPikaDB          int    `etcd3:"FriendPikaDB"`
	FriendPikaDBPwd       string `etcd3:"FriendPikaDBPwd"`
	NatsUrl               string `etcd3:"NatsUrl"` // 若多个，则用","隔开，如：nats://127.0.0.1:4222,nats://127.0.0.1:4222
	Gid                   uint
	// GiftCodeServer        string `etcd3:"GiftCodeServer"`
	ESUrl        string `etcd3:"ESUrl"`
	ESIndex      string `etcd3:"ESIndex"`
	ESGamexIndex string `etcd3:"ESGamexIndex"`
	// 备份数据库映射
	AccountPikaDBSrcUrl       string `etcd3:"AccountPikaDBSrcUrl"`
	AccountPikaDBDstUrl       string `etcd3:"AccountPikaDBDstUrl"`
	CloudDriver               string `etcd3:"CloudDriver"`
	CloudRegion               string `etcd3:"CloudRegion"`
	CloudAccessKey            string `etcd3:"CloudAccessKey"`
	CloudSecretKey            string `etcd3:"CloudSecretKey"`
	CloudRootDir              string `etcd3:"CloudRootDir"`       // 如oss，bucket内的根目录名称
	BattleCheckDisable        string `etcd3:"BattleCheckDisable"` // battlecheck是否禁用，true为禁用
	battleCheckDisable        atomic.String
	NewGameLoadHot            string `etcd3:"NewGameLoadHot"` // 新服自动加载热更文件和配置开关，false关，true开
	newGameLoadHot            atomic.String
	TypeNoticeCustomerService string `etcd3:"TypeNoticeCustomerService"` // 公告界面客服开关
	typeNoticeCustomerService atomic.String
	TestKcpPing               string `etcd3:"TestKcpPing"` // 公告界面客服开关
	testKcpPing               atomic.String
	AndroidPrivacy            string `etcd3:"AndroidPrivacy"` // 安卓隐私政策开关
	androidPrivacy            atomic.String
	IosPrivacy                string `etcd3:"IosPrivacy"` // ios隐私政策开关
	iosPrivacy                atomic.String
	Battle63WithZone          string `etcd3:"Battle63WithZone"` // 63大区战斗区分地区
	battle63WithZone          atomic.String

	UploadFrameDisable string `etcd3:"UploadFrameDisable"` // 客户端上传是否禁用，true为禁用，用来控制全部玩法是否上传帧数据
	uploadFrameDisable atomic.String
	battleInfoSwitch   *sync.Map // 各个玩法是否上传帧数据，分多个概率档位。由于sync.Map里有锁，会有锁拷贝，所以要改成指针类型

	TypeWinScanSwitchService string `etcd3:"TypeWinScanSwitchService"`
	typeWinScanSwitchService atomic.String

	// sentinel数据库哨兵设置
	RedisSentinelEnable string `etcd3:"RedisSentinelEnable"`
	RedisMasterName     string `etcd3:"RedisMasterName"`
	PikaSentinelEnable  string `etcd3:"PikaSentinelEnable"`
	PikaMasterName      string `etcd3:"PikaMasterName"`

	// supervisor操作账户和密码
	OpUser string `etcd3:"op_user"`
	OpPwd  string `etcd3:"op_pwd"`

	BattleDomainName string `etcd3:"BattleDomainName"` // 给客户端ping battle 用的域名

	// For k8s. 如果不为 "" 就认为当前运行在 k8s 集群中.
	K8sNamespace string `etcd3:"namespace"` // 当前大区的k8s namespace

	loadRevision int64 // 记录从etcd加载的版本号
}

func (g *GidConfig) String() string {
	ret, err := json.Marshal(g)
	if err != nil {
		tilogs.L().Errorf("gid config string err %v", err)
	}
	return string(ret)
}

func (g *GidConfig) GetCloudDBConfig() cloud_db.CloudDbConfig {
	return cloud_db.CloudDbConfig{
		DbDriver:    g.CloudDriver,
		Region:      g.CloudRegion,
		Bucket:      g.HotDataBucket,
		CloudDbRoot: g.CloudRootDir,
		AccessKey:   g.CloudAccessKey,
		SecretKey:   g.CloudSecretKey,
		Format:      "",
		Seq:         "",
	}
}

// GetLoadRevision 获取加载的版本号
func (g *GidConfig) GetLoadRevision() int64 {
	if g.loadRevision <= 0 {
		return -1
	}
	return g.loadRevision
}

// init GidConfig
func GenGidConfig() *GidConfig {
	return &GidConfig{
		battleInfoSwitch: new(sync.Map),
	}
}

// InitLoadGidConfig 初始化GidConfig, 仅限于服务启动时加载自身的配置. 动态配置不要走这个接口
func InitLoadGidConfig(etcdRoot string, gid uint) *GidConfig {
	gidConfig := LoadGidConfig(etcdRoot, gid)
	if gidConfig == nil {
		tilogs.L().Errorf("etcd LoadGidConfig failed")
		return nil
	}
	gidConfig.SetServiceCloseTimeout()
	return gidConfig
}

func LoadGidConfig(etcdRoot string, gid uint) *GidConfig {
	gidKey := fmt.Sprintf("%s/%d/defaults", etcdRoot, gid)
	    // 打印生成的键值
	tilogs.L().Infof("Trying to load config for gid: %d from key: %s", gid, gidKey)

	gidConfig := GenGidConfig()
	
	// 在绑定前打印初始化的 GidConfig 对象
	tilogs.L().Infof("Initialized GidConfig: %+v", gidConfig)
	rev, err := etcd.BindWithRev(gidKey, gidConfig)
	if err != nil {
		tilogs.L().Errorf("load config from etcd err gid %d, %v", gid, err)
		return nil
	}
	gidConfig.Gid = gid
	gidConfig.loadRevision = rev
	// tilogs.L().Infof("load gid config %v", gidConfig)

	// 初始化所有大区开关key
	initGidSwitchAtomicKey(gidConfig, etcdRoot, gid)

	return gidConfig
}

func initGidSwitchAtomicKey(gidConfig *GidConfig, root string, gid uint) {
	// 初始化新服加载热更开关，正式服默认true，非正式服默认false
	initNewGameLoadHot(gidConfig, root, gid)
	initTestKcpPing(gidConfig, root, gid)
	initPrivacy(gidConfig, root, gid)
	initBattle63WithZoneSwitch(gidConfig, root, gid)

	gidConfig.battleCheckDisable.Store(gidConfig.BattleCheckDisable)
	gidConfig.testKcpPing.Store(gidConfig.TestKcpPing)
	gidConfig.newGameLoadHot.Store(gidConfig.NewGameLoadHot)
	gidConfig.typeNoticeCustomerService.Store(gidConfig.TypeNoticeCustomerService)
	gidConfig.uploadFrameDisable.Store(gidConfig.UploadFrameDisable)
	gidConfig.typeWinScanSwitchService.Store(gidConfig.TypeWinScanSwitchService)
	gidConfig.androidPrivacy.Store(gidConfig.AndroidPrivacy)
	gidConfig.iosPrivacy.Store(gidConfig.IosPrivacy)
	gidConfig.battle63WithZone.Store(gidConfig.Battle63WithZone)
	gidConfig.StoreAllBattleInfoSwitch(root, gid)
}

func initNewGameLoadHot(gidConfig *GidConfig, root string, gid uint) {
	if gidConfig.NewGameLoadHot != "" {
		return
	}

	if planx.IsRunProd(gidConfig.RunMode) {
		gidConfig.NewGameLoadHot = "true"
	} else {
		gidConfig.NewGameLoadHot = "false"
	}

	err := etcd.Put(fmt.Sprintf("%s/%d/defaults/NewGameLoadHot", root, gid), gidConfig.NewGameLoadHot)
	if err != nil {
		tilogs.L().Errorf("put config from etcd err gid %d, %v", gid, err)
		return
	}
}

func initTestKcpPing(gidConfig *GidConfig, root string, gid uint) {
	if gidConfig.TestKcpPing != "" {
		return
	}

	gidConfig.TestKcpPing = "false"

	err := etcd.Put(fmt.Sprintf("%s/%d/defaults/TestKcpPing", root, gid), gidConfig.TestKcpPing)
	if err != nil {
		tilogs.L().Errorf("put config from etcd err gid %d, %v", gid, err)
		return
	}
}

func initPrivacy(gidConfig *GidConfig, root string, gid uint) {
	if gidConfig.AndroidPrivacy == "" {
		gidConfig.AndroidPrivacy = "false"

		err := etcd.Put(fmt.Sprintf("%s/%d/defaults/AndroidPrivacy", root, gid), gidConfig.AndroidPrivacy)
		if err != nil {
			tilogs.L().Errorf("put config from etcd err gid %d, %v", gid, err)
			return
		}
	}
	if gidConfig.IosPrivacy == "" {
		gidConfig.IosPrivacy = "false"

		err := etcd.Put(fmt.Sprintf("%s/%d/defaults/IosPrivacy", root, gid), gidConfig.IosPrivacy)
		if err != nil {
			tilogs.L().Errorf("put config from etcd err gid %d, %v", gid, err)
			return
		}
	}
}

func initBattle63WithZoneSwitch(gidConfig *GidConfig, root string, gid uint) {
	if gidConfig.Battle63WithZone == "" {
		gidConfig.Battle63WithZone = "true"

		err := etcd.Put(fmt.Sprintf("%s/%d/defaults/Battle63WithZone", root, gid), gidConfig.Battle63WithZone)
		if err != nil {
			tilogs.L().Errorf("put config from etcd err gid %d, %v", gid, err)
			return
		}
	}
}

// 此接口给只需要GidConfig中的runmode的情况用，不需要拿其他的配置信息
func GetRunMode(etcdRoot string, gid uint) (string, error) {
	key := fmt.Sprintf("%s/%d/defaults/RunMode", etcdRoot, gid)
	v, err := etcd.Get(key)
	if err != nil {
		tilogs.L().Errorf("load config from etcd err gid %d, %v", gid, err)
		return "", err
	}
	return v, nil
}

// 此接口给只需要GidConfig中的CheatEnable的情况用，不需要拿其他的配置信息
func GetCheatEnable(etcdRoot string, gid uint) (string, error) {
	key := fmt.Sprintf("%s/%d/defaults/CheatEnable", etcdRoot, gid)
	v, err := etcd.Get(key)
	if err != nil {
		tilogs.L().Errorf("load config from etcd err gid %d, %v", gid, err)
		return "", err
	}
	return v, nil
}

func GetProj(etcdRoot string, gid uint) (string, error) {
	key := fmt.Sprintf("%s/%d/defaults/Proj", etcdRoot, gid)
	v, err := etcd.Get(key)
	if err != nil {
		tilogs.L().Errorf("load config from etcd err gid %d, %v", gid, err)
		return "", err
	}
	return v, nil
}

var (
	quit chan struct{}
)

func init() {
	quit = make(chan struct{}, 1)
}

func (g *GidConfig) StopConfigBattleCheck() {
	close(quit)
}

func (g *GidConfig) StartWatch(wait *util.WaitGroupWrapper, etcdRoot string, gid uint) {
	// 监听gid battleCheckDisable
	watchKey := etcd.GetGidSwitchKey(etcdRoot, gid)

	etcd.WatchWithRevNoPrevRetry(watchKey, g.GetLoadRevision(), true, quit, wait, func(resp clientv3.WatchResponse) {
		for _, event := range resp.Events {
			if event.Kv == nil {
				continue
			}
			if event.Type == clientv3.EventTypePut {
				tilogs.L().Infof("GidSwitchState Change key:%s value:%s", string(event.Kv.Key), string(event.Kv.Value))
				switch getGidSwitchKey(event.Kv.Key) {
				case battleCheckDisable:
					g.updateValue(string(event.Kv.Value), battleCheckDisable)
				case newGameLoadHot:
					g.updateValue(string(event.Kv.Value), newGameLoadHot)
				case typeNoticeCustomerService:
					g.updateValue(string(event.Kv.Value), typeNoticeCustomerService)
				case typeWinScanSwitchService:
					g.updateValue(string(event.Kv.Value), typeWinScanSwitchService)
				case uploadFrameDisable:
					g.updateValue(string(event.Kv.Value), uploadFrameDisable)
				case typeTestKcpPing:
					g.updateValue(string(event.Kv.Value), typeTestKcpPing)
				case AndroidPrivacySwitch:
					g.updateValue(string(event.Kv.Value), AndroidPrivacySwitch)
				case IosPrivacySwitch:
					g.updateValue(string(event.Kv.Value), IosPrivacySwitch)
				case Battle63WithZone:
					g.updateValue(string(event.Kv.Value), Battle63WithZone)
				}
			}
		}
	})

	// 监听玩法帧数据上传概率battleInfoSwitch
	watchKey2 := etcd.GetBattleInfoSwitchKeyDir(etcdRoot, gid)

	etcd.WatchWithRevNoPrevRetry(watchKey2, g.GetLoadRevision(), true, quit, wait, func(resp clientv3.WatchResponse) {
		for _, event := range resp.Events {
			if event.Kv == nil {
				continue
			}
			if event.Type == clientv3.EventTypePut {
				g.SetBattleInfoSwitch(getGidSwitchKey(event.Kv.Key), string(event.Kv.Value))
			}
		}
	})
}

func getGidSwitchKey(k []byte) (key string) {
	sections := strings.Split(string(k), "/")
	if len(sections) == 0 {
		return
	}
	key = sections[len(sections)-1]
	return
}

func (g *GidConfig) updateValue(s string, key string) {
	switch key {
	case battleCheckDisable:
		g.BattleCheckDisable = s
		g.SetBattleCheckDisable(s)
	case newGameLoadHot:
		g.NewGameLoadHot = s
		g.SetNewGameLoadHot(s)
	case typeNoticeCustomerService:
		g.TypeNoticeCustomerService = s
		g.SetTypeNoticeCustomerService(s)
	case uploadFrameDisable:
		g.UploadFrameDisable = s
		g.SetUploadFrameDisable(s)
	case typeWinScanSwitchService:
		g.TypeWinScanSwitchService = s
		g.SetTypeWinScanSwitchService(s)
	case typeTestKcpPing:
		g.TestKcpPing = s
		g.SetTestKcpPing(s)
	case AndroidPrivacySwitch:
		g.AndroidPrivacy = s
		g.SetAndroidPrivacy(s)
	case IosPrivacySwitch:
		g.IosPrivacy = s
		g.SetIosPrivacy(s)
	case Battle63WithZone:
		g.Battle63WithZone = s
		g.SetBattle63WithZone(s)
	}
}

// 原子获取battlecheckdisable
func (g *GidConfig) GetBattleCheckDisable() string {
	return g.battleCheckDisable.Load()
}

func (g *GidConfig) SetBattleCheckDisable(check string) {
	g.battleCheckDisable.Store(check)
}

func (g *GidConfig) GetNewGameLoadHot() string {
	return g.newGameLoadHot.Load()
}

func (g *GidConfig) SetNewGameLoadHot(check string) {
	g.newGameLoadHot.Store(check)
}

func (g *GidConfig) GetTypeNoticeCustomerService() string {
	return g.typeNoticeCustomerService.Load()
}

func (g *GidConfig) SetTypeNoticeCustomerService(check string) {
	g.typeNoticeCustomerService.Store(check)
}

func (g *GidConfig) GetTypeWinScanSwitchService() string {
	return g.typeWinScanSwitchService.Load()
}

func (g *GidConfig) SetTypeWinScanSwitchService(check string) {
	g.typeWinScanSwitchService.Store(check)
}

func (g *GidConfig) GetTestKcpPing() string {
	return g.testKcpPing.Load()
}

func (g *GidConfig) SetTestKcpPing(check string) {
	g.testKcpPing.Store(check)
}

func (g *GidConfig) GetAndroidPrivacy() string {
	return g.androidPrivacy.Load()
}

func (g *GidConfig) SetAndroidPrivacy(check string) {
	g.androidPrivacy.Store(check)
}

func (g *GidConfig) GetIosPrivacy() string {
	return g.iosPrivacy.Load()
}

func (g *GidConfig) SetIosPrivacy(check string) {
	g.iosPrivacy.Store(check)
}

func (g *GidConfig) GetBattle63WithZone() bool {
	return g.battle63WithZone.Load() == "true"
}

func (g *GidConfig) SetBattle63WithZone(check string) {
	g.battle63WithZone.Store(check)
}

func (g *GidConfig) GetGidSuffixByServerType() string {
	switch g.ServerType {
	case ServerType_Channel:
		return ServerType_Channel
	case ServerType_Test:
		return ServerType_Test
	default:
		return ""
	}
}

func (g *GidConfig) StoreAllBattleInfoSwitch(root string, gid uint) {
	key := etcd.GetBattleInfoSwitchKeyDir(root, gid)
	res, err := etcd.GetSubRecursive(key)
	if err != nil {
		tilogs.L().Errorf("StoreAllBattleInfoSwitch GetSubRecursive failed , %v", err)
	}
	for k, v := range res {
		sections := strings.Split(string(k), "/")
		if len(sections) == 0 {
			continue
		}
		activityType := sections[len(sections)-1]
		g.battleInfoSwitch.Store(activityType, v)
	}
}

func (g *GidConfig) GetUploadFrameDisable() string {
	return g.uploadFrameDisable.Load()
}

func (g *GidConfig) SetUploadFrameDisable(check string) {
	g.uploadFrameDisable.Store(check)
}

func (g *GidConfig) GetBattleInfoSwitch(activityType string) string {
	// 默认开启，概率100
	probability, ok := g.battleInfoSwitch.LoadOrStore(activityType, "100")

	if ok && probability != "" {
		return probability.(string)
	}
	return "100"
}

func (g *GidConfig) SetBattleInfoSwitch(activityType, check string) {
	g.battleInfoSwitch.Store(activityType, check)
}

// SetServiceCloseTimeout 设置服务关闭超时时间. 函数可以重入, 随意调用. 以最后一次调用为准
func (g *GidConfig) SetServiceCloseTimeout() {

	const (
		RemoteWriteDir = "/opt/supervisor/log/supervisord" // 开发机/线上环境的堆栈信息写入目录.
	)

	var stopTimeout = map[string]signalhandler.TimeoutStackConfig{
		planx.RunMode_Prod: {
			StopTimeout: 30 * time.Second,
			WriteDir:    RemoteWriteDir,
		}, // 线上 45s 强杀进程,
		planx.RunMode_Dev: {
			StopTimeout: 20 * time.Second,
			WriteDir:    RemoteWriteDir,
		}, // qa 30s 强杀进程,
		planx.RunMode_Local: {
			StopTimeout: 10 * time.Second,
			WriteDir:    ".",
		}, // 本地, 等个 10s 够了吧...,
	}

	if cfg, ok := stopTimeout[g.RunMode]; ok {
		signalhandler.SetTimeout(cfg.StopTimeout)
		signalhandler.SetWriteDir(cfg.WriteDir)
	}
}

// IsRunMode 是否是X运行模式
func (g *GidConfig) IsRunMode(mode string) bool {
	return g.RunMode == mode
}

// IsRunProd 是否是生产环境
func (g *GidConfig) IsRunProd() bool {
	return planx.IsRunProd(g.RunMode)
}

// IsRunDev 是否是开发环境
func (g *GidConfig) IsRunDev() bool {
	return planx.IsRunDev(g.RunMode)
}

// IsRunInLocal 是否是本地环境(包括本地测试环境和本地环境)
func (g *GidConfig) IsRunInLocal() bool {
	return planx.IsRunInLocal(g.RunMode)
}

// IsRunLocal 是否是本地环境
func (g *GidConfig) IsRunLocal() bool {
	return planx.IsRunLocal(g.RunMode)
}

// IsRunLocalTest 是否是本地测试环境
func (g *GidConfig) IsRunLocalTest() bool {
	return planx.IsRunLocalTest(g.RunMode)
}

// IsRunPrefTest 是否是性能测试环境
func (g *GidConfig) IsRunPrefTest() bool {
	return planx.IsRunPrefTest(g.RunMode)
}

type RunEnvType int

const (
	RunEnvSuperVisor RunEnvType = iota
	RunEnvK8s
)

// Is 是否是t类型
func (r RunEnvType) Is(t RunEnvType) bool { return r == t }

// RunEnvType 运行环境类型
func (g *GidConfig) RunEnvType() RunEnvType {
	if g.K8sNamespace != "" {
		return RunEnvK8s
	}
	return RunEnvSuperVisor
}

// IsRunInK8s 是否是k8s环境
func (g *GidConfig) IsRunInK8s() bool { return g.RunEnvType().Is(RunEnvK8s) }

// InitK8s 初始化k8s客户端
func (g *GidConfig) InitK8s() error {
	if g.IsRunInK8s() {
		tilogs.L().Infof("running in k8s namespace %v", g.K8sNamespace)
		return k8s.InitClient(g.K8sNamespace)
	}
	return nil
}
