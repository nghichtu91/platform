package cheat

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/gin-gonic/gin"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

// 参数
type CheatParam struct {
	CmdTyp   string   `json:"ct"`
	Acid     string   `json:"aid"` // acid或玩家name
	StrParam []string `json:"strps"`
	IntParam []int64  `json:"intps"`
}

// 返回值
type CheatRetParam struct {
	Success bool       `json:"sucs"`
	Msg     string     `json:"msg"`
	CSVInfo [][]string `json:"csv_info"`
}

type CheatHandler func(param *CheatParam) *CheatRetParam

var (
	cheatHandler CheatHandler
)

func InitCheat(etcdServer string, gid, sid uint, cheatEnable, debugAddr string, lis net.Listener, handler CheatHandler) bool {
	if planx.IsCheatEnable(cheatEnable) {
		cheatHandler = handler
		router := gin.Default()
		router.POST("/debug", func(c *gin.Context) {
			data, err := c.GetRawData()
			if err != nil {
				tilogs.L().Errorf("Cheat GetRawData err %s", err.Error())
				c.JSON(200, CheatRetParam{
					Success: false,
					Msg:     "参数解析错误",
				})
				return
			}
			param := &CheatParam{}
			if err := json.Unmarshal(data, param); err != nil {
				tilogs.L().Errorf("Cheat param json.Unmarshal err %s, %v", err.Error(), string(data))
				c.JSON(200, CheatRetParam{
					Success: false,
					Msg:     "参数解析错误",
				})
				return
			}

			tilogs.L().Debugf("cheat got param %v", param)
			ret := cheatHandler(param)
			c.JSON(200, ret)
			return
		})
		chanInternalErr := make(chan error, 1)
		go func(c chan error) {
			if err := http.Serve(lis, router); err != nil {
				tilogs.L().Errorf("http.Serve %s err %s", debugAddr, err.Error())
				c <- err
			}
		}(chanInternalErr)
		if !util.CheckGoroutineStartErr(chanInternalErr, "http.Serve %s", debugAddr) {
			return false
		}

		tilogs.L().Infof("start Cheat addr %s", debugAddr)

		if err := etcd.Put(cheatKeyPath(etcdServer, gid, sid),
			fmt.Sprintf("http://%s/debug", debugAddr)); err != nil {
			tilogs.L().Errorf("InitCheat etcd.Put err %s", err.Error())
		} else {
			tilogs.L().Infof("InitCheat debugAddr %s", debugAddr)
		}
	}
	return true
}

func cheatKeyPath(serverRoot string, gid, sid uint) string {
	return fmt.Sprintf("%s/%d/%s/%d/%s", serverRoot, gid, etcd.GamexDirs, sid, etcd.KeyDebugAddr)
}

func UnRegCheatAddr(etcdServer, cheatEnable string, gid, sid uint) {
	if planx.IsCheatEnable(cheatEnable) {
		cheatKey := cheatKeyPath(etcdServer, gid, sid)
		delErr := etcd.Delete(cheatKey)
		if delErr != nil {
			tilogs.L().Errorf("delete %v error %v", cheatKey, delErr)
		} else {
			tilogs.L().Infof("delete %v success", cheatKey)
		}
	}
}
