package timeutil

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/planx/etcd"

	"github.com/nghichtu91/platform/share/planx"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

/*
时间cheat
1. 在cheatEnable为false模式下无效
2. 支持根据每个shard一个时间，也支持全大区统一时间，当设置了全大区时间，单服时间无效
3. 保存在etcd上的，所以重启服务器后也是生效的
*/
var (
	tGlobal_Offset int64 // 全局时间的offset
	gid            uint
	ser_id         string
	etcd_server    string
	cheat_enable   *string
	values         map[string]int64
	lock           sync.RWMutex
	quit           chan struct{}
	once           sync.Once

	// 时间发生改变后的回调
	// 会在锁里执行，不要放阻塞或耗时长的操作
	changeCallback []func()
)

func init() {
	_s := ""
	cheat_enable = &_s
	changeCallback = make([]func(), 0, 4)
}

const (
	TimeCheatAll = "all" // 全大区的标记
)

const (
	// 如果大区设置的时间小于0, 就是有特殊含义的删除.

	TimeCheatClear       = -1 // 标记删除1: 单服/大区重置
	TimeCheatOneKeyClear = -2 // 标记删除2: 一件重置全部的时间
)

var (
	TimeCheatClearStr      = strconv.Itoa(TimeCheatClear) // 标记删除, 而非真正的删除-字符串
	TimeCheatOnKeyClearStr = strconv.Itoa(TimeCheatOneKeyClear)
)

func Init(Gid uint, serId, etcdServer, cheatEnable string, wait *util.WaitGroupWrapper) error {
	gid = Gid
	ser_id = serId
	etcd_server = etcdServer
	StoreCheatEnable(cheatEnable)
	quit = make(chan struct{}, 1)
	return watchDebugTime(wait)
}

func Close() {
	once.Do(func() {
		if quit != nil {
			close(quit)
		}
		tilogs.L().Infof("time_debug close")
	})
}

// Now 获取当前时间
func Now() time.Time {
	if !planx.IsCheatEnable(LoadCheatEnable()) {
		return time.Now()
	}
	_offset := GetValidCheatTime(atomic.LoadInt64(&tGlobal_Offset))
	return time.Now().Add(time.Duration(_offset))
}

// NowByShardId 若设置了全区时间，则全部用全区时间。若没有设全区时间，则用参数shardId对应的时间
func NowByShardId(shardId uint) time.Time {
	if !planx.IsCheatEnable(LoadCheatEnable()) {
		return time.Now()
	}
	lock.RLock()
	defer lock.RUnlock()
	allT, ok := values[TimeCheatAll]
	if ok && !IsClearCheatTime(allT) {
		return Now()
	}
	shardT, shardOk := values[fmt.Sprint(shardId)]
	if shardOk && !IsClearCheatTime(shardT) {
		shardT = GetValidCheatTime(shardT)
		return time.Now().Add(time.Duration(shardT))
	}
	return time.Now()
}

// SetNow 通过etcd将指定时间同步到指定服
// 如果shardId为0，表示设置全大区时间
// timeValue需要符合格式 2006-01-02 15:04:05
// 可以将时间按DebugFormat2Set格式化
func SetNow(timeValue string, shardId uint) bool {
	if !planx.IsCheatEnable(LoadCheatEnable()) {
		return true
	}
	t, err := ParseInLocationWithDebugFormat(timeValue)
	if err != nil {
		tilogs.L().Errorf("time util.SetNow %s err %s", timeValue, err.Error())
		return false
	}
	iOffset := int64(t.Sub(time.Now()))
	// atomic.StoreInt64(&tGlobal_Offset, int64(strOffset))
	subKey := getShardKey(shardId)
	if !IsValidCheatTime(iOffset) {
		// 标记删除, 非真正删除
		iOffset = TimeCheatClear
	}
	strOffset := strconv.Itoa(int(iOffset))
	if putErr := etcd.Put(subKey, strOffset); putErr != nil {
		tilogs.L().Errorf("time_debug SetNow etcd.Put putErr %s", putErr.Error())
		return false
	}
	tilogs.L().Infof("TimeDebug SetNow time %v, shard %v", timeValue, shardId)
	return true
}

func ParseInLocationWithDebugFormat(timeValue string) (time.Time, error) {
	return time.ParseInLocation(DebugFormat2Set, timeValue, time.Local)
}

func ClearOffset(shardId uint) bool {
	if !planx.IsCheatEnable(LoadCheatEnable()) {
		return true
	}
	key := getShardKey(shardId)
	// 标记删除
	if err := etcd.Put(key, TimeCheatClearStr); err != nil {
		tilogs.L().Errorf("time_debug ClearOffset etcd.Delete err %s", err.Error())
		return false
	}
	tilogs.L().Infof("TimeDebug ClearOffset shardId %v", shardId)
	return true
}

// RegTimeChangeCallback 注册debug时间发生变化后的回调函数
func RegTimeChangeCallback(fn func()) {
	lock.Lock()

	changeCallback = append(changeCallback, fn)

	lock.Unlock()
}

func watchDebugTime(wait *util.WaitGroupWrapper) error {
	if !planx.IsCheatEnable(LoadCheatEnable()) {
		return nil
	}
	key := etcd.GetDebugTimeKey(gid, etcd_server)
	allSubKey, rev, err := etcd.GetSubRecursiveWithRev(key)
	if err != nil {
		tilogs.L().Errorf("watchDebugTime GetSubRecursiveWithRev err %s", err.Error())
		return err
	}
	lock.Lock()
	values = make(map[string]int64, 16)
	for _k, _v := range allSubKey {
		parseT, parseErr := strconv.ParseInt(_v, 10, 64)
		if parseErr != nil {
			tilogs.L().Errorf("watchDebugTime get first ParseInt k=%s parseT=%s parseErr %s", _k, _v, parseErr.Error())
			continue
		}
		if IsValidCheatTime(parseT) {
			_setKVInLock(_k, parseT)
		}
	}
	_updateInLock()
	tilogs.L().Infof("watchDebugTime get first %v", values)
	lock.Unlock()
	etcd.WatchWithRevRetry(key, rev, false, true, quit, wait, func(resp clientv3.WatchResponse) {
		for _, ev := range resp.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				tilogs.L().Debugf("watchDebugTime watch put %v, %v", string(ev.Kv.Key), string(ev.Kv.Value))
				lock.Lock()
				_k := string(ev.Kv.Key)
				_v := string(ev.Kv.Value)
				v, err := strconv.ParseInt(_v, 10, 64)
				if err != nil {
					tilogs.L().Errorf("watchDebugTime watch ParseInt k=%s v=%s err %s", _k, _v, err.Error())
					continue
				}
				if !IsValidCheatTime(v) {
					_delKVInLock(string(ev.Kv.Key))
				} else {
					_setKVInLock(_k, v)
				}
				tilogs.L().Debugf("watchDebugTime now %s", time.Now().Add(time.Duration(v)).Format(DebugFormat2Show))
				_updateInLock()

				if len(changeCallback) > 0 {
					for _, fn := range changeCallback {
						fn()
					}
				}

				lock.Unlock()
			case clientv3.EventTypeDelete:
				tilogs.L().Debugf("watchDebugTime watch del %v %v", string(ev.Kv.Key), string(ev.Kv.Value))
				lock.Lock()
				_delKVInLock(string(ev.Kv.Key))
				_updateInLock()
				lock.Unlock()
			}
		}
	})
	return nil
}

func getShardKey(shardId uint) string {
	path := etcd.GetDebugTimeKey(gid, etcd_server)
	if shardId <= 0 {
		// 全大区
		return fmt.Sprintf("%s/%s", path, TimeCheatAll)
	} else {
		// 指定服务器
		return fmt.Sprintf("%s/%d", path, shardId)
	}
}

// IsValidCheatTime 有效的时间必须是大于等于1s的
func IsValidCheatTime(t int64) bool {
	return int64(math.Abs(float64(t))) >= int64(time.Second)
}

func GetValidCheatTime(t int64) int64 {
	if !IsValidCheatTime(t) {
		return 0
	}
	return t
}

// IsClearCheatTime 是不是标记清除
func IsClearCheatTime(t int64) bool {
	return t < 0
}

func _setKVInLock(k string, v int64) {
	ks := strings.Split(k, "/")
	values[ks[len(ks)-1]] = v
}

func _delKVInLock(k string) {
	ks := strings.Split(k, "/")
	delete(values, ks[len(ks)-1])
}

// 若设置了全区时间，则全部用全区时间。若没有设全区时间，则找自己服务id对应的时间
func _updateInLock() {
	vAll, ok := values[TimeCheatAll]
	if ok {
		atomic.StoreInt64(&tGlobal_Offset, GetValidCheatTime(vAll))
	} else {
		v, shardOk := values[ser_id]
		if shardOk {
			atomic.StoreInt64(&tGlobal_Offset, GetValidCheatTime(v))
		} else {
			atomic.StoreInt64(&tGlobal_Offset, 0)
		}
	}
}

// SetNowForUT 专为单元测试用，隔离etcd的接口
func SetNowForUT(timeValue string) bool {
	if !planx.IsCheatEnable(LoadCheatEnable()) {
		return true
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", timeValue, time.Local)
	if err != nil {
		tilogs.L().Errorf("time_util.SetNow %s err %s", timeValue, err.Error())
		return false
	}
	iOffset := int64(t.Sub(time.Now()))
	atomic.StoreInt64(&tGlobal_Offset, iOffset)
	return true
}

func StoreCheatEnable(enable string) {
	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&cheat_enable)),
		(unsafe.Pointer)(&enable))
}

func LoadCheatEnable() string {
	return *(*string)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&cheat_enable))))
}

// Timestamp2DebugShowFormat 时间戳转换为可读格式
func Timestamp2DebugShowFormat(ts int64) string {
	return time.Unix(ts, 0).Format(DebugFormat2Show)
}

// Timestamp2WebFormat 时间戳转换为web格式，没有斜线和冒号，便于json解析或者拼接url
func Timestamp2WebFormat(ts int64) string {
	return time.Unix(ts, 0).Format(Format4Web)
}
