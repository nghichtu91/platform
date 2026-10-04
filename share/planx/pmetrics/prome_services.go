package pmetrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/nghichtu91/platform/share/planx/config"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var (
	jobPrefix          string        // Job的昵称，用于区分不同项目和服务器。
	stopPromethuesChan chan struct{} // 用于终止Promethues的channel。

	jobConfig PromeConfig // Promethues相关配置。
)

func init() {
	stopPromethuesChan = make(chan struct{})
}

// StartPromethues 开启普罗米修斯指标监控。
//
// Des:服务器向PushGateWay推送相关Metrics，Promethues周期性从PushGateWay刮取Metrics。
//
// Param-cfgName：配置文件名，Param-projName：项目名，Param-prefix：服务器信息。
//
// Return-error: 异常信息，当Promethues启动异常时，需要终止起服。
func StartPromethues(cfgName, projName, prefix string) error {

	// 服务器以Job为单位向PushGateWay推送指标时，使用的Job标识（项目名+服务器信息）。
	jobPrefix = fmt.Sprintf("%s_%s", projName, prefix)

	// 读取Toml配置文件，启动服务，并将加载配置加入ReloadHandler中。
	if config.NewConfig(cfgName, true, readConfigHandler) == nil {
		return fmt.Errorf("<Promethues> NewConfig %s failed", cfgName)
	}

	return nil
}

// startPushMetrics 开始Promethues指标推送（服务器推送至PushGateWay，Promethues从PushGateWay刮取）。
//
// Param-stopChannel:通知停止服务的channel。
//
// Return-error:启动Metrics推送服务器时的报错信息。
func startPushMetrics(stopChannel <-chan struct{}) error {
	tilogs.L().Infof("<Promethues> StartPushMetrics")

	// 是否开启向PushGateWay推送服务器相关的Collector。
	if !jobConfig.PushSpecialOpen {
		tilogs.L().Infof("<Promethues> StartPushMetrics cancel : PushSpecialOpen is false.")
		return nil
	}

	// 如果不配置推送网关地址，视为不允许启动。
	if jobConfig.PushGateWayHost == "" {
		tilogs.L().Infof("<Promethues> StartPushMetrics cancel : PushGateWayHost is \"\" value.")
		return nil
	}

	// 为防止PushGateWay的Metrics混乱，必须设置jobPrefix。
	if jobPrefix == "" {
		tilogs.L().Infof("<Promethues> StartPushMetrics cancel : JobPrefix is \"\" value.")
		return nil
	}

	// 判断、开启服务器进程自身相关的周期性推送。
	if err := pushServerProcess(stopChannel); err != nil {
		return err
	}

	// 判断、开启服务器进程操作系统相关的周期性推送（即使配置为开启，也仅对linux类和windows操作系统生效）。
	if err := pushServerOS(stopChannel); err != nil {
		return err
	}

	// 开启StateRegistry与CustomStateRegistry的周期性推送。
	pushStateRegistry(stopChannel)

	return nil
}

// pushServerProcess 判断、开启服务器进程自身相关的周期性推送。
//
// Param-stopChannel:通知停止服务的channel。
func pushServerProcess(stopChannel <-chan struct{}) error {
	if !jobConfig.PushServerProcessOpen {
		return nil
	}

	// 注册prometheus提供的进程收集Collector。
	registry := prometheus.NewPedanticRegistry()
	if err := registry.Register(prometheus.NewGoCollector()); err != nil {
		return err
	}

	pushTick := time.NewTicker(time.Duration(jobConfig.PushProcessIntervalTime))
	pusher := push.New(jobConfig.PushGateWayHost, jobPrefix+ProcessPush).Gatherer(registry)
	go func() {
		tilogs.L().Debugf("<Promethues> StartPushMetrics pushServerProcess PushStart")
		for {
			select {
			case <-stopChannel:
				tilogs.L().Debugf("<Promethues> StartPushMetrics pushServerProcess PushStop")
				pushTick.Stop()
				return
			case <-pushTick.C:
				if err := pusher.Add(); err != nil {
					tilogs.L().Warnf("<Promethues> pushServerProcess Push ERR: %s", err.Error())
				}
			}
		}
	}()

	return nil
}

// pushServerOS 判断、开启服务器进程操作系统相关的周期性推送。
func pushServerOS(stopChannel <-chan struct{}) error {
	if !jobConfig.PushServerOSOpen {
		return nil
	}

	// 注册prometheus提供的系统收集Collector。
	registry := prometheus.NewPedanticRegistry()
	if err := registry.Register(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{})); err != nil {
		return err
	}

	pushTick := time.NewTicker(time.Duration(jobConfig.PushProcessIntervalTime))
	pusher := push.New(jobConfig.PushGateWayHost, jobPrefix+OSPush).Gatherer(registry)
	go func() {
		tilogs.L().Debugf("<Promethues> StartPushMetrics pushServerOS PushStart")
		for {
			select {
			case <-stopChannel:
				tilogs.L().Debugf("<Promethues> StartPushMetrics pushServerOS PushStop")
				pushTick.Stop()
				return
			case <-pushTick.C:
				if err := pusher.Add(); err != nil {
					tilogs.L().Warnf("<Promethues> pushServerOS Push ERR: %s", err.Error())
				}
			}
		}
	}()

	return nil
}

// pushStateRegistry 开启StateRegistry的周期性推送。
//
// Param-stopChannel:通知停止服务的channel。
func pushStateRegistry(stopChannel <-chan struct{}) {
	pushTick := time.NewTicker(time.Duration(jobConfig.PushSpecialIntervalTime))
	// 将StateRegistry（当前两种）的Collector推送至PushGateWay的Pusher。
	pusher := push.New(jobConfig.PushGateWayHost, jobPrefix).Gatherer(serverStateRegistry)
	go func() {
		tilogs.L().Debugf("<Promethues> StartPushMetrics pushStateRegistry PushStart")
		for {
			select {
			case <-stopChannel:
				tilogs.L().Debugf("<Promethues> StartPushMetrics pushStateRegistry PushStop")
				pushTick.Stop()
				return
			case <-pushTick.C:
				if err := pusher.Add(); err != nil {
					tilogs.L().Warnf("<Promethues> StateRegistry pushStateRegistry Push ERR: %s", err.Error())
				}
			}
		}
	}()
}

// Stop 终止Promethues监控服务。
func Stop() {
	close(stopPromethuesChan)
}

// 消息处理时间独立推送
// Deprecated: TimeStatisticsAlonePush implement jsonlog instead.
//func TimeStatisticsAlonePush(msgName string, lastNanoTime int64) {
//	handleTime := time.Now().UnixNano() - lastNanoTime
//	promethuesAlonePush(msgName, fmt.Sprintf("%s_time", msgName), handleTime)
//}

// 消息处理计数独立推送
// Deprecated: CountStatisticsAlonePush implement jsonlog instead.
//func CountStatisticsAlonePush(msgName string) {
//	promethuesAlonePush(msgName, fmt.Sprintf("%s_count", msgName), 1)
//}

// 消息内容大小独立推送
// Deprecated: SizeStatisticsAlonePush implement jsonlog instead.
//func SizeStatisticsAlonePush(msgName string, msgSize int64) {
//	promethuesAlonePush(msgName, fmt.Sprintf("%s_size", msgName), msgSize)
//}

// 执行普罗米修斯独立推送（相同Job和Label推送视为不同的Metric）。
// Des:服务器向PushGateWay推送相关Metrics，Metrics会拼接递增的globalAloneCount来保证旧metrics不被替换。
// msgName：消息名（带request、respones），typeName：全名（msgName+统计类别），value：Metrics值。
// Deprecated: promethuesAlonePush implement elasticsearch instead.
//func promethuesAlonePush(msgName string, typeName string, value int64) {
//	globalLock.Lock()
//	defer globalLock.Unlock()
//	fullName := typeName + fmt.Sprintf("_%v_%v", time.Now().UnixNano(), globalCount)
//	pusher := push.New(jobConfig.PushGateWayHost, jobPrefix).Collector(NewPushAloneGuage(fullName, value))
//	pusher.Grouping("Message", msgName)
//	if err := pusher.Add(); err != nil {
//		tilogs.L().Warnf("<Promethues> promethuesAlonePush PushGateWay ERR :%v", err)
//	}
//	globalCount++
//	if globalCount == math.MaxUint8 {
//		globalCount = 0
//	}
//}
