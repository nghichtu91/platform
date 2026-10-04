package allmetrics

import (
	"fmt"
	"strings"
	"sync"
	"time"

	gm "github.com/rcrowley/go-metrics"

	"github.com/nghichtu91/platform/share/planx/3rd_party_api/feishu"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/timeutil"
)

var (
	scFail  map[string]gm.Meter
	scMutex sync.RWMutex
)

// PrefixServerCheckMetrics 前缀
func PrefixServerCheckMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", ServerCheckPrefix, gid, serverId)
}

// InitServerCheckMetrics 初始化
func InitServerCheckMetrics() {
	scFail = make(map[string]gm.Meter, 1024)
}

// ServerCheckFail 增加一次失败统计
func ServerCheckFail(key string) {
	scMutex.RLock()
	m, ok := scFail[key]
	scMutex.RUnlock()
	if !ok {
		m = metrics.NewMeter(key)
		scMutex.Lock()
		scFail[key] = m
		scMutex.Unlock()
	}

	m.Mark(1)
}

// 国内线上时间统计都是以周六一整天的最大值为准
// 阈值取最大时间*2，向上取整，单位ms
const (
	ThresholdTime_CN_Auth   = 1500 // 国内线上max 1.03s，但是除了这个最大值外，都不超过100ms
	ThresholdTime_CN_Battle = 1000 // 国内线上max 338ms

	ThresholdTime_CN_ClientBi = 400   // 国内线上max 未查询到 本地测试最大值50ms
	ThresholdTime_CN_Cometx   = 10000 // 国内线上max 未查询到 本地测试发现普遍在5s左右

	ThresholdTime_CN_Game   = 8000 // 国内线上max 4.45s，大部分区间在2s左右。 2023.02.07 由于gamex检查增加了拉取cross消息和场景服检查，增加到8s
	ThresholdTime_CN_Gate   = 400  // 国内线上max 28.9ms
	ThresholdTime_CN_Notice = 400  // 国内线上max 12.9ms
	ThresholdTime_CN_pay    = 200  // 国内线上max 20.9ms 大部分区间在10ms以内
)

// 海外线上时间统计都是以周六一整天的最大值为准
// 阈值取最大时间*2，向上取整，单位ms
const (
	ThresholdTime_Auth   = 3000
	ThresholdTime_Battle = 1500

	ThresholdTime_ClientBi = 400
	ThresholdTime_Cometx   = 10000

	ThresholdTime_Game   = 10000
	ThresholdTime_Gate   = 400
	ThresholdTime_Notice = 400
	ThresholdTime_pay    = 200
)

func GetThresholdByServerTyp(serTyp string) int64 {
	switch serTyp {
	case etcd.Server_Auth:
		return ThresholdTime_CN_Auth
	case etcd.Server_Battle:
		return ThresholdTime_CN_Battle
	case etcd.Server_ClientBiLog:
		return ThresholdTime_CN_ClientBi
	case etcd.Server_ChatComet:
		return ThresholdTime_CN_Cometx
	case etcd.Server_Gamex:
		return ThresholdTime_CN_Game
	case etcd.Server_Gate:
		return ThresholdTime_CN_Gate
	case etcd.Server_Notice:
		return ThresholdTime_CN_Notice
	case etcd.Server_Pay:
		return ThresholdTime_CN_pay
	default:
		return 0
	}
}

func GetThresholdByServerTypeNotCN(serTyp string) int64 {
	switch serTyp {
	case etcd.Server_Auth:
		return ThresholdTime_Auth
	case etcd.Server_Battle:
		return ThresholdTime_Battle
	case etcd.Server_ClientBiLog:
		return ThresholdTime_ClientBi
	case etcd.Server_ChatComet:
		return ThresholdTime_Cometx
	case etcd.Server_Gamex:
		return ThresholdTime_Game
	case etcd.Server_Gate:
		return ThresholdTime_Gate
	case etcd.Server_Notice:
		return ThresholdTime_Notice
	case etcd.Server_Pay:
		return ThresholdTime_pay
	default:
		return 0
	}
}

func ServerCheckTime(url string, isCN bool, gid uint, timeKey string, serverTyp string, serId string, beforeTime int64, extra ...string) {
	threshold := ServerCheckDelay(isCN, serverTyp)
	delayTime := timeutil.Now().UnixNano() - beforeTime
	ServerCheckTimeWithDelay(url, gid, timeKey, serverTyp, serId, delayTime, threshold, extra...)
}

func ServerCheckDelay(isCn bool, serverType string) int64 {
	if isCn {
		return GetThresholdByServerTyp(serverType) * time.Millisecond.Nanoseconds()
	} else {
		return GetThresholdByServerTypeNotCN(serverType) * time.Millisecond.Nanoseconds()
	}
}

func ServerCheckTimeWithDelay(url string, gid uint, timeKey string, serverTyp string, serId string,
	delayTime int64, threshold int64, extra ...string) {
	if delayTime < threshold {
		return
	}

	content := fmt.Sprintf(" timeKey:%s\n 大区:%d\n 服务类型: %s\n 服务器id:%s\n 时间阈值(ms):%d\n 消耗时间(ms):%d\n 额外信息:%s\n",
		timeKey, gid, serverTyp, serId,
		threshold/time.Millisecond.Nanoseconds(), delayTime/time.Millisecond.Nanoseconds(),
		strings.Join(extra, "\n"))

	ServerCheck2FeiShu(url, content)
}

func ServerCheck2FeiShu(url, content string) {
	// 飞书发消息
	err := feishu.SendFeishu(url, content, false)
	if err != nil {
		tilogs.L().Errorf("FailedByYiDun send feishu failed, err %s", err.Error())
	}
}
