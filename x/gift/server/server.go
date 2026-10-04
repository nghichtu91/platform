package server

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/util"
)

// StartServer 开启服务器。
func StartServer(w *util.WaitGroupWrapper) error {

	// 启动所有模块。
	if err := StartModuleMng(w); err != nil {
		return fmt.Errorf("StartServer err: %v", err)
	}

	return nil
}
