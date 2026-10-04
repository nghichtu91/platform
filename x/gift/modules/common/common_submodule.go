package common

import (
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

// 通用受理子模块。
// 当请求方已经携带了所有的信息可以抽象出接口 HandlerInterface，或者公共信息为全局信息时可以使用这一模块。
type CommonSubModule struct {
	chanPool []chan *Cmd // chan列表，接收的请求在上层Hash到不同的chan中。

	moduleName    string // 模块标识符（用于Log输出）。
	subModuleName string // 子模块标识（用于Log输出）。
	thNum         int    // 模块的服务线程数。
	bufferNum     int    // 模块的chan缓冲数。

	quit chan struct{} // 控制协程终止的chan
}

// CreateCommonSubModule 创建通用受理子模块。
//
// Param-thNum: 当前模块的处理线程（chan）数; Param-bufferNum: 每个chan的缓冲数目。
// Param-moduleName: 模块标识符（用于日志输出）;Param-subModuleName: 子模块标识符（用于日志输出）。
//
// Return-*CommonSubModule: 已初始化的通用子模块。
func CreateCommonSubModule(thNum int, bufferNum int, moduleName string, subModuleName string) *CommonSubModule {
	// 初始化Channel池。
	pool := make([]chan *Cmd, 0, thNum)
	for index := 0; index < thNum; index++ {
		pool = append(pool, make(chan *Cmd, bufferNum))
	}

	return &CommonSubModule{
		chanPool:      pool,
		moduleName:    moduleName,
		subModuleName: subModuleName,
		thNum:         thNum,
		bufferNum:     bufferNum,
		quit:          make(chan struct{}, 1),
	}
}

// GetThreadsNum 获取子模块的处理协程数目。
func (cc *CommonSubModule) GetThreadsNum() int {
	return cc.thNum
}

// GetSubModuleName 获取子模块的标识符。
func (cc *CommonSubModule) GetSubModuleName() string {
	return cc.subModuleName
}

// GetChanPool 获取子模块的Chan池。
func (cc *CommonSubModule) GetChanPool() []chan *Cmd {
	return cc.chanPool
}

// Start 启动通用协程受理池。
func (cc *CommonSubModule) Start(w *util.WaitGroupWrapper) {
	for index := 0; index < cc.thNum; index++ {
		func(tempIndex int) {
			w.Wrap(func() {
				for {
					select {
					case cmd := <-cc.chanPool[tempIndex]:
						func() {
							defer tilogs.PanicCatcher("Module:[%s] subModule:[%s] Handle panic, API is %v", cc.moduleName, cc.subModuleName, cmd.ReqAPI)
							respData, err, errMsg := cmd.ReqInfo.Handle()
							cmd.ResChan <- &Ret{
								Data: respData,
								Err:  err,
								Msg:  errMsg,
							}
						}()
					case <-cc.quit:
						return
					}
				}
			})
		}(index)
	}
}

func (cc *CommonSubModule) Stop() {
	close(cc.quit)
}
