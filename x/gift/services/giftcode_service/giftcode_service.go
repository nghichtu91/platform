package giftcode_service

import (
	"fmt"
	"reflect"
	"time"

	common2 "github.com/nghichtu91/platform/share/x/gift/services/common"

	"github.com/nghichtu91/platform/share/x/gift/modules/giftcode"
	"github.com/nghichtu91/platform/share/x/gift/modules/giftcode/giftcode_handler"

	"github.com/nghichtu91/platform/share/x/gift/modules/common"

	"github.com/nghichtu91/platform/share/x/gift/model/api"

	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/gift/config"
)

// 模块受理的API请求。
var apiRegistry map[string]reflect.Type

// 注册模块受理的API请求。
func Service(g *gin.Engine) error {
	// 初始化注册表。
	apiRegistry = make(map[string]reflect.Type, 32)

	// 以模块名作为gin的服务分组.
	group := g.Group("/" + giftcode.ModuleGiftCode)

	// 获取受理API请求的模块。
	module := giftcode.GetModule()
	if module == nil {
		return fmt.Errorf("service [%s] module is nil", giftcode.ModuleGiftCode)
	}

	/*
		注册准许受理API。
	*/
	// 生成兑换码。
	if err := regPostApi(group, api.GenGiftCode, &giftcode_handler.GenGiftCodeRequest{}, module); err != nil {
		return fmt.Errorf("service [%s] err: %v", giftcode.ModuleGiftCode, err)
	}
	// 追加兑换码。
	if err := regPostApi(group, api.AddGiftCode, &giftcode_handler.AddGiftCodeRequest{}, module); err != nil {
		return fmt.Errorf("service [%s] err: %v", giftcode.ModuleGiftCode, err)
	}
	// 下载兑换码。
	if err := regPostApi(group, api.DownloadGiftCode, &giftcode_handler.DownloadGiftCodeRequest{}, module); err != nil {
		return fmt.Errorf("service [%s] err: %v", giftcode.ModuleGiftCode, err)
	}
	// 查询已生成兑换码。
	if err := regPostApi(group, api.QueryGenGiftCode, &giftcode_handler.QueryGenInfoRequest{}, module); err != nil {
		return fmt.Errorf("service [%s] err: %v", giftcode.ModuleGiftCode, err)
	}
	// 查询兑换码。
	if err := regPostApi(group, api.QueryGift, &giftcode_handler.QueryGiftCodeRequest{}, module); err != nil {
		return fmt.Errorf("service [%s] err: %v", giftcode.ModuleGiftCode, err)
	}
	// 更新兑换码。
	if err := regPostApi(group, api.UpdateGift, &giftcode_handler.UpdateGiftCodeRequest{}, module); err != nil {
		return fmt.Errorf("service [%s] err: %v", giftcode.ModuleGiftCode, err)
	}
	// 销毁兑换码。
	if err := regPostApi(group, api.DestroyGiftCode, &giftcode_handler.DestroyGiftCodeRequest{}, module); err != nil {
		return fmt.Errorf("service [%s] err: %v", giftcode.ModuleGiftCode, err)
	}
	// 领取兑换码。
	if err := regPostApi(group, api.ClaimGiftCode, &giftcode_handler.ClaimGiftCodeRequest{}, module); err != nil {
		return fmt.Errorf("service [%s] err: %v", giftcode.ModuleGiftCode, err)
	}
	// 获取GidS。
	if err := regPostApi(group, api.QueryLA, &giftcode_handler.QueryLARequest{}, module); err != nil {
		return fmt.Errorf("service [%s] err: %v", giftcode.ModuleGiftCode, err)
	}
	// 操作GidS
	if err := regPostApi(group, api.UpdateLA, &giftcode_handler.UpdateLARequest{}, module); err != nil {
		return fmt.Errorf("service [%s] err: %v", giftcode.ModuleGiftCode, err)
	}

	return nil
}

// 注册受理的API POST请求。
func regPostApi(rg *gin.RouterGroup, reqApi string, handlerType common.HandlerInterface, gcm *giftcode.Module) error {
	// 检查受理API是否已被注册。
	if _, ok := apiRegistry[reqApi]; ok {
		return fmt.Errorf("regPostApi RegApi [%s] Repeated: %v", giftcode.ModuleGiftCode, reqApi)
	}

	// 向注册表中添加API对应的处理类型信息。
	apiRegistry[reqApi] = reflect.TypeOf(handlerType).Elem()

	// 向路由分组中注册API。
	rg.POST(reqApi, func(c *gin.Context) {
		// 开始处理的时间戳。
		startTime := time.Now().UnixNano()

		tilogs.L().Debugf("GetReq api: [%v]", reqApi)

		// 获取上下文中的请求信息。
		handlerInfo, errParse, errParseMsg := common2.ParseRequest(reqApi, c, apiRegistry)
		if errParse != nil {
			common2.FailCur(errParse, errParseMsg, c)
			return
		}

		// 对应模块受理请求。
		respData, errHandle, errHandleMsg := gcm.Handle(reqApi, handlerInfo)

		tilogs.L().Debugf("Handle Req api: [%v]", reqApi)
		if errHandle != nil {
			common2.FailCur(errHandle, errHandleMsg, c)
			return
		}

		// 立刻向请求方反馈成功信息。
		common2.SuccessCur(respData, c)

		// 成功受理的内部提示信息。
		tilogs.L().Debugf("API is [%v], Handler [%v] costTime [%v]", reqApi, handlerInfo, (time.Now().UnixNano()-startTime)/config.Millisecond)
	})

	return nil
}
