package util

import (
	"errors"
	"fmt"

	"github.com/astaxie/beego/httplib"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

var WaitGroup util.WaitGroupWrapper

type CommonResult struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func PostGMServer(api string, obj, res interface{}) error {
	r := &CommonResult{
		Data: res,
	}
	//url := config.Cfg.ChatServer // 改为从本地配置读取
	url := "" // 改为从本地配置读取
	key := fmt.Sprintf("%s%s", url, api)
	req, err := httplib.Post(key).JSONBody(obj)
	tilogs.L().Infof("%v", req)
	if err != nil {
		tilogs.L().Errorf("PostGMServer http lib Post err %v", err)
		return err
	}

	err = req.ToJSON(r)
	if err != nil {
		tilogs.L().Errorf("PostGMServer ToJSON err %v", err)
		return err
	}
	tilogs.L().Debugf("信息%v", r.Message)
	tilogs.L().Debugf("代码%v", r.Code)
	if r.Code != 200 {
		return errors.New(r.Message)
	}
	return nil
}
