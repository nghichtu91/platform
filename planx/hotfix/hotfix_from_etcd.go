package hotfix

// #cgo CFLAGS: -g -O -Wall
// #cgo LDFLAGS: -ldl
// #include <stdlib.h>
// #include "patch.h"
import "C"
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
	"unsafe"

	"github.com/nghichtu91/platform/share/planx/util/file"

	"github.com/nghichtu91/platform/share/planx/version"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/cloud_db"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

var mgr *HotPatchMng

func InitHotPatch(option HotPatchOption, wait *util.WaitGroupWrapper, newGameLoadHot string) error {
	mgr = &HotPatchMng{
		option: option,
		wait:   wait,
		quit:   make(chan struct{}, 1),
	}
	if err := mgr.Start(newGameLoadHot); err != nil {
		return err
	}
	return nil
}

func StopHotPatch() {
	close(mgr.quit)
}

type HotPatchMng struct {
	option  HotPatchOption
	cloudDb cloud_db.CloudDb

	initValue      string
	curHotPatchVer string
	etcdKeyPrefix  string

	wait *util.WaitGroupWrapper
	quit chan struct{}
}

type HotPatchOption struct {
	Gid uint

	// etcd 监听配置
	EtcdRoot    string
	ServiceName string
	ServerId    string

	// 云存储读取配置
	Bucket string // 云存储的桶

	CloudDbCfg cloud_db.CloudDbConfig
}

func (m *HotPatchMng) Start(newGameLoadHot string) error {
	m.etcdKeyPrefix = etcd.GetHotPatchPathPrefix(m.option.EtcdRoot, m.option.Gid, m.option.ServiceName, m.option.ServerId)
	if err := m.initCloudDb(); err != nil {
		return err
	}
	m.startWatch(newGameLoadHot)
	return nil
}

func (m *HotPatchMng) startWatch(newGameLoadHot string) bool {
	updateKey := m.getUpdateKey()
	initValue, initVer, err := etcd.GetWithRev(updateKey) // 先获取初始值
	if err != nil {
		// may be key not found
		tilogs.L().Warnf("HotPatchMng get watch init value err %v", err)
	} else {
		m.initValue = initValue
	}
	if m.initValue != "" {
		err := m.onWatch(m.initValue, true)
		if err != "" {
			tilogs.L().Errorf("HotPatchMng init hot update value error %v", err)
			if newGameLoadHot == "true" {
				panic(fmt.Errorf("HotPatchMng init hot update value error %v", err))
			} else {
				if err := etcd.Put(m.getCurrentKey(), ""); err != nil {
					tilogs.L().Errorf("<HotPatchMng> after reload etcd.Put err %v", err)
				}
			}
		}
	}
	// 写入当前版本号
	versionKey := m.getVersionKey()
	etcd.Put(versionKey, version.Version)

	tilogs.L().Infof("<HotPatchMng> start watch hot update key %s", updateKey)
	etcd.WatchWithRevRetry(updateKey, initVer, false, false, m.quit, m.wait,
		func(resp clientv3.WatchResponse) {
			for _, e := range resp.Events {
				switch e.Type {
				case clientv3.EventTypePut:
					m.onWatch(string(e.Kv.Value), false)
				}
			}
		})
	return true
}

func (m *HotPatchMng) onWatch(verString string, firstUpdate bool) string {
	defer tilogs.PanicCatcher("HotPatchMng hot panic")
	val, _ := etcd.Get(etcd.GetHotPatchPathProgressPath(m.option.EtcdRoot, m.option.Gid, m.option.ServiceName, m.option.ServerId))
	info := &etcd.ProgressInfo{}
	json.Unmarshal([]byte(val), info)
	tilogs.L().Debugf("Un marshall value %v, key %s", val, info)
	m.recordProgress(etcd.HotPatchProgress_Read_Version, info.BuildingId, info.UpdateList, firstUpdate, "")
	// 获取版本号
	var hotVersion *etcd.HotPatchVersion
	if verString == "" {
		tilogs.L().Errorf("<HotPatchMng> not ver is empty")
		m.onError(etcd.HotPatchProgress_Read_Version, "数据版本为空", info.BuildingId, info.UpdateList, firstUpdate, "")
		return ""
	} else {
		hotVersion = &etcd.HotPatchVersion{}
		if err := json.Unmarshal([]byte(verString), hotVersion); err != nil {
			tilogs.L().Errorf("<HotPatchMng> json parse version %v", err)
			m.onError(etcd.HotPatchProgress_Read_Version, "解析数据号错误", info.BuildingId, info.UpdateList, firstUpdate, "")
			return ""
		}
	}
	tilogs.L().Infof("HotPatchMng onWatch hotVersion %v", verString)
	if hotVersion.HotPatchBuild == "" { // 摘热更
		clearPatch()
	} else if hotVersion.HotPatchBuild == m.curHotPatchVer {
		tilogs.L().Infof("<HotPatchMng> already data build %d", m.curHotPatchVer)
		m.onError(etcd.HotPatchProgress_OK, "版本号重复", info.BuildingId, info.UpdateList, firstUpdate, info.SourcePath)
		return "" // 已经是当前版本的数据
	} else {
		tilogs.L().Infof("<HotPatchMng> parse new version %v", hotVersion)
		// 从S3加载到本地
		var path string
		// 拷贝数据
		isCopy := false
		if info.SourcePath != "" {
			// /opt/supervisor/gamexxxx/hotpatch/
			shardName := strings.Split(info.SourcePath, "/")[3]
			localShardName := strings.Split(GetWorkPath(), "/")[3]
			if shardName != localShardName {
				isCopy = true
			}
			if isCopy && !info.IsValidSourcePath() {
				tilogs.L().Warnf("HotPatchMng onWatch sourcePath is invalid %s, fallback to download", info.SourcePath)
				isCopy = false
			}
		}
		if isCopy {
			//执行拷贝
			tilogs.L().Infof("<HotPatchMng> Start copy patch from %s", info.SourcePath)
			m.recordProgress(etcd.HotPatchProgress_SoftLink, info.BuildingId, info.UpdateList, firstUpdate, info.SourcePath)
			path = filepath.Join(GetWorkPath(), etcd.DirPatch, fmt.Sprint(hotVersion.HotPatchBuild))
			err := file.CreateSoftLink(info.SourcePath, path)
			if err != nil {
				tilogs.L().Errorf("<HotPatchMng> copy from path %v failed", info.SourcePath)
				m.onError(etcd.HotPatchProgress_SoftLink, "拷贝热更数据错误", info.BuildingId, info.UpdateList, firstUpdate, info.SourcePath)
				return fmt.Sprintf("%s:copy patch error", hotVersion.HotPatchBuild)
			}
		} else {
			m.recordProgress(etcd.HotPatchProgress_Downloading, info.BuildingId, info.UpdateList, firstUpdate, info.SourcePath)
			path = m.loadFromS3(hotVersion.HotPatchBuild)
			if path == "" {
				tilogs.L().Errorf("<HotPatchMng> load from s3 failed")
				m.onError(etcd.HotPatchProgress_Downloading, "下载热更数据错误", info.BuildingId, info.UpdateList, firstUpdate, "")
				return fmt.Sprintf("%s:s3 error", hotVersion.HotPatchBuild)
			}
			info.SourcePath = path
		}
		lastVer := m.curHotPatchVer

		// 热更新到内存
		m.recordProgress(etcd.HotPatchProgress_Loading, info.BuildingId, info.UpdateList, firstUpdate, info.SourcePath)
		clearPatch() // 这里会先清理之前的patch，所以若新patch失败了，则需要还原老patch
		err := reLoadPatch(path)
		if err != nil {
			tilogs.L().Errorf("<HotPatchMng> load patch data error %s", err)
			m.onError(etcd.HotPatchProgress_Loading, "加载patch错误", info.BuildingId, info.UpdateList, firstUpdate, info.SourcePath)
			// 还原。若还原还失败，则清理所有patch，并清空etcd的current
			if lastVer != "" {
				lastPath := filepath.Join(GetWorkPath(), etcd.DirPatch, fmt.Sprint(lastVer))
				if err := reLoadPatch(lastPath); err != nil {
					tilogs.L().Errorf("HotPatchMng reLoadPatch again still fail, path %s err %v", lastPath, err)
					if err := etcd.Put(m.getCurrentKey(), ""); err != nil {
						tilogs.L().Errorf("HotPatchMng reLoadPatch again still fail, etcd.Put err %v", err)
					}
				}
			} else {
				if err := etcd.Put(m.getCurrentKey(), ""); err != nil {
					tilogs.L().Errorf("HotPatchMng clear etcd current, etcd.Put err %v", err)
				}
			}
			return fmt.Sprintf("%s:load error", hotVersion.HotPatchBuild)
		}
	}

	m.recordProgress(etcd.HotPatchProgress_OK, info.BuildingId, info.UpdateList, firstUpdate, info.SourcePath)
	m.curHotPatchVer = hotVersion.HotPatchBuild
	if err := etcd.Put(m.getCurrentKey(), hotVersion.HotPatchBuild); err != nil {
		tilogs.L().Errorf("<HotPatchMng> after reload etcd.Put err %v", err)
	}

	return ""
}

func (m *HotPatchMng) initCloudDb() error {
	m.cloudDb = cloud_db.NewCloudDB(m.option.CloudDbCfg)
	if m.cloudDb == nil {
		err := fmt.Errorf("<HotPatchMng> no cloud db cfg %v", m.option.CloudDbCfg)
		tilogs.L().Errorf("%v", err)
		return err
	}
	if err := m.cloudDb.Open(); err != nil {
		tilogs.L().Errorf("DataHotPatchModule s3.Open err %s", err.Error())
		return err
	}
	return nil
}

func (m *HotPatchMng) loadFromS3(buildId string) string {
	path := filepath.Join(GetWorkPath(), etcd.DirPatch, fmt.Sprint(buildId))
	util.DelFile(path) // 删除同名目录
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		tilogs.L().Errorf("HotPatchMng loadFromS3 MkdirAll %s err %s", path, err.Error())
		return ""
	}
	// s3上获取所有文件列表
	prefix := etcd.GetHotPatchCloudDBPath(version.Version, m.option.ServiceName, buildId)
	tilogs.L().Infof("HotPatchMng loadFromS3 prepare List files %s %s", m.option.Bucket, prefix)
	files, _, err := m.cloudDb.ListObjectWithBucket(m.option.Bucket, "", 100, m.option.CloudDbCfg.CloudDbRoot, prefix)
	if err != nil {
		tilogs.L().Errorf("HotPatchMng loadFromS3 ListObjectWithBucket err %s", err.Error())
		return ""
	}
	tilogs.L().Infof("HotPatchMng loadFromS3 prepare update files %v", files)

	// 读取每个文件并写入本地
	for _, f := range files {
		splitList := strings.Split(f, "/")
		fileName := splitList[len(splitList)-1]
		if fileName == "" {
			continue
		}
		fc, err := m.cloudDb.GetWithBucket(m.option.Bucket, "", f)
		if err != nil {
			tilogs.L().Errorf("HotPatchMng loadFromS3 GetWithBucket err %s", err.Error())
			return ""
		}
		fn := filepath.Join(path, fileName)
		tilogs.L().Infof("HotPatchMng loadFromS3 write file %s, %s", fileName, fn)
		fh, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			tilogs.L().Errorf("HotPatchMng loadFromS3 OpenFile failed %s", err.Error())
			return ""
		}
		err = func() error {
			defer fh.Close()
			if _, err := fh.Write(fc); err != nil {
				tilogs.L().Errorf("HotPatchMng loadFromS3 filewrite failed %s", err.Error())
				return err
			}
			return nil
		}()
		if err != nil {
			tilogs.L().Errorf("HotPatchMng LocalFromS3 File Write failed %s", err.Error())
			return ""
		}
	}
	return path
}

func GetWorkPath() string {
	workPath, _ := os.Getwd()
	workPath, _ = filepath.Abs(workPath)
	// initialize default configurations
	AppPath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	tilogs.L().Debugf("data path work=%s, app=%s", workPath, AppPath)
	appConfigPath := AppPath
	if workPath != AppPath {
		if err := os.Chdir(AppPath); err != nil {
			appConfigPath = workPath
		}
	}
	return appConfigPath
}

const (
	HotPatchEtcdKey_Version  = "version"
	HotPatchEtcdKey_Error    = "error"
	HotPatchEtcdKey_Previous = "previous"
)

func (m *HotPatchMng) getErrorKey() string {
	return m.etcdKeyPrefix + "/" + HotPatchEtcdKey_Error
}

func (m *HotPatchMng) getUpdateKey() string {
	return m.etcdKeyPrefix + "/" + etcd.HotPatchEtcdKey_Update
}

func (m *HotPatchMng) getVersionKey() string {
	return m.etcdKeyPrefix + "/" + HotPatchEtcdKey_Version
}

func (m *HotPatchMng) getCurrentKey() string {
	return m.etcdKeyPrefix + "/" + etcd.HotPatchEtcdKey_Current
}

func (m *HotPatchMng) getPreviousKey() string {
	return m.etcdKeyPrefix + "/" + HotPatchEtcdKey_Previous
}

func (m *HotPatchMng) onError(progress *etcd.HotFixProgress, errDesc string, buildingId string, serverList []string, firstUpdate bool, sourcePath string) {
	if firstUpdate {
		return
	}
	etcd.SetHotPatchProgressError(etcd.GetHotPatchPathProgressPath(m.option.EtcdRoot, m.option.Gid, m.option.ServiceName, m.option.ServerId), m.option.ServiceName, progress.GetTag(), errDesc, buildingId, serverList, firstUpdate, sourcePath)
}

func (m *HotPatchMng) recordProgress(progress *etcd.HotFixProgress, buildingId string, serverList []string, firstUpdate bool, sourcePath string) {
	if firstUpdate {
		return
	}
	etcd.SetHotPatchProgress(etcd.GetHotPatchPathProgressPath(m.option.EtcdRoot, m.option.Gid, m.option.ServiceName, m.option.ServerId), m.option.ServiceName, progress.GetTag(), buildingId, serverList, firstUpdate, sourcePath)
}

// -----------------------------------------------------------------

func reLoadPatch(path string) (err error) {
	defer func() {
		if err != nil {
			clearPatch()
		}
	}()
	allFiles, err := ioutil.ReadDir(path)
	if err != nil {
		tilogs.L().Errorf("HotPatch loadPatch ReadDir err %s", err.Error())
		return err
	}
	patchFiles := make([]string, 0, 8)
	for _, file := range allFiles {
		if file.IsDir() {
			continue
		}

		// patch_xxxxx.so
		if strings.HasPrefix(file.Name(), "patch") && strings.HasSuffix(file.Name(), ".so") {
			patchFiles = append(patchFiles, filepath.Join(path, file.Name()))
		}
	}

	if len(patchFiles) <= 0 {
		err := fmt.Errorf("HotPatch loadPatch not found patch file")
		tilogs.L().Errorf("%v", err)
		return err
	}

	for _, _f := range patchFiles {
		plugin, err := plugin.Open(_f)
		if err != nil {
			tilogs.L().Errorf("HotPatch loadPatch plugin.Open file %s err %v", _f, err)
			return err
		}
		funcList, err := plugin.Lookup("GetHotfixFuncList")
		if err != nil {
			tilogs.L().Errorf("HotPatch loadPatch plugin.Lookup file %s err %v", _f, err)
			return err
		}
		list := funcList.(func() map[string]string)()
		funcMap := map[string]uintptr{}
		//newName 是补丁文件中新的函数名,oldName是需要被替换的旧函数名称
		for newName, oldName := range list {
			newFunc, err := plugin.Lookup(newName)
			if err != nil {
				tilogs.L().Errorf("HotPatch: find newFunc failed.newFuncName=%s,err=%v\n", newName, err)
				return err
			}
			p := getPtr(reflect.ValueOf(newFunc))
			funcMap[oldName] = *(*uintptr)(unsafe.Pointer(p))
		}

		for oldName, newFunc := range funcMap {
			name := C.CString(oldName)
			defer C.free(unsafe.Pointer(name))
			newPtr := unsafe.Pointer(newFunc)

			//调用 patch.h，进行替换。 go里没有原声的 dlsym函数，不能找到原函数的地址
			retCode := C.patch(name, newPtr)
			if retCode != 0 {
				err := fmt.Errorf("HotPatch func fail,oldFuncName=%s.retCode=%v", oldName, retCode)
				return err
			} else {
				tilogs.L().Infof("HotPatch func success,oldFuncName=%s", oldName)
			}
		}
	}
	return nil
}

func clearPatch() {
	C.restore()
}
