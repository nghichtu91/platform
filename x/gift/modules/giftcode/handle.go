package giftcode

import (
	"context"
	"fmt"

	"github.com/nghichtu91/platform/share/x/gift/config"
	"github.com/nghichtu91/platform/share/x/gift/model/api"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
	"github.com/nghichtu91/platform/share/x/gift/modules/common"
)

// Handle GiftCode模块对外的统一处理函数。
// 负责将gin给与的请求封装后路由到不同的channel池中。
func (mo *Module) Handle(reqApi string, handler common.HandlerInterface) (interface{}, error, string) {
	return gfCommon(reqApi, handler)
}

// gfCommon GiftCode通用受理Chan分发逻辑。
func gfCommon(reqApi string, handler common.HandlerInterface) (interface{}, error, string) {
	var ctx context.Context
	var cancel context.CancelFunc
	var subModule *common.CommonSubModule

	switch reqApi {
	case api.AddGiftCode, api.GenGiftCode, api.QueryGenGiftCode, api.DownloadGiftCode,
		api.QueryGift, api.QueryLA, api.UpdateLA:
		ctx, cancel = context.WithTimeout(context.Background(), config.TimeConsumeReqTimeOut)
		subModule = GetModule().GetIndependentSubModule()
	case api.UpdateGift, api.DestroyGiftCode, api.ClaimGiftCode:
		ctx, cancel = context.WithTimeout(context.Background(), config.DefaultReqTimeOut)
		subModule = GetModule().GetCommonSubModule()
	default:
		return nil, fmt.Errorf("[%v Module] Handle API: %v, reg but not Acceptance", ModuleGiftCode, reqApi), errs.ErrRegButNotAcceptance
	}

	defer cancel()

	// 获取Handler的Hash依据。
	hashNum, err, errMsg := handler.GetDistinguish()
	if err != nil {
		return nil, err, errMsg
	}

	// 获取对应子模块的Chan的Index。
	hashIndex := gmCommonHash(hashNum, subModule.GetThreadsNum())

	// 创建用于装载信息的Cmd。
	cmd := &common.Cmd{
		ReqAPI:  reqApi,
		ReqInfo: handler,
	}

	// 初始化resChan用于同步操作。
	resChan := make(chan *common.Ret, 1)
	cmd.ResChan = resChan

	// 处理Chan已满的情况。
	select {
	case subModule.GetChanPool()[hashIndex] <- cmd:
	case <-ctx.Done():
		return nil, fmt.Errorf("[%v Module] Handle API: %v err, [%v] chan has full", ModuleGiftCode, reqApi, hashIndex), errs.ErrHashChannelFull
	}

	// 处理处理时间超时的情况。
	select {
	case res := <-resChan:
		return res.Data, res.Err, res.Msg
	case <-ctx.Done():
		return nil, fmt.Errorf("[%v Module] Handle API: %v err, [%v] handle time out", ModuleGiftCode, reqApi, hashIndex), errs.ErrHashChannelTimeout
	}
}

// gmCommonHash 获取被Hash到的Chan的Index。
//
// Param-HashNum: Hash依据; Param-thNum: 受理子模块的服务线程数（等于Chan数）。
func gmCommonHash(HashNum int, thNum int) int {
	return HashNum % thNum
}
