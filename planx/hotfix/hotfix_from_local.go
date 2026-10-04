package hotfix

// #cgo CFLAGS: -g -O -Wall
// #cgo LDFLAGS: -ldl
// #include <stdlib.h>
// #include "patch.h"
import "C"
import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"plugin"
	"reflect"
	"strings"
	"unsafe"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var lastFileMd5 string
var curPluginLoaded *plugin.Plugin

//尝试HotFix，patchFile是动态库，用go build -buildmode=plugin生成的
func TryHotfixLocal() {
	allFiles, err := ioutil.ReadDir(".")
	if err != nil {
		clean()
		return
	}

	var patchFile string = ""
	for _, file := range allFiles {
		if file.IsDir() {
			continue
		}

		// patch_xxxxx.so
		if strings.HasPrefix(file.Name(), "patch") && strings.HasSuffix(file.Name(), ".so") {
			patchFile = file.Name()
			break
		}
	}

	if patchFile == "" {
		clean()
		return
	}

	file, err := os.Open(patchFile)
	if err != nil {
		//取消所有的补丁
		clean()
		return
	}
	defer file.Close()

	md5h := md5.New()
	io.Copy(md5h, file)
	md5Str := hex.EncodeToString(md5h.Sum(nil))
	if md5Str == lastFileMd5 {
		//只Patch一次
		return
	}

	if curPluginLoaded != nil {
		//先把上次的clean
		clean()
	}

	plugin, err := plugin.Open(patchFile)
	if err != nil {
		tilogs.L().Errorf("TryHotfixLocal failed. err=%v\n", err)
		return
	}

	funcList, err := plugin.Lookup("GetHotfixFuncList")
	if err != nil {
		tilogs.L().Errorf("TryHotfixLocal: find GetHotfixFuncList failed. err=%v\n", err)
		return
	}

	list := funcList.(func() map[string]string)()
	funcMap := map[string]uintptr{}

	//newName 是补丁文件中新的函数名,oldName是需要被替换的旧函数名称
	for newName, oldName := range list {
		newFunc, err := plugin.Lookup(newName)
		if err != nil {
			tilogs.L().Errorf("TryHotfixLocal: find newFunc failed.newFuncName=%s,err=%v\n", newName, err)
			return
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
			tilogs.L().Errorf("patch func fail,oldFuncName=%s.retCode=%d\n", oldName, retCode)
		} else {
			tilogs.L().Infof("patch func success,oldFuncName=%s.\n", oldName)
		}
	}

	lastFileMd5 = md5Str
	curPluginLoaded = plugin

	tilogs.L().Infof("hotfix success,patchFile=%s.\n", patchFile)
}

type value struct {
	_   uintptr
	ptr unsafe.Pointer
}

func getPtr(v reflect.Value) unsafe.Pointer {
	return (*value)(unsafe.Pointer(&v)).ptr
}

func clean() {
	lastFileMd5 = ""
	curPluginLoaded = nil
	C.restore()
}
