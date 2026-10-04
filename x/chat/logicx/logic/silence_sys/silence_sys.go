package silence_sys

import (
	"github.com/nghichtu91/platform/share/planx/servers"
	"github.com/nghichtu91/platform/share/planx/silence_sys"
)

var instance *silence_sys.SilenceSys

func NewSilenceSysModule(gid, sid uint32) servers.Module {
	instance = silence_sys.NewSilenceSysManager(gid, sid)
	return instance
}

func GetModule() *silence_sys.SilenceSys {
	return instance
}

func StopModule() {
	instance.Stop()
}
