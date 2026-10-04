package metrics

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var (
	_gid      uint
	_serverId string
	// 数量和大小统计
	_mapCount          map[string]int32
	_mapClientMsgCount map[string]MsgInfo
	_addCountChan      chan msgInfo // 小于32KB的结构会分到栈上，相比指针更快
	_ClientReqCountLog func(uint, map[string]MsgInfo)

	// 时间统计
	_msgTimeChan chan msgTimeInfo
	_mapTime     map[string]TimeCount
	_reqTimeLog  func(uint, map[string]TimeCount)

	// warn统计
	_mapWarn  map[string]int32
	_warnChan chan string
	_warnLog  func(uint, map[string]int32)

	// yidun统计
	_mapYidun  map[string]int32
	_yidunChan chan string
	_yidunLog  func(uint, map[string]int32)
)

type MsgInfo struct {
	Count int32
	Size  int64
}
type TimeCount struct {
	Time  int64
	Count int32
}

func InitMetricUtil(gid uint, serverId string) {
	InitMetricUtilWithClientLog(gid, serverId, nil, nil, nil, nil)
}

func InitMetricUtilWithClientLog(gid uint, serverId string,
	clientReqCountLog func(uint, map[string]MsgInfo),
	reqTimeLog func(uint, map[string]TimeCount),
	warnLog func(uint, map[string]int32),
	yidunLog func(uint, map[string]int32),
) {
	_gid = gid
	_serverId = serverId
	_mapCount = make(map[string]int32, 256)
	_mapClientMsgCount = make(map[string]MsgInfo, 256)
	_addCountChan = make(chan msgInfo, 16384)
	_ClientReqCountLog = clientReqCountLog

	_msgTimeChan = make(chan msgTimeInfo, 16384)
	_mapTime = make(map[string]TimeCount, 256)
	_reqTimeLog = reqTimeLog

	_mapWarn = make(map[string]int32, 256)
	_warnChan = make(chan string, 1024)
	_warnLog = warnLog

	_mapYidun = make(map[string]int32, 256)
	_yidunChan = make(chan string, 1024)
	_yidunLog = yidunLog
	timerSimpleSend()
}

func ReqTimeStatistics(serv, name string, beforeTimeNano int64) {
	TimeStatistics(fmt.Sprintf("requests.%d.%s.%s", _gid, serv, name), name, beforeTimeNano)
}

// TimeStatisticsWithThreshold 耗费时间大于阈值时进行记录
func TimeStatisticsWithThreshold(name, key4Stat string, beforeTimeNano, threshold int64) {
	TimeStatisticsWithDelayAndThreshold(name, key4Stat, time.Now().UnixNano()-beforeTimeNano, threshold)
}

func TimeStatisticsWithDelayAndThreshold(name, key4Stat string, delayTime int64, threshold int64) {
	if delayTime < threshold {
		return
	}
	_ = SimpleSend(name+".time", strconv.Itoa(int(delayTime)))
	addTime(key4Stat, delayTime)
}

func TimeStatisticsWithDelayTime(name, key4Stat string, delayTime int64) {
	if delayTime >= 0 && delayTime < int64(10*time.Millisecond) {
		return
	}
	if delayTime < 0 && delayTime > -int64(10*time.Millisecond) {
		return
	}
	_ = SimpleSend(name+".time", strconv.Itoa(int(delayTime)))
	addTime(key4Stat, delayTime)
}

func TimeStatistics(name, key4Stat string, beforeTimeNano int64) {
	TimeStatisticsWithThreshold(name, key4Stat, beforeTimeNano, int64(10*time.Millisecond))
}

func ReqCountStatistics(serv, name string) {
	CountStatistics(fmt.Sprintf("requests.%d.%s.%s", _gid, serv, name))
}

func CountStatistics(name string) {
	addCount(name, "", 1, 0, MsgInfoTypeDefault)
	// metrics.SimpleSend(fmt.Sprintf("%s.count", metricsName), "1")
}

func CountNStatistics(name string, n int32) {
	addCount(name, "", n, 0, MsgInfoTypeDefault)
	// metrics.SimpleSend(fmt.Sprintf("%s.count", metricsName), "1")
}

func ClientCountStatistics(name string, clientMsgName string, size int64) {
	addCount(name, clientMsgName, 1, size, MsgInfoTypeDefault)
}

func ClientNCountStatistics(name string, clientMsgName string, n int32, size int64) {
	addCount(name, clientMsgName, n, size, MsgInfoTypeDefault)
}

func FieldChangeCountStatistics(name string, count int32) {
	addCount(name, "", count, 0, MsgInfoTypeField)
}

func SizeStatistics(name string, size int) {
	if size < 500 {
		return
	}
	_ = SimpleSend(name+".size", strconv.Itoa(size))
}

func ReqCountAndSizeStatistics(serv, name string, size int) {
	// name = fmt.Sprintf("requests.%d.%s.%s", _gid, serv, name)
	// CountStatistics(name)
	// SizeStatistics(name, size)
}

func WarnStatistics(str string) {
	if !IsMetricsValid() {
		return
	}
	select {
	case _warnChan <- str:
	default:
		tilogs.L().Errorf("allmetrics util WarnStatistics full, drop %s", str)
	}
}

func YidunStatistics(str string) {
	if !IsMetricsValid() {
		return
	}
	select {
	case _yidunChan <- str:
	default:
		tilogs.L().Errorf("allmetrics util YidunStatistics full, drop %s", str)
	}
}

func GetDBStatPrefix(serType, key, oper string) string {
	return fmt.Sprintf("%s.%d.%s.%s.%s", serType, _gid, _serverId, key, oper)
}

func timerSimpleSend() {
	go func() {
		timerChan := timeutil.Timer10MS.After(time.Second)
		minuteTimerChan := timeutil.TimerSec.After(time.Minute)
		for {
			select {
			case info := <-_addCountChan:
				switch info.msgInfoType {
				case MsgInfoTypeField:
					if info.n > _mapCount[info.metricsName] {
						_mapCount[info.metricsName] = info.n
					}
				default:
					_mapCount[info.metricsName] = _mapCount[info.metricsName] + info.n
				}
				if info.clientMsgName != "" {
					if _ClientReqCountLog != nil {
						_info, ok := _mapClientMsgCount[info.clientMsgName]
						if !ok {
							_info = MsgInfo{}
						}
						if info.n > 0 {
							_info.Count = _info.Count + info.n
						}
						if info.size > 0 {
							_info.Size = _info.Size + info.size
						}
						_mapClientMsgCount[info.clientMsgName] = _info
					}
				}
			case info := <-_msgTimeChan:
				if _reqTimeLog != nil {
					_tc := _mapTime[info.name]
					_tc.Time = _tc.Time + info.time
					_tc.Count = _tc.Count + 1
					_mapTime[info.name] = _tc
				}
			case str := <-_warnChan:
				if _warnLog != nil {
					_mapWarn[str] = _mapWarn[str] + 1
				}
			case str := <-_yidunChan:
				if _yidunLog != nil {
					_mapYidun[str] = _mapYidun[str] + 1
				}
			case str := <-_warnChan:
				if _warnLog != nil {
					_mapWarn[str] = _mapWarn[str] + 1
				}
			case <-timerChan:
				if len(_mapCount) > 0 {
					// map的遍历和SimpleSend其实是比较耗时的操作
					// 如果有可能，后续用ring优化这里，将读写分离到两个goroutine
					for k := range _mapCount {
						// 之前这里会make新map
						// 但性能测试发现iter并将value设置成0性能更好一些
						if _mapCount[k] != 0 {
							v := _mapCount[k]
							if v > 1 {
								_ = SimpleSend(k+".count", strconv.Itoa(int(v)))

								// 某个字段30s内赋值超过500次报个错
								metricsName := k
								if strings.HasPrefix(metricsName, "field.") {
									if _mapCount[metricsName] > 500 {
										tilogs.L().Errorf("field change metricsName %s too large, count %d", metricsName, _mapCount[metricsName])
									}
								}
							}
							_mapCount[k] = 0
						}
					}
				}
				timerChan = timeutil.Timer10MS.After(time.Second)
			case <-minuteTimerChan:
				if len(_mapClientMsgCount) > 0 && _ClientReqCountLog != nil {
					_ClientReqCountLog(_gid, _mapClientMsgCount)
					_mapClientMsgCount = make(map[string]MsgInfo, 512)
				}
				if len(_mapTime) > 0 && _reqTimeLog != nil {
					_reqTimeLog(_gid, _mapTime)
					_mapTime = make(map[string]TimeCount, 256)
				}
				if len(_mapWarn) > 0 && _warnLog != nil {
					_warnLog(_gid, _mapWarn)
					_mapWarn = make(map[string]int32, 256)
				}
				if len(_mapYidun) > 0 && _yidunLog != nil {
					_yidunLog(_gid, _mapYidun)
					_mapYidun = make(map[string]int32, 256)
				}
				if len(_mapWarn) > 0 && _warnLog != nil {
					_warnLog(_gid, _mapWarn)
					_mapWarn = make(map[string]int32, 256)
				}
				minuteTimerChan = timeutil.TimerSec.After(time.Minute)
			}
		}
	}()
}

func addCount(name string, clientMsgName string, n int32, size int64, t int32) {
	if !IsMetricsValid() {
		return
	}
	select {
	case _addCountChan <- msgInfo{
		metricsName:   name,
		clientMsgName: clientMsgName,
		n:             n,
		size:          size,
		msgInfoType:   t,
	}:
	default:
		tilogs.L().Errorf("allmetrics util addCount full, drop %s", name)
	}
}

func addTime(name string, time int64) {
	if !IsMetricsValid() {
		return
	}
	select {
	case _msgTimeChan <- msgTimeInfo{
		name: name,
		time: time,
	}:
	default:
		tilogs.L().Errorf("allmetrics util addTime full, drop %s", name)
	}
}

const (
	MsgInfoTypeDefault = iota
	MsgInfoTypeField
)

type msgInfo struct {
	metricsName   string
	clientMsgName string
	n             int32
	size          int64
	msgInfoType   int32
}

type msgTimeInfo struct {
	name string
	time int64
}

// ModuleCountTime 记录次数和时间
func ModuleCountTime(cmd string, start time.Time, gid, serverId string) {
	name := fmt.Sprintf("requests.%s.%s.%s", gid, serverId, cmd)
	TimeStatistics(name, cmd, start.UnixNano())
	CountStatistics(name)
}
