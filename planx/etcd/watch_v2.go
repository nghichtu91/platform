package etcd

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"go.etcd.io/etcd/api/v3/mvccpb"
	v3rpc "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

// WatchV2版本
// 	1. 通过 context 控制 watch 的生命周期
//	2. 默认情况下, 复用底层一条 stream. 可以通过 ctx 创建新的 stream
//	3. 支持处理 compact 的情况

var (
	ErrInvalidBindFunc = errors.New("invalid bind function")
)

type WatchAction func(response clientv3.WatchResponse)

type WatchOption struct {
	// 浅拷贝后存在影响的字段

	logger    tilogs.TiLogger
	waitGroup *util.WaitGroupWrapper // 用于等待 Watch 协程结束
	action    WatchAction            // Watch 事件的回调函数
	bindOpt   IBind                  // 用于绑定数据. 相当于提前获取一次数据

	// 浅拷贝后互不影响的字段

	key      string // 需要处理的 key
	revision int64  // 开始监听的 revision
	prefix   bool   // 是否要处理以这个 key 为前缀的所有 key
	prev     bool   // 是否需要获取到这个 key 变化之前的值
}

// KeyOption 用于构建 key 的 WatchOption
func KeyOption(key string) *WatchOption {
	return &WatchOption{
		key: key,
	}
}

// DirOption 用于构建 dir 的 WatchOption
func DirOption(dir string) *WatchOption {
	return &WatchOption{
		key:    dir,
		prefix: true,
	}
}

// Clone 复制 WatchOption. 注意: 这是浅拷贝!
func (w *WatchOption) Clone() *WatchOption {
	var ret = *w
	return &ret
}

// Prev 设置 Watch 需要获取历史数据
func (w *WatchOption) Prev() *WatchOption {
	w.prev = true
	return w
}

// Revision 设置 Watch 的 revision
func (w *WatchOption) Revision(revision int64) *WatchOption {
	w.revision = revision
	return w
}

// Action 设置 Watch 的回调函数
func (w *WatchOption) Action(action WatchAction) *WatchOption {
	w.action = action
	return w
}

// WaitGroup 设置 Watch 的 WaitGroup
func (w *WatchOption) WaitGroup(wg *util.WaitGroupWrapper) *WatchOption {
	w.waitGroup = wg
	return w
}

// Bind 设置 Watch 的绑定函数
func (w *WatchOption) Bind(bind IBind) *WatchOption {
	w.bindOpt = bind
	return w
}

// toOption 转换为 OpOption
func (w *WatchOption) toOption() []clientv3.OpOption {
	var opts = make([]clientv3.OpOption, 0, 3)
	if w.prefix {
		opts = append(opts, clientv3.WithPrefix())
	}
	if w.prev {
		opts = append(opts, clientv3.WithPrevKV())
	}
	if w.revision > 0 {
		opts = append(opts, clientv3.WithRev(w.revision))
	}
	return opts
}

// validate 将 WatchOption 的参数进行校验并设置合法的默认值
func (w *WatchOption) validate() error {

	if w.waitGroup == nil {
		w.waitGroup = &util.WaitGroupWrapper{}
	}

	if w.revision < 1 {
		w.revision = 0
	}

	if w.logger == nil {
		w.logger = tilogs.L().With("key", w.key)
	}

	if w.bindOpt != nil {
		if err := w.bindOpt.check(); err != nil {
			return err
		}
	}
	return nil
}

// doAction 执行 Watch 的回调函数
func (w *WatchOption) doAction(response clientv3.WatchResponse) {
	defer tilogs.PanicCatcher("watch action panic %v", w.key)
	// 更新 用于后续 watch 的 revision
	w.Revision(response.Header.GetRevision() + 1)
	w.action(response)
}

// bind
func (w *WatchOption) bind(ctx context.Context) error {
	if w.bindOpt == nil {
		return nil
	}

	get, err := GetEtcd().Get(ctx, w.key, w.toOption()...)
	if err != nil {
		w.logger.Errorf("watch get failed, err %s", err.Error())
		return err
	}

	if w.bindOpt != nil {
		if err := w.bindOpt.bind(w.key, get); err != nil {
			w.logger.Errorf("watch bind failed, err %s", err.Error())
			return err
		}
	}

	// 只有首次需要执行 bind, 执行完之后就清空, 如果后续基于这个 option 再次 watch, 就不会再执行 bind
	w.bindOpt = nil

	// 确定 watch 的 revision
	if w.revision > 0 {
		// 如果指定了 revision, 就从指定的 revision的下一个版本开始 watch
		w.Revision(w.revision + 1)
	} else {
		// 如果没有指定 revision, 则应该使用当前 get 的 revision
		w.Revision(get.Header.GetRevision() + 1)
	}
	return nil
}

// Deprecated: genKvEvents 用于生成 kv 变化事件
// prev: 之前的 kv
// cur: 当前的 kv
func (w *WatchOption) genKvEvents(prev []*mvccpb.KeyValue, cur *clientv3.GetResponse) []*clientv3.Event {
	// 对比 prev 和 current, 生成 kv 变化事件
	var prevMap = make(map[string]*mvccpb.KeyValue, len(prev))
	for _, kv := range prev {
		prevMap[string(kv.Key)] = kv
	}

	current := cur.Kvs

	getPrev := func(kv *mvccpb.KeyValue) *mvccpb.KeyValue {
		if !w.prev {
			return nil
		}
		return kv
	}

	// 生成 kv 变化事件
	var events = make([]*clientv3.Event, 0, len(current))
	for _, kv := range current {
		if prevKv, ok := prevMap[string(kv.Key)]; ok {
			delete(prevMap, string(kv.Key)) // 删除已经处理的 kv
			if prevKv.ModRevision == kv.ModRevision {
				// 没有变化
				continue
			}
			// 有变化
			events = append(events, &clientv3.Event{
				Type:   mvccpb.PUT,
				Kv:     kv,
				PrevKv: getPrev(prevKv),
			})
		} else {
			// 新增
			events = append(events, &clientv3.Event{
				Type: mvccpb.PUT,
				Kv:   kv,
			})
		}
	}

	// prevMap中剩下的就是被删除的kv
	for _, kv := range prevMap {
		events = append(events, &clientv3.Event{
			Type: mvccpb.DELETE,
			Kv: &mvccpb.KeyValue{
				Key:         kv.Key,
				ModRevision: cur.Header.GetRevision(), // 修订的版本号采用当前 get 的版本号
			},
			PrevKv: getPrev(kv),
		})
	}

	return events
}

// Watch normal watch
func Watch(ctx context.Context, w *WatchOption) error {
	err := w.validate()
	if err != nil {
		w.logger.Errorf("watch option validate failed, err %s", err.Error())
		return err
	}

	err = w.bind(ctx)
	if err != nil {
		// 可能需要重试, 尤其是在指定了 revision 并且该 revision 已经被 compact 的情况下
		return err
	}

	if w.action == nil {
		// ? why not set action
		w.logger.Warnf("watch action is nil")
		return nil
	}

	var cancelCtx = util.NewCancelCtxWith(ctx)
	watchChan := GetEtcd().Watch(cancelCtx, w.key, w.toOption()...)
	if watchChan == nil {
		w.logger.Errorf("watchChan is nil")
		return v3rpc.ErrCorrupt
	}
	w.logger.Infof("start key watch at revision %v", w.revision)

	w.waitGroup.Wrap(func() {
		defer cancelCtx.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Infof("watch exit")
				return
			case resp, ok := <-watchChan:
				if !ok {
					w.logger.Infof("watchChan closed")
					return
				}
				if err := resp.Err(); err != nil {
					opt := w.Clone()
					last := opt.revision
					if errors.Is(err, v3rpc.ErrCompacted) {
						// 如果发生了 compact, 则需要根据 compact 的 revision 重新 watch
						opt.Revision(resp.CompactRevision)
					} else if errors.Is(err, v3rpc.ErrFutureRev) {
						// 可能输入的 revision 太大, 那么此时的 revision 是不可用的
						// etcd v3.5 目前发现, 允许 watch 一个不存在的 revision, 并且发生这个错误(?)
						opt.Revision(0)
					} else {
						// 暂时基于最后一次的 revision+1 重新 watch
					}
					w.logger.Alarm(
						"watch error %v, last revision %v retry start revision %d",
						err, last,
						opt.revision)
					// 再次 watch 不会触发 bind, 因为 bind 只在第一次 watch 时触发
					_ = Watch(ctx, opt)
					return
				} else {
					w.doAction(resp)
				}
			}
		}
	})

	return nil
}

// IBind 用于初始化 的值绑定
type IBind interface {
	check() error
	bind(string, *clientv3.GetResponse) error
}

type bindBase struct{}

// check 用于检查是否是有效的 bind
func (b bindBase) check() error { return nil }

// BindValue 用于绑定单个值. value必须要是指针类型
// loaded 的类型应该是形如  func (Type) error 的函数
// 如果 loaded 函数返回了错误, 则后续的 watch 将不会执行
// TODO use generic
func BindValue(allowEmpty bool, loaded interface{}) IBind {
	return &bindVal{
		allowEmpty: allowEmpty,
		loaded:     loaded,
	}
}

type bindVal struct {
	bindBase
	allowEmpty bool // 是否忽略空值
	loaded     interface{}
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// check
func (b *bindVal) check() error {
	fnValue := reflect.ValueOf(b.loaded)
	if fnValue.Kind() != reflect.Func {
		return fmt.Errorf("%w: not a function", ErrInvalidBindFunc)
	}

	fnType := fnValue.Type()
	// 需要是形如 func (Type) error 的函数
	if fnType.NumIn() != 1 || fnType.NumOut() != 1 || fnType.Out(0) != errorType {
		return fmt.Errorf("%w: function must be of type func(Type) error", ErrInvalidBindFunc)
	}
	return nil
}

// bind
func (b *bindVal) bind(path string, resp *clientv3.GetResponse) error {
	kvs := resp.Kvs

	if err := b.check(); err != nil {
		return err
	}

	fnValue := reflect.ValueOf(b.loaded)
	fnType := fnValue.Type()

	paramType := fnType.In(0)
	// 构建参数
	var paramValue reflect.Value
	if paramType.Kind() == reflect.Ptr {
		paramValue = reflect.New(paramType.Elem())
	} else {
		paramValue = reflect.New(paramType)
	}

	if len(kvs) == 0 {
		if !b.allowEmpty {
			// 如果没有值, 并且不忽略空值, 则返回错误
			return EmptyKvErr
		}
	} else {
		bindErr := BindKvs(path, paramValue.Interface(), kvs)
		if bindErr != nil {
			return bindErr
		}
	}

	if paramType.Kind() != reflect.Ptr {
		paramValue = paramValue.Elem()
	}

	result := fnValue.Call([]reflect.Value{paramValue})
	if result[0].IsNil() {
		return nil
	} else {
		return result[0].Interface().(error)
	}
}

func BindMap(fn bindMap) IBind { return fn }

type bindMap func(map[string]string) error

// check
func (b bindMap) check() error { return nil }

// bind
func (b bindMap) bind(_ string, resp *clientv3.GetResponse) error {
	kvs := resp.Kvs
	mp := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		mp[string(kv.Key)] = string(kv.Value)
	}
	return b(mp)
}

// BindFunc 用于处理 自定义的 bind
func BindFunc(fn bindFunc) IBind { return fn }

type bindFunc func(path string, resp *clientv3.GetResponse) error

// check
func (b bindFunc) check() error { return nil }

// bind
func (b bindFunc) bind(path string, resp *clientv3.GetResponse) error {
	return b(path, resp)
}
