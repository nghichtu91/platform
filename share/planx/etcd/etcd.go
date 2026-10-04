package etcd

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"

	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	clientv3 "go.etcd.io/etcd/client/v3"
)

/*
	针对etcd3
	封装了一些比较通用的接口, 设计思路
	1、方便通用控制超时context
	2、隐藏一些etcd3的设计细节
	3、简化keepalive的使用
*/

const (
	time_out = 5 * time.Second
)

var (
	ErrInvalidRange = errors.New("invalid param")
	ErrNotExist     = errors.New("key not exist")
)

var etcdClient *clientv3.Client = nil

func InitEtcd(endPoins []string) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:            endPoins,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    10 * time.Second, // 根据 google.golang.org/grpc@v1.44.0/keepalive/keepalive.go 如果这个值小于10s，会被设置成10s
		DialKeepAliveTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		tilogs.L().Errorf("InitEtcd err %s", err.Error())
		return err
	}
	etcdClient = cli
	tilogs.L().Infof("InitEtcd success")
	return nil
}

func GetEtcd() *clientv3.Client {
	if etcdClient == nil {
		panic(fmt.Errorf("Etcd not init"))
	}
	return etcdClient
}

func CloseEtcd() {
	if etcdClient != nil {
		etcdClient.Close()
		tilogs.L().Infof("CloseEtcd success")
	}
}

func KeyExist(key string) bool {
	s, err := Get(key)
	if err != nil {
		return false
	}
	return s != ""
}

// ExistThenGet 判断key是否存在, 如果存在则返回对应的值, 否则返回 ErrNotExist
func ExistThenGet(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()
	resp, err := GetEtcd().Get(ctx, key)
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) <= 0 {
		return "", ErrNotExist
	}
	return string(resp.Kvs[0].Value), nil
}

// Get 获取单个key的value
func Get(key string) (string, error) {
	val, err := ExistThenGet(key)
	if errors.Is(err, ErrNotExist) {
		// 如果不存在, 这里不算是错误
		return "", nil
	}
	return val, err
}

// 获取单个key的value, 并返回版本号
// 版本号用于watch的时候用
func GetWithRev(key string) (string, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()
	resp, err := GetEtcd().Get(ctx, key)
	if err != nil {
		return "", 0, err
	}
	if len(resp.Kvs) <= 0 {
		return "", resp.Header.Revision, nil
	}
	return string(resp.Kvs[0].Value), resp.Header.Revision, nil
}

// 获得key目录下以及所有子目录下所有键值对的值
func GetSubRecursive(key string) (map[string]string, error) {
	kvs, _, err := GetSubRecursiveWithRev(key)
	return kvs, err
}

// 获得key目录下以及所有子目录下所有键值对的值，并返回版本号
// 版本号用于watch的时候用
func GetSubRecursiveWithRev(key string) (map[string]string, int64, error) {
	var res map[string]string
	rev, err := getSubRecursiveWithCallback(key, func(kv []*mvccpb.KeyValue) {
		res = make(map[string]string, len(kv))
		for _, v := range kv {
			res[string(v.Key)] = string(v.Value)
		}
	})
	return res, rev, err

}

func GetSubRecursiveWithKVRev(key string) (map[string]KvInfo, int64, error) {
	var allKvInfo map[string]KvInfo
	rev, err := getSubRecursiveWithCallback(key, func(kv []*mvccpb.KeyValue) {
		allKvInfo = make(map[string]KvInfo, len(kv))
		for _, v := range kv {
			info := GenKvInfo(v)
			allKvInfo[info.Key] = info
		}
	})
	return allKvInfo, rev, err
}

func getSubRecursiveWithCallback(key string, cb func(kv []*mvccpb.KeyValue)) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()
	resp, err := GetEtcd().Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return 0, err
	}
	if len(resp.Kvs) <= 0 {
		// 即使key不存在, 也返回版本号, 用于watch
		return resp.Header.Revision, nil
	}
	if cb != nil {
		cb(resp.Kvs)
	}
	return resp.Header.Revision, nil
}

// 获得一级目录key的集合
// 由于etcd3的存储是扁平结构, 所以获得一级目录下所有key没有直接的方法
// 目前先用过了路径的方法筛选出一级目录下所有key
func GetSubKeyOnly(key string) ([]string, error) {
	keys, _, err := GetSubKeyOnlyWithRev(key)
	return keys, err
}

// GetSubKeyOnlyWithRev 获得一级目录key的集合, 并返回版本号
func GetSubKeyOnlyWithRev(key string) ([]string, int64, error) {
	key = strings.TrimRight(key, "/")
	r, rev, err := GetSubRecursiveWithRev(key)
	if err != nil {
		return nil, rev, err
	}
	length := len(key)
	tmp := make(map[string]struct{}, 2)
	res := make([]string, 0, 4)
	for k := range r {
		if len(k) <= length {
			continue
		}
		s := strings.TrimPrefix(k, key+"/")
		ss := strings.Split(s, "/")
		if len(ss) > 0 {
			tmp[key+"/"+ss[0]] = struct{}{}
		}
	}
	for k := range tmp {
		res = append(res, k)
	}
	return res, rev, nil
}

func Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()
	_, err := GetEtcd().Delete(ctx, key)
	if err != nil {
		return err
	}
	return nil
}

// 删除key下以及所有子节点的值
func DeleteRecursive(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()
	_, err := GetEtcd().Delete(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	return nil
}

// PutWithPreRev 返回Put之前的Version信息
// Raft算法基本可以确保etcd集群内的Version的原子性
func PutWithPreRev(key, value string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()
	resp, err := GetEtcd().Put(ctx, key, value, clientv3.WithPrevKV())
	if err != nil {
		return 0, err
	}
	if resp.PrevKv == nil {
		return 0, nil
	}
	return resp.PrevKv.Version, nil
}

func Put(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()
	_, err := GetEtcd().Put(ctx, key, value)
	if err != nil {
		return err
	}
	return nil
}

// PutWithTTL 创建带TTL的键值，TTL为过期时间秒数，譬如3600
func PutWithTTL(key, value string, ttl int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()

	resp, err := GetEtcd().Grant(ctx, ttl)
	if err != nil {
		return err
	}

	_, err = GetEtcd().Put(ctx, key, value, clientv3.WithLease(resp.ID))
	return err
}

// Grant 获取LeaseID，用于其他批量操作
func Grant(ttl int64) (clientv3.LeaseID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()

	resp, err := GetEtcd().Grant(ctx, ttl)
	if err != nil {
		return 0, err
	}

	return resp.ID, nil
}

// PutWithLeaseID 基于某个lease id写etcd值
func PutWithLeaseID(key, value string, id clientv3.LeaseID) error {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()

	_, err := GetEtcd().Put(ctx, key, value, clientv3.WithLease(id))
	return err
}

// 监听key的变化
// 参数：
// revision: 大于等于0则有效，起始监听的版本号
// withPrev: 监听时，返回上一个版本的数据，true有效
// watchAllSub为true则监听所有子级目录；为false则不监听子级目录
func watch(key string, revision int64, withPrev, watchAllSub bool) (clientv3.Watcher, clientv3.WatchChan) {
	watcher := clientv3.NewWatcher(GetEtcd())
	var ch clientv3.WatchChan
	ops := make([]clientv3.OpOption, 0, 4)
	if watchAllSub {
		ops = append(ops, clientv3.WithPrefix())
	}
	if revision >= 0 {
		ops = append(ops, clientv3.WithRev(revision))
	}
	if withPrev {
		ops = append(ops, clientv3.WithPrevKV())
	}
	ch = watcher.Watch(context.Background(), key, ops...)
	return watcher, ch
}

/*
为了应对etcd的compact时，可能引起watch失败的情况，实现了watch的重试机制
如果etcdcompact的配置，预留一些历史版本，理论上watch就不应该再失败了
*/
func WatchWithRetry(key string, watchAllSub bool, quitChan <-chan struct{},
	wait *util.WaitGroupWrapper, action func(resp clientv3.WatchResponse)) {

	WatchWithRevNoPrevRetry(key, -1, watchAllSub, quitChan, wait, action)
}

// WatchWithRevNoPrevRetry 监听key的变化，注意: 这里的revision是拿到key时的版本号. 函数内部会进行+1处理
func WatchWithRevNoPrevRetry(key string, revision int64, watchAllSub bool, quitChan <-chan struct{},
	wait *util.WaitGroupWrapper, action func(resp clientv3.WatchResponse)) {
	if revision > 0 {
		revision++ // 从下一个版本开始监听
	}
	WatchWithRevRetry(key, revision, false, watchAllSub, quitChan, wait, action)
}

func WatchWithPrevRetry(key string, withPrev, watchAllSub bool, quitChan chan struct{},
	wait *util.WaitGroupWrapper, action func(resp clientv3.WatchResponse)) {

	WatchWithRevRetry(key, -1, withPrev, watchAllSub, quitChan, wait, action)
}

func WatchWithRevRetry(key string, revision int64, withPrev, watchAllSub bool, quitChan <-chan struct{},
	wait *util.WaitGroupWrapper, action func(resp clientv3.WatchResponse)) {

	var watcher clientv3.Watcher
	var respChan clientv3.WatchChan
	watcher, respChan = watch(key, revision, withPrev, watchAllSub)

	if watcher == nil || respChan == nil {
		tilogs.L().Errorf("watch failed %s", key)
		return
	}
	tilogs.L().Infof("start watch, %s, revision %v, withPrev %v, watchAllSub %v",
		key, revision, withPrev, watchAllSub)
	wait.Wrap(func() {
		for {
			select {
			case resp, ok := <-respChan:
				if !ok {
					tilogs.L().Errorf("watch closed, [%s]", key)
					return
				} else {
					if err := resp.Err(); err != nil {
						var nextWatchRevision int64 = -1
						if errors.Is(err, rpctypes.ErrCompacted) {
							nextWatchRevision = resp.CompactRevision
						}
						tilogs.L().Alarm("watch meet error [%s] %v nextWatch %v", key, err, nextWatchRevision)
						err := watcher.Close()
						if err != nil {
							tilogs.L().Errorf("watch close error [%s] %v", key, err)
						}
						/*
							因为watch失败后，并不知道的当前最新可用的revision是多少，所以重启的watch就从当前最新的开始监听
							可能会有丢失，但如果etcd的compact配置的预留一些历史版本，丢失的可能性就不大了
						*/
						WatchWithRevRetry(key, nextWatchRevision, withPrev, watchAllSub, quitChan, wait, action)
						return
					} else {
						func() {
							defer tilogs.PanicCatcher("watch %v action panic", key)
							action(resp)
						}()
					}
				}
			case <-quitChan:
				tilogs.L().Infof("watch quit %s", key)
				err := watcher.Close()
				if err != nil && err != context.Canceled && err != fmt.Errorf("Context canceled as expected") {
					tilogs.L().Errorf("watch close error on quit [%s] %v", key, err)
				}
				return
			}
		}
	})
}

// 以下是带有ttl的put和keepalive此put的一系列方法

func LeaseKeepAlive(key, value string, ttl int64, stop_ch chan struct{}) error {
	lease := clientv3.NewLease(GetEtcd())
	resp, err := lease.Grant(context.Background(), ttl)
	if err != nil {
		tilogs.L().Errorf("LeaseKeepAlive lease.Grant err %s, "+
			"k: %s v: %s ttl:%d", err.Error(), key, value, ttl)
		return err
	}

	_, err = GetEtcd().Put(context.Background(), key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		tilogs.L().Errorf("LeaseKeepAlive Put err %s, "+
			"k: %s v: %s ttl:%d", err.Error(), key, value, ttl)
		return err
	}

	ch, kaerr := lease.KeepAlive(context.Background(), resp.ID)
	if kaerr != nil {
		tilogs.L().Errorf("LeaseKeepAlive lease.KeepAlive err %s, "+
			"k: %s v: %s ttl:%d", err.Error(), key, value, ttl)
		return err
	}

	go func() {
		needRetry := false
	LOOP:
		for {
			select {
			case _, ok := <-ch:
				if !ok {
					lease.Close()
					tilogs.L().Infof("LeaseKeepAlive close")
					needRetry = true
					break LOOP
				}
			case <-stop_ch:
				lease.Close()
				return
			}
		}

		for needRetry {
			select {
			case <-stop_ch:
				lease.Close()
				return
			default:
			}
			tilogs.L().Infof("LeaseKeepAlive retry")
			err := LeaseKeepAlive(key, value, ttl, stop_ch)
			if err != nil {
				time.Sleep(time.Second * 5)
			} else {
				return
			}
		}
	}()

	return nil
}

// FormatKv 格式化KV
func FormatKv(kv *mvccpb.KeyValue) string {
	if kv == nil {
		return "<nil>"
	}
	return fmt.Sprintf("{Key: %s, Value: %s, CreateRevision: %d, ModRevision: %d, Version: %d, Lease: %d}",
		string(kv.Key), string(kv.Value), kv.CreateRevision, kv.ModRevision, kv.Version, kv.Lease)
}

type KvInfo struct {
	Key         string
	Value       string
	ModRevision int64 // key本次修改的Revision
	// TODO 其他字段, 尽量和
	// mvccpb.KeyValue 保持一致, 说不定后续需要直接替换呢?
}

func GenKvInfo(kv *mvccpb.KeyValue) (ret KvInfo) {
	ret.FromKeyValue(kv)
	return
}

func (k *KvInfo) FromKeyValue(kv *mvccpb.KeyValue) {
	if k == nil || kv == nil {
		return
	}
	k.Key = string(kv.Key)
	k.Value = string(kv.Value)
	k.ModRevision = kv.ModRevision
}

type KvInfoArray []KvInfo

func (s KvInfoArray) Len() int           { return len(s) }
func (s KvInfoArray) Less(i, j int) bool { return s[i].ModRevision < s[j].ModRevision }
func (s KvInfoArray) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s KvInfoArray) FirstRev() int64 {
	if !s.Empty() {
		return s[0].ModRevision
	}
	return 0
}

func (s KvInfoArray) LastRev() int64 {
	if !s.Empty() {
		return s[s.Len()-1].ModRevision
	}
	return 0
}

func (s KvInfoArray) Empty() bool {
	return s.Len() == 0
}

// RangeKVByRev 获取指定key rev属于[from,to]中间所有版本的数据
// 注意: 因为可能发生compact, 所以不保证所有版本都是存在的,
// 采用的方法是倒序迭代到指定版本之前
// to >= from
// 如果允许allowCompact, 那么当key的某些版本被Compact后不会返回错误
// 返回的数据不会包含key修改的时间, 如果有需要请在Value中自行处理
// 返回的数据包含from和to两个版本(如果没有被compact的话)
//   - 如果被key删除了, 那么就是空的列表
//   - 如果还存在, 那么首个版本的rev(KvInfoArray.FirstRev)应该是大于等于from的
//
// 返回指定区间内的变化, 以及key创建的版本
func RangeKVByRev(key string, from, to int64, allowCompact bool) (ret KvInfoArray, create int64, err error) {
	if from > to {
		err = ErrInvalidRange
		return
	}
	const (
		StartRevision = -1 // 起始的迭代游标
	)

	var resp *clientv3.GetResponse

	// 栈上分配
	var opCache [1]clientv3.OpOption

	// 获取指定版本的Key
	var get = func(v int64) (*clientv3.GetResponse, error) {
		var ctx, cancel = context.WithTimeout(context.Background(), time_out)
		defer cancel()
		var opt = opCache[:0]
		if v != StartRevision {
			opt = append(opt, clientv3.WithRev(v))
		}
		return GetEtcd().Get(ctx, key, opt...)
	}
	// 过滤键值对, 并返回下一个迭代的游标
	var filterKv = func(kvs []*mvccpb.KeyValue) int64 {
		// kvs应该是不为空的切片, 上层需要判断一下
		var minRev int64 = math.MaxInt64
		for _, kv := range kvs {
			if kv == nil {
				continue
			}
			if minRev > kv.ModRevision {
				minRev = kv.ModRevision
			}
			create = kv.CreateRevision
			if kv.ModRevision > to {
				continue
			}
			ret = append(ret, GenKvInfo(kv))
		}
		// 获取前一个版本
		return minRev - 1
	}
	var nextRev int64 = StartRevision
	for {
		resp, err = get(nextRev)
		if err != nil {
			// 如果发生了compact..? 旧版本的找不到就找不到了
			if err == rpctypes.ErrCompacted {
				if allowCompact {
					err = nil
					break
				}
			}
			return
		}
		// 如果数据为空了, 应该是没有更靠前的版本了
		if len(resp.Kvs) == 0 {
			break
		}
		// 过滤KV, 并获取更为靠前的版本
		nextRev = filterKv(resp.Kvs)
		// 如果下一个版本小于下区间, 就跳出循环
		if nextRev < from {
			break
		}
	}
	sort.Sort(ret)
	return
}
