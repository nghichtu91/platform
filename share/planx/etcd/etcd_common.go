package etcd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func GetHotDataEtcdProgressKey(etcdroot string, gid uint) string {
	return fmt.Sprintf("%s/%d/%s/progress", etcdroot, gid, DirHotWatch)
}

func GetHotDataPathPrefix(etcdroot string, gid uint, serverName string, serverId string) string {
	return fmt.Sprintf("%s/%d/%s/%s/%s", etcdroot, gid, DirHotData, serverId, serverName)
}

func GetHotDataEtcdRoot(etcdroot, gid string) string {
	return fmt.Sprintf("%s/%s/%s", etcdroot, gid, DirHotData)
}

func GetHotDataCrossCurrentEtcdRoot(etcdRoot, gid string) string {
	return fmt.Sprintf("%s/%s/%s/0/crossx/current", etcdRoot, gid, DirHotData)
}

type ProgressInfo struct {
	ServiceName   string
	Progress      int
	ErrorMsg      string
	HotData       string
	AvailableTime int64
	FirstUpdate   bool
	IsAll         bool
	UpdateList    []string
	BuildingId    string
	SourcePath    string
}

// IsValidSourcePath 检查sourcePath是否合法
func (p *ProgressInfo) IsValidSourcePath() bool {
	if p.SourcePath == "" {
		return false
	}
	// 判断文件夹是否存在
	fileInfo, err := os.Stat(p.SourcePath)
	if err != nil {
		tilogs.L().Errorf("sourcePath %s not exist, err %v", p.SourcePath, err)
		return false
	}
	if !fileInfo.IsDir() {
		tilogs.L().Errorf("sourcePath %s is not a directory", p.SourcePath)
		return false
	}
	return true
}

type HotUpdateVersion struct {
	BaseDataBuild   string `json:"basedatabuild"` // 初始数据版本， 游戏中以BaseDataBuild为标志
	HotDataBuild    int    `json:"hotdatabuild"`  // 在初始数据上包装的热更数据号， 用来指引热更
	PreHotDataBuild int    `json:"prehotdatabuild"`
	AvailableTime   int64  `json:"availabletime"`
}

type HotFixProgress struct {
	tag      int
	progress int
	msg      string
}

func (h *HotFixProgress) GetProgress() int {
	return h.progress
}

func (h *HotFixProgress) GetMsg() string {
	return h.msg
}

func (h *HotFixProgress) GetTag() int {
	return h.tag
}

// 描述一组服务器的热更过程
var HotDataProgress_Write_Etcd = &HotFixProgress{
	tag:      1,
	progress: 1,
	msg:      "通知%s热更信息",
} // 写入etcd
var HotDataProgress_Read_Version = &HotFixProgress{
	tag:      2,
	progress: 2,
	msg:      "%s读取更新信息",
} // 读etcd
var HotDataProgress_Downloading = &HotFixProgress{
	tag:      3,
	progress: 3,
	msg:      "%s下载热更数据中...",
} // 下载或拷贝
var HotDataProgress_Loading = &HotFixProgress{
	tag:      4,
	progress: 4,
	msg:      "%s读取热更数据中...",
} // 读取数据
var HotDataProgress_OK = &HotFixProgress{
	tag:      5,
	progress: 5,
	msg:      "%s更新成功",
} // 成功
var HotDataProgress_Notify = &HotFixProgress{
	tag:      6,
	progress: 6,
	msg:      "%s通知下一个服务器",
} // 承接
var HotDataProgress_SolfLink = &HotFixProgress{
	tag:      7,
	progress: 3,
	msg:      "%s正在拷贝文件(创建软链接)",
}

var HotDataMsgMap = map[int]*HotFixProgress{
	1: HotDataProgress_Write_Etcd,
	2: HotDataProgress_Read_Version,
	3: HotDataProgress_Downloading,
	4: HotDataProgress_Loading,
	5: HotDataProgress_OK,
	6: HotDataProgress_Notify,
	7: HotDataProgress_SolfLink,
}

const (
	HotDataEtcdKey_Update   = "update"
	HotDataEtcdKey_Version  = "version"
	HotDataEtcdKey_Error    = "error"
	HotDataEtcdKey_Current  = "current"
	HotDataEtcdKey_Delta    = "delta"
	HotDataEtcdKey_Previous = "previous"
	HotDataEtcdKey_Hope     = "hope"
)

func SetHotDataProgress(etcdroot string, gid uint, serverId string, serviceName string, progress int, hotData string, availableTime int64, firstUpdate, isAll bool, serverList []string, sourcePath string) {
	if firstUpdate {
		return
	}
	SetHotDataProgressError(etcdroot, gid, serverId, serviceName, progress, "", hotData, availableTime, firstUpdate, isAll, serverList, sourcePath)
}

func SetHotDataProgressError(etcdroot string, gid uint, serverId string, serviceName string, progress int, errMsg string, hotData string, availableTime int64, firstUpdate, isAll bool, serverList []string, sourcePath string) {
	key := strings.Join([]string{GetHotDataEtcdProgressKey(etcdroot, gid), serviceName, serverId}, "/")
	jsonBytes, err := json.Marshal(ProgressInfo{
		Progress:      progress,
		ErrorMsg:      errMsg,
		ServiceName:   serviceName,
		HotData:       hotData,
		AvailableTime: availableTime,
		UpdateList:    serverList,
		FirstUpdate:   firstUpdate,
		IsAll:         isAll,
		SourcePath:    sourcePath,
	})
	if err != nil {
		tilogs.L().Errorf("SetHotDataProgress json error %v", err)
		return
	}
	err = Put(key, string(jsonBytes))
	if err != nil {
		tilogs.L().Errorf("SetHotDataProgress put etcd error %v", err)
	} else {
		tilogs.L().Debugf("SetHotDataProgress put key=%s, value=%v", key, string(jsonBytes))
	}
}

// -------------------------------------------------------------------------
type HotPatchVersion struct {
	HotPatchBuild string `json:"hotpatchbuild"`
}

func GetHotPatchPathPrefix(etcdRoot string, gid uint, serverName, serverId string) string {
	return fmt.Sprintf("%s/%d/%s/%s/%s", etcdRoot, gid, DirPatch, serverName, serverId)
}

func GetHotPatchPathWatch(etcdRoot string, gid uint) string {
	return fmt.Sprintf("%s/%d/%s", etcdRoot, gid, DirPatchWatch)
}

func GetHotPatchPathProgressPath(etcdRoot string, gid uint, serverName, serverId string) string {
	return fmt.Sprintf("%s/%d/%s/progress/%s/%s", etcdRoot, gid, DirPatchWatch, serverName, serverId)
}

func GetHotPatchEtcdProgressKey(prefix string) string {
	return fmt.Sprintf("%s/progress", prefix)
}

func GetHotPatchCloudDBPath(version, serverName, buildId string) string {
	return fmt.Sprintf("%s/%s/%s/%s", DirPatch, version, serverName, buildId)
}

func GetHotPatchEtcdRoot(etcdroot, gid string) string {
	return fmt.Sprintf("%s/%s/%s", etcdroot, gid, DirPatch)
}

// 描述一组服务器的热更过程
var HotPatchProgress_Write_Etcd = &HotFixProgress{
	tag:      1,
	progress: 1,
	msg:      "通知%s热更信息",
} // 写入etcd
var HotPatchProgress_Read_Version = &HotFixProgress{
	tag:      2,
	progress: 2,
	msg:      "%s读取更新信息",
} // 读etcd
var HotPatchProgress_Downloading = &HotFixProgress{
	tag:      3,
	progress: 3,
	msg:      "%s下载热更数据中...",
} // 下载或拷贝
var HotPatchProgress_Loading = &HotFixProgress{
	tag:      4,
	progress: 4,
	msg:      "%s读取热更数据中...",
} // 读取数据
var HotPatchProgress_OK = &HotFixProgress{
	tag:      5,
	progress: 5,
	msg:      "%s更新成功",
} // 成功
var HotPatchProgress_SoftLink = &HotFixProgress{
	tag:      7,
	progress: 3,
	msg:      "%s正在拷贝文件(创建软链接)",
} // 承接

var HotPatchMsgMap = map[int]*HotFixProgress{
	1: HotPatchProgress_Write_Etcd,
	2: HotPatchProgress_Read_Version,
	3: HotPatchProgress_Downloading,
	4: HotPatchProgress_Loading,
	5: HotPatchProgress_OK,
	7: HotPatchProgress_SoftLink,
}

const (
	HotPatchEtcdKey_Update   = "update"
	HotPatchEtcdKey_Version  = "version"
	HotPatchEtcdKey_Error    = "error"
	HotPatchEtcdKey_Current  = "current"
	HotPatchEtcdKey_Delta    = "delta"
	HotPatchEtcdKey_Previous = "previous"
)

func SetHotPatchProgress(path string, serviceName string, progress int, buildingId string, serverList []string, firstUpdate bool, sourcePath string) {
	if firstUpdate {
		return
	}
	SetHotPatchProgressError(path, serviceName, progress, "", buildingId, serverList, firstUpdate, sourcePath)
}

func SetHotPatchProgressError(path, serviceName string, progress int, errMsg string, buidingId string, serverList []string, firstUpdate bool, sourcePath string) {
	if firstUpdate {
		return
	}
	key := path
	jsonBytes, err := json.Marshal(ProgressInfo{
		Progress:    progress,
		ErrorMsg:    errMsg,
		ServiceName: serviceName,
		UpdateList:  serverList,
		BuildingId:  buidingId,
		SourcePath:  sourcePath,
	})
	if err != nil {
		tilogs.L().Errorf("SetHotPatchProgress json error %v", err)
		return
	}
	err = Put(key, string(jsonBytes))
	if err != nil {
		tilogs.L().Errorf("SetHotPatchProgress put etcd error %v", err)
	} else {
		tilogs.L().Debugf("SetHotPatchProgress put key=%s, value=%v", key, string(jsonBytes))
	}
}

// -------------------------------------------------------------------------

func GetDebugTimeKey(gid uint, etcd_server string) string {
	return fmt.Sprintf("%s/%d/%s", etcd_server, gid, KeyDebugTimePath)
}

// 获取etcd上所有大区的gid
func GetAllGid(etcd_server string) ([]string, error) {
	return GetSubKeyOnly(etcd_server)
}

// 拼接 获取etcd上服务器名字 的路径
func GetServerNamePath(address string, gid uint, sid uint) string {
	return fmt.Sprintf("%s/%d/shards/%d/displayName", address, gid, sid)
}

// GetMergedUID 获取指定服务器的合服唯一ID
func GetMergedUID(srvRoot string, gid, sid uint) (int, error) {
	key := fmt.Sprintf("%s/%d/%s/%d/%s", srvRoot, gid, Merges, sid, KeyMergedUID)
	v, err := Get(key)
	if err != nil {
		return 0, err
	}

	if v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		return id, nil
	}

	return 0, nil
}

func GetDevopsCrossSpecificPrefix(etcdRoot string, gid uint, serverId uint) string {
	return fmt.Sprintf("%s/%d/%s/%s%d/", etcdRoot, gid, CrossxDir, CrossxDir, serverId)
}

func GetDevopsDefaultPrefix(etcdRoot string, gid uint) string {
	return fmt.Sprintf("%s/%d/%s/", etcdRoot, gid, Defaults)
}

func GetAutoLeaveGuildKey(etcd_server string, gid uint) string {
	return fmt.Sprintf("%s/%d/defaults/autoleaveguildcheat", etcd_server, gid)
}

const (
	virtualWarZonePath = "virtual_war_zone"
)

func GetVirtualWarZonePath(etcd_server string, gid uint) string {
	return fmt.Sprintf("%s/%d/%s", etcd_server, gid, virtualWarZonePath)
}
