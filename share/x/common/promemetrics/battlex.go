package promemetrics

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs/pmetricslog"

	"github.com/nghichtu91/platform/share/planx/pmetrics"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	BattlexCCUName            = "ccu"
	BattlexConnNumName        = "conn_num"
	BattlexRoomNumName        = "room_num"
	BattlexCommonRevName      = "rev"
	BattlexCommonSendName     = "send"
	BattlexWillSendName       = "will_send"
	BattlexCommonRevSizeName  = "rev_size"
	BattlexCommonSendSizeName = "send_size"
	BattlexReadCostName       = "read_cost"
	BattlexWriteCostName      = "write_cost"
)

const (
	BattlexRobotMaxRSName = "maxrs"
	BattlexRobotMinRSName = "minrs"
	BattlexRobotMidRSName = "midrs"
	BattlexRobotAvgRSName = "avgrs"
)

var (
	battlexCCUGauge           prometheus.Gauge // 记录战斗服CCU的Gauge，次值将被用于负载均衡。
	battlexConnNumGauge       prometheus.Gauge
	battlexRoomNumGauge       prometheus.Gauge // 记录战斗服当前当前房间数目的Gauge。
	battlexWillSendCounter    prometheus.Counter
	battlexReadCostHistogram  prometheus.Histogram // 用以记录战斗服玩家读取网络消息时间的直方图指标。
	battlexWriteCostHistogram prometheus.Histogram // 用以记录战斗服玩家写入网络消息时间的直方图指标。
)

var (
	battlexRobotMaxRSGauge prometheus.Gauge // 用以记录战斗服所有机器人最近一次Ping的最大值。
	battlexRobotMinRSGauge prometheus.Gauge // 用以记录战斗服所有机器人最近一次Ping的最小值。
	battlexRobotMidRSGauge prometheus.Gauge // 用以记录战斗服所有机器人最近一次Ping的中位值。
	battlexRobotAvgRSGauge prometheus.Gauge // 用以记录战斗服所有机器人最近一次Ping的平均值。
)

var (
	battlexCommonRevCounter      prometheus.Counter // 用以记录战斗服累计接受消息的数目。
	battlexCommonSendCounter     prometheus.Counter // 用以记录战斗服累计发送消息的数目。
	battlexCommonRevSizeCounter  prometheus.Counter // 用以记录战斗服累计接受消息的大小（字节）。
	battlexCommonSendSizeCounter prometheus.Counter // 用以记录战斗服累计发送消息的大小（字节）。
)

// 用于向PushGateWay推送时的JobName。
var BattlxJobName string

// GetBattlexJobName 获取Battlx服在PushGateWay和Prome展示的Job昵称
//
// Param-gid:服务器的大区ID。 Param-serverid：服务器区服ID。
//
// Return-string:向PushGateWay推送时的JobName。
func GetBattlexJobName(gid uint, serverId string) string {
	BattlxJobName = fmt.Sprintf("%s_%d_%s", BattlePrefix, gid, serverId)
	return BattlxJobName
}

// GetBattlexDBStatPrefix 获取BattlexDB相关的前缀
func GetBattlexDBStatPrefix(key, oper string, battlexGid uint) string {
	return fmt.Sprintf("%s_%d_%s_%s", BattlePrefix, battlexGid, key, oper)
}

// InitBattlexMetrics 初始化Battlex服务器关于Metrics的Collector与Statistics。
//
// Param-cfgPath:配置文件名。 Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-error:初始化时的报错信息。
func InitBattlexMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetBattlexJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitBattlexMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	if !pmetricslog.LoadPMetricsLog(cfgPath, gid, sid, BattlxJobName) {
		return fmt.Errorf("<InitBattlexMetrics> LoadPMetricsLog err")
	}

	ccuGauge, ccuErr := pmetrics.NewGauge(pmetrics.GetCollectorName(BattlxJobName, BattlexCCUName))
	if ccuErr != nil {
		return ccuErr
	}
	battlexCCUGauge = ccuGauge

	connNumGauge, connNumErr := pmetrics.NewGauge(pmetrics.GetCollectorName(BattlxJobName, BattlexConnNumName))
	if connNumErr != nil {
		return connNumErr
	}
	battlexConnNumGauge = connNumGauge

	roomNumGauge, roomNumErr := pmetrics.NewGauge(pmetrics.GetCollectorName(BattlxJobName, BattlexRoomNumName))
	if roomNumErr != nil {
		return roomNumErr
	}
	battlexRoomNumGauge = roomNumGauge

	willSendCounter, willSendErr := pmetrics.NewCounter(pmetrics.GetCollectorName(BattlxJobName, BattlexWillSendName))
	if willSendErr != nil {
		return willSendErr
	}
	battlexWillSendCounter = willSendCounter

	readCostHistogram, readCostErr := pmetrics.NewHistogram(pmetrics.GetCollectorName(BattlxJobName, BattlexReadCostName))
	if readCostErr != nil {
		return readCostErr
	}
	battlexReadCostHistogram = readCostHistogram

	writeCostHistogram, writeCostErr := pmetrics.NewHistogram(pmetrics.GetCollectorName(BattlxJobName, BattlexWriteCostName))
	if writeCostErr != nil {
		return writeCostErr
	}
	battlexWriteCostHistogram = writeCostHistogram

	if err := initBattlexCommonMetrics(); err != nil {
		return err
	}

	return nil
}

// InitBattlexRobotMetrics 初始化Battlx服务器关于Robot的Metrics的Collector。
//
// Param-cfgPath:配置文件名。 Param-gid:服务器的大区ID。 Param-sid：服务器区服ID。
//
// Return-error:初始化时的报错信息。
func InitBattlexRobotMetrics(cfgPath string, gid uint, sid string, proj string) error {
	// 加载prometheus的定期metrics发送服务。
	if err := pmetrics.StartPromethues(cfgPath, proj, GetBattlexJobName(gid, sid)); err != nil {
		return fmt.Errorf("<InitBattlexRobotMetrics> prometheus start err: %v", err)
	}
	// 加载PMetrics相关的Log组件。
	pmetricslog.LoadPMetricsLog(cfgPath, gid, sid, BattlxJobName)

	maxRsGuage, maxRsErr := pmetrics.NewGauge(pmetrics.GetCollectorName(BattlxJobName, BattlexRobotMaxRSName))
	if maxRsErr != nil {
		return maxRsErr
	}
	battlexRobotMaxRSGauge = maxRsGuage

	minRsGuage, minRsErr := pmetrics.NewGauge(pmetrics.GetCollectorName(BattlxJobName, BattlexRobotMinRSName))
	if minRsErr != nil {
		return minRsErr
	}
	battlexRobotMinRSGauge = minRsGuage

	midRsGuage, midRsErr := pmetrics.NewGauge(pmetrics.GetCollectorName(BattlxJobName, BattlexRobotMidRSName))
	if midRsErr != nil {
		return midRsErr
	}
	battlexRobotMidRSGauge = midRsGuage

	avgRsGuage, avgRsErr := pmetrics.NewGauge(pmetrics.GetCollectorName(BattlxJobName, BattlexRobotAvgRSName))
	if avgRsErr != nil {
		return avgRsErr
	}
	battlexRobotAvgRSGauge = avgRsGuage

	if err := initBattlexCommonMetrics(); err != nil {
		return err
	}

	return nil
}

// initBattlexCommonMetrics 初始化Battlex服务器通用Metrics的Collector。
func initBattlexCommonMetrics() error {
	revCounter, revErr := pmetrics.NewCounter(pmetrics.GetCollectorName(BattlxJobName, BattlexCommonRevName))
	if revErr != nil {
		return revErr
	}
	battlexCommonRevCounter = revCounter

	sendCounter, sendErr := pmetrics.NewCounter(pmetrics.GetCollectorName(BattlxJobName, BattlexCommonSendName))
	if sendErr != nil {
		return sendErr
	}
	battlexCommonSendCounter = sendCounter

	revSizeCounter, revSizeErr := pmetrics.NewCounter(pmetrics.GetCollectorName(BattlxJobName, BattlexCommonRevSizeName))
	if revSizeErr != nil {
		return revSizeErr
	}
	battlexCommonRevSizeCounter = revSizeCounter

	sendSizeCounter, sendSizeErr := pmetrics.NewCounter(pmetrics.GetCollectorName(BattlxJobName, BattlexCommonSendSizeName))
	if sendSizeErr != nil {
		return sendSizeErr
	}
	battlexCommonSendSizeCounter = sendSizeCounter

	return nil
}

// IncBattlexCCUGauge 记录战斗服CCU的Gauge+1。
func IncBattlexCCUGauge() {
	if battlexCCUGauge == nil {
		return
	}
	battlexCCUGauge.Inc()
}

// DecBattlexCCUGauge 记录战斗服CCU的Gauge-1。
func DecBattlexCCUGauge() {
	if battlexCCUGauge == nil {
		return
	}
	battlexCCUGauge.Dec()
}

// IncBattlexConnNumGauge 记录战斗服连接的数量的Gauge+1。
func IncBattlexConnNumGauge() {
	if battlexConnNumGauge == nil {
		return
	}
	battlexConnNumGauge.Inc()
}

// DecBattlexConnNumGauge 记录战斗服连接的数量的Gauge-1。
func DecBattlexConnNumGauge() {
	if battlexConnNumGauge == nil {
		return
	}
	battlexConnNumGauge.Dec()
}

// GetBattlexCCUGauge 获取记录战斗服CCU的Gauge，此值会被写入ETCD，用于战斗服的负载均衡。
//
// Return-int64:当前战斗服的玩家连接数量。
func GetBattlexCCUGauge() int64 {
	if battlexCCUGauge == nil {
		return pmetrics.CannotGetValue
	}
	return pmetrics.GetMetricValueByType(battlexCCUGauge, pmetrics.Gauge)
}

// IncBattlexRoomNumGuage 记录战斗服当前战斗房间数量的Gauge+1。
func IncBattlexRoomNumGuage() {
	if battlexRoomNumGauge == nil {
		return
	}
	battlexRoomNumGauge.Inc()
}

// DecBattlexRoomNumGuage 记录战斗服当前战斗房间数量的Gauge-1。
func DecBattlexRoomNumGuage() {
	if battlexRoomNumGauge == nil {
		return
	}
	battlexRoomNumGauge.Dec()
}

// OnBattlexRev 当战斗服收到协议时的相关处理（处理数+1，累计接受大小+Size）。
//
// Param-msgSize:接受的消息的字节大小。
func OnBattlexRev(msgSize int64) {
	if battlexCommonRevCounter != nil {
		battlexCommonRevCounter.Inc()
	}

	if battlexCommonRevSizeCounter != nil {
		battlexCommonRevSizeCounter.Add(float64(msgSize))
	}
}

// OnBattlexSend 当战斗服发送协议时的相关处理（处理数+1，累计发送大小+Size）。
//
// Param-msgSize:发送的消息的字节大小。
func OnBattlexSend(msgSize int64) {
	if battlexCommonSendCounter != nil {
		battlexCommonSendCounter.Inc()
	}

	if battlexCommonSendSizeCounter != nil {
		battlexCommonSendSizeCounter.Add(float64(msgSize))
	}
}

// UpdateReadCostHistogram 将最近一次玩家读取网络消息的时间花费加入直方图指标。
//
// Param-value:花费时间nanoseconds（纳秒）。
func UpdateReadCostHistogram(value int64) {
	if battlexReadCostHistogram == nil {
		return
	}
	battlexReadCostHistogram.Observe(float64(value))
}

// UpdateWriteCostHistogram 将最近一次玩家消息写入网络的时间花费加入直方图指标。
//
// Param-value:花费时间nanoseconds（纳秒）。
func UpdateWriteCostHistogram(value int64) {
	if battlexWriteCostHistogram == nil {
		return
	}
	battlexWriteCostHistogram.Observe(float64(value))
}

// UpdateBattleRobotRS 更新战斗服机器人相关的指标。
// des:获取的参数均是通过每个机器人的Ping时间来计算的，正常情况下为1s左右。
//
// Param-max:所有机器人Ping的最大值。 Param-min:所有机器人Ping的最小值。 Param-mid:所有机器人Ping的中位数。
// Param-avg:所有机器人Ping的平均值。
func UpdateBattleRobotRS(max, min, mid, avg int64) {
	if battlexRobotMaxRSGauge != nil && max > 0 {
		battlexRobotMaxRSGauge.Set(float64(max))
	}

	if battlexRobotMinRSGauge != nil && min > 0 {
		battlexRobotMinRSGauge.Set(float64(min))
	}

	if battlexRobotMidRSGauge != nil && mid > 0 {
		battlexRobotMidRSGauge.Set(float64(mid))
	}

	if battlexRobotAvgRSGauge != nil && avg > 0 {
		battlexRobotAvgRSGauge.Set(float64(avg))
	}
}
