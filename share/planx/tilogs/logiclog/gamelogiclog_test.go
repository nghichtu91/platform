package logiclog

import (
	"testing"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"

	"github.com/nghichtu91/platform/share/planx/util"
)

func TestLogicLogConfig(t *testing.T) {
	zaplog.InitZapLog("", map[string]string{}, "")
	defer tilogs.Close()
	LoadGameLogic("logiclog.toml")
}

func TestLogicLog(t *testing.T) {
	zaplog.InitZapLog("", map[string]string{}, "")
	defer tilogs.Close()
	w := util.WaitGroupWrapper{}
	w.Wrap(func() {
		//LoadGameLogic()


		//initGameLogicLog("./logiclog_%y-%m-%d_%h.log",
		//	"Asia/Shanghai",
		//	bilog.DayRotate, "logiclog.log")
		//initGameLogicLevelLog(errorLevel,
		//	"./logiclog_error_%y-%m-%d_%h.log",
		//	"Asia/Shanghai",
		//	bilog.DayRotate, "logiclog.log")
	})
	time.Sleep(time.Second)

	w.Wrap(func() {
		LogGameLogicTrace("test", &Common{
			Gid:         1,
			Sid:         1,
			Channel:     "channel",
			Uid:         "asdfadsf",
			DeviceId:    "dev",
			AccountID:   "acid",
			PlayerName:  "name",
			PlayerLevel: 1,
			VIP:         1,
		}, struct {
			Name string
			age  int
		}{"name", 18}, "last")
		LogGameLogicDebug("test", &Common{
			Gid:         1,
			Sid:         1,
			Channel:     "channel",
			Uid:         "asdfadsf",
			DeviceId:    "dev",
			AccountID:   "acid",
			PlayerName:  "name",
			PlayerLevel: 1,
			VIP:         1,
		}, struct {
			Name string
			age  int
		}{"name", 18}, "last")
		LogGameLogicInfo("test", &Common{
			Gid:         1,
			Sid:         1,
			Channel:     "channel",
			Uid:         "asdfadsf",
			DeviceId:    "dev",
			AccountID:   "acid",
			PlayerName:  "name",
			PlayerLevel: 1,
			VIP:         1,
		}, struct {
			Name string
			age  int
		}{"name", 18}, "last")
		LogGameLogicWarn("test", &Common{
			Gid:         1,
			Sid:         1,
			Channel:     "channel",
			Uid:         "asdfadsf",
			DeviceId:    "dev",
			AccountID:   "acid",
			PlayerName:  "name",
			PlayerLevel: 1,
			VIP:         1,
		}, struct {
			Name string
			age  int
		}{"name", 18}, "last")
		LogGameLogicError("test", &Common{
			Gid:         1,
			Sid:         1,
			Channel:     "channel",
			Uid:         "asdfadsf",
			DeviceId:    "dev",
			AccountID:   "acid",
			PlayerName:  "name",
			PlayerLevel: 1,
			VIP:         1,
		}, struct {
			Name string
			age  int
		}{"name", 18}, "last")
	})
	w.Wait()
	CloseGameLogicLog()
}
