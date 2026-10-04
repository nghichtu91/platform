package common

import (
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/gift/model"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
	"github.com/nghichtu91/platform/share/x/gift/modules/common"
)

// ParseRequest 依据reqApi将请求上下文中的请求信息提取。
//
// Param-reqApi: 请求的API; Param-c: gin收到请求的上下文; Param-apiRegistry: 处理方接口的查找注册表。
//
// Return-modules.HandlerInterface: 已填充请求信息的处理结构。
// Return-error: 用于内部显示的异常信息; Return-string: 用于反馈显示的异常信息。
func ParseRequest(reqApi string, c *gin.Context, apiRegistry map[string]reflect.Type) (common.HandlerInterface, error, string) {
	// 获取与reqApi对应的提取类型。
	curType := apiRegistry[reqApi]
	if curType == nil {
		return nil, fmt.Errorf("ParseRequest Handler Unknown reqApi: %v", reqApi), errs.ErrUnknownApiType
	}

	// 实例化一个内容提取实例。
	blank := reflect.New(curType).Interface()
	errBind := c.Bind(blank)
	if errBind != nil {
		return nil, fmt.Errorf("ParseRequest Handler Bind Err: %v", errBind), errs.ErrBindReqData
	}

	// 将填充请求数据的结构体对应转换。
	handler, ok := blank.(common.HandlerInterface)
	if !ok {
		return nil, fmt.Errorf("ParseRequest Handler Parse request type error"), errs.ErrExchangeReqInterface
	}

	return handler, nil, errs.Success
}

// FailCur GiftServer立刻向请求方报告异常。
func FailCur(err error, errMsg string, c *gin.Context) {
	tilogs.L().Errorf("FailCur ERROR: %v", err)

	commonResult := model.CreateDefaultFailResult()
	commonResult.SetMessage(errMsg + ", err:" + err.Error())
	commonResult.SetTag(errMsg)

	c.JSON(model.CodeSuccess, commonResult)
}

// SuccessCur GiftServer立刻向请求方报告成功信息。
func SuccessCur(respData interface{}, c *gin.Context) {
	commonResult := model.CreateDefaultOkResult()
	commonResult.SetData(respData)

	c.JSON(model.CodeSuccess, commonResult)
}
