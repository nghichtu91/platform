package logic

import (
	_ "database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/nghichtu91/platform/share/x/chat/web/db"
	"github.com/nghichtu91/platform/share/x/chat/web/model/command"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"

	"github.com/nghichtu91/platform/share/planx/arrayhelper"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/x/chat/web/model/errs"
)

/*
*
前端和后端通信的接口
*/
type RequestInterface interface {
	Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string)
}

type GmResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
	// Msg     string      `json:"msg"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

// 如果response需要下载文件， 需要实现这个接口
type DownloadResponse interface {
	GetFileName() string
	GetData() []byte
}

// 如果该条请求不需要记录， 需要实现这个接口， 默认是记录的， 防止有纰漏
type NeedRecordResponse interface {
	NeedRecord() bool
}

func OnFailed(resp *GmResponse, msg string) {
	resp.Msg = msg
	resp.Success = false
	tilogs.L().Errorf(msg)
}

// error 用于服务器内部打印日志 msg 用于前端显示
func FailNow(error error, msg string, c *gin.Context) {
	tilogs.L().Errorf("fail now %v", error)
	resp := &GmResponse{}
	OnFailed(resp, msg)
	ret, err := json.Marshal(resp)
	if err != nil {
		c.String(200, err.Error())
	} else {
		c.String(200, string(ret))
	}
}

func OnSuccess(resp *GmResponse) {
	resp.Msg = "Success"
	resp.Success = true
}

type HandlerFunc func(req interface{}) (interface{}, error)

var typeRegistry map[string]reflect.Type

func init() {
	typeRegistry = make(map[string]reflect.Type, 64)
}

type LoginName struct {
	Name string `form:"username"`
	Pwd  string `form:"password"`
}

func Handler(loginToken, reqType string, c *gin.Context) {
	startTime := time.Now().UnixNano()
	handler, err, errMsg, reqJson := parseRequest(reqType, c)
	if err != nil {
		FailNow(err, errMsg, c)
		return
	}

	clientIp := c.ClientIP()

	session := sessions.Default(c)
	userName := session.Get(loginToken)
	if reqType == command.GmCommandLogin {
		info := LoginName{}
		err := json.Unmarshal(reqJson, &info)
		userName = info.Name
		if err != nil {
			tilogs.L().Errorf("get LoginName err %v", err)
			return
		}
	}

	var userInfo super_user.UserInfo

	if userName == nil {
		userInfo.UserName = ""
	} else {
		userInfo.UserName = userName.(string)
	}
	// 匹配每个路由的处理函数
	respData, err, errMsg := handler.Handle(c, userInfo)
	if err != nil {
		FailNow(err, errMsg, c)
		return
	}

	needRecord := processResponse(c, reqType, respData)

	if needRecord {
		// TODO
		reqStr := string(reqJson)
		if reqType == command.GmCommandLogin {
			reqStr = userName.(string)
		}
		_, err := db.GetDB().Exec("insert into operation_log(user, ip, time, operation_type, operation_detail) values (?, ?, ?, ?, ?)", userName, clientIp, time.Now().Unix(), reqType, reqStr)
		if err != nil {
			tilogs.L().Errorf("%v", err)
			return
		}

	}
	processTime := (time.Now().UnixNano() - startTime) / 1000000
	tilogs.L().Debugf("[%s][%d][resp] %v [reqJson] %v", reqType, processTime, respData, reqJson)
}

func parseRequest(reqType string, c *gin.Context) (RequestInterface, error, string, []byte) {
	myType := typeRegistry[reqType]
	if myType == nil {
		return nil, fmt.Errorf("bad type %s", reqType), errs.BadType, nil
	}
	req := reflect.New(myType).Interface()
	err := c.Bind(req)
	if err != nil {
		return nil, err, errs.SERVER, nil
	}

	reqJson, err := json.Marshal(req)
	if err != nil {
		return nil, err, errs.SERVER, nil
	}
	tilogs.L().Debugf("[%s][req] %v", reqType, string(reqJson))

	handler, ok := req.(RequestInterface)
	if !ok {
		return nil, fmt.Errorf("not request interface"), errs.SERVER, nil
	}
	return handler, nil, "", reqJson
}

var ignoreRecordCommand = []string{
	command.GmCommandToken,
	command.GmCommandGetOneUser,
	// more ignoreRecord
}

func processResponse(c *gin.Context, reqType string, respData interface{}) bool {
	if downResp, ok := respData.(DownloadResponse); ok {
		c.Header("Content-Disposition", fmt.Sprintf("attachment;fileName=%s", downResp.GetFileName()))
		c.Data(200, "multipart/form-data", downResp.GetData())
		return true
	}

	needRecord := true
	if arrayhelper.ContainsString(ignoreRecordCommand, reqType) {
		needRecord = false
	}

	resp := &GmResponse{}
	resp.Success = true
	resp.Msg = "成功"
	resp.Data = respData
	retJson, err := json.Marshal(resp)
	if err != nil {
		tilogs.L().Errorf("resp json failed %v", err)
		return needRecord
	}
	retStr := string(retJson)
	c.String(200, retStr)

	return needRecord
}
