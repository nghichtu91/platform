package etcd_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v3rpc "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
	"github.com/nghichtu91/platform/share/planx/util"
)

func init() {
	zaplog.TestLogger()

	if err := etcd.InitEtcd([]string{"localhost:2379"}); err != nil {
		panic(err)
	}
}

func logWatchEvent(t *testing.T, resp clientv3.WatchResponse) {
	t.Logf("header: %v", resp.Header.String())
	for _, event := range resp.Events {
		t.Logf("event kv: %s, prev_kv: %s", event.Kv.String(), event.PrevKv.String())
	}
}

func TestWatch(t *testing.T) {

	const Key = "test_key"

	client := etcd.GetEtcd()

	get := func(rev int64) {
		var getResp, err = client.Get(context.Background(), Key, clientv3.WithRev(rev))
		require.NoError(t, err, "is compacted %v", errors.Is(err, v3rpc.ErrCompacted))
		t.Logf("getResp: %v", getResp)
	}

	get(0)

	put := func() int64 {
		var putResp, err = client.Put(context.Background(), Key, "test_value")
		require.NoError(t, err)
		t.Logf("put revision: %d", putResp.Header.GetRevision())
		return putResp.Header.GetRevision()
	}

	t.Run("Compact", func(t *testing.T) {
		firstRev := put()
		t.Logf("firstRev: %d", firstRev)
		put()
		put()
		put()
		put()
		put()
		put()
		var compactNotify = make(chan struct{})
		var bindNotify = make(chan struct{})
		go func() {
			<-bindNotify

			compactRevision := firstRev + 5
			t.Logf("start compact revision: %d", compactRevision)
			compactResp, err := client.Compact(context.Background(), compactRevision)
			require.NoError(t, err)
			t.Logf("compactResp: %v", compactResp)
			close(compactNotify)
		}()

		var ctx = util.NewCancelCtx()

		var wg util.WaitGroupWrapper

		err := etcd.Watch(ctx, etcd.KeyOption("test_key").
			Revision(firstRev+1).Prev().
			WaitGroup(&wg).
			Bind(etcd.BindFunc(func(path string, resp *clientv3.GetResponse) error {
				t.Logf("bind header %v", resp.Header.String())
				for _, kv := range resp.Kvs {
					t.Logf("bind kvs %s", kv.String())
				}
				close(bindNotify)
				<-compactNotify
				return nil
			})).
			Action(func(response clientv3.WatchResponse) {
				logWatchEvent(t, response)
			}),
		)

		require.NoError(t, err)

		// pgrep TestWatch | xargs kill -SIGINT
		//var sigChan = make(chan os.Signal, 1)
		//signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		//t.Logf("signal: %v", <-sigChan)
		time.Sleep(time.Second)
		ctx.Stop()
		wg.Wait()
	})

	t.Run("Future", func(t *testing.T) {
		firstRev := put()
		t.Logf("firstRev: %d", firstRev)
		var wg util.WaitGroupWrapper
		var opt = etcd.KeyOption(Key)
		opt.Revision(firstRev).Prev().
			Bind(etcd.BindFunc(func(path string, resp *clientv3.GetResponse) error {
				t.Logf("bind header %v", resp.Header.String())
				for _, kv := range resp.Kvs {
					t.Logf("bind kvs %s", kv.String())
				}
				// 强制 watch 一个未来的 revision
				opt.Revision(firstRev + 4)
				return nil
			})).
			WaitGroup(&wg).
			Action(func(response clientv3.WatchResponse) {
				logWatchEvent(t, response)
			})

		var ctx = util.NewCancelCtx()
		err := etcd.Watch(ctx, opt)
		require.NoError(t, err)
		var putDone = make(chan struct{})
		wg.Wrap(func() {
			for i := 0; i < 6; i++ {
				t.Logf("put revision: %d", put())
				time.Sleep(time.Second)
			}
			close(putDone)
		})
		<-putDone
		ctx.Stop()
		wg.Wait()
	})
}

type TestData struct {
	Name    string `etcd3:"name"`
	Age     int    `etcd3:"age"`
	Address string `etcd3:"address"`
}

func TestExample(t *testing.T) {
	t.Run("Bind", func(t *testing.T) {
		const Dir = "test_val"

		require.NoError(t, etcd.SyncObj(Dir, &TestData{
			Name:    "test",
			Age:     18,
			Address: "test address",
		}))

		var ctx = util.NewCancelCtx()
		var wg util.WaitGroupWrapper

		runWatch := func(bind etcd.IBind, tag ...string) {
			tags := strings.Join(tag, " ")
			require.NoError(t, etcd.Watch(ctx, etcd.DirOption(Dir).
				WaitGroup(&wg).
				Bind(bind).
				Prev().
				Action(func(response clientv3.WatchResponse) {
					t.Logf("[%v] header: %v", tags, response.Header.String())
					for _, event := range response.Events {
						t.Logf("[%v] event kv: %s, prev_kv: %s",
							tags, event.Kv.String(), event.PrevKv.String())
					}
				}),
			))
		}

		runWatch(
			// bindValue Demo
			etcd.BindValue(false, func(val TestData) error {
				t.Logf("bindval: %+v", val)
				return nil
			}),
			"value",
		)

		runWatch(
			// bindValue Demo
			etcd.BindValue(false, func(val *TestData) error {
				t.Logf("bindval point: %+v", val)
				return nil
			}),
			"value pointer",
		)

		runWatch(
			// bindValue Demo
			etcd.BindValue(false, func(val map[string]string) error {
				t.Logf("val map: %+v", val)
				return nil
			}),
			"val map",
		)

		runWatch(
			// bindMap Demo
			etcd.BindMap(func(m map[string]string) error {
				t.Logf("map: %v", m)
				return nil
			}),
			"map",
		)

		runWatch(
			// bindFunc Demo
			etcd.BindFunc(func(path string, resp *clientv3.GetResponse) error {
				kvs := resp.Kvs
				for _, kv := range kvs {
					t.Logf("kvs: %s", kv.String())
				}
				return nil
			}),
			"func",
		)

		wg.Wrap(func() {
			err := etcd.SyncObj(Dir, &TestData{
				Name:    "test2",
				Age:     19,
				Address: "test address 2",
			})
			require.NoError(t, err)
			time.Sleep(time.Second * 2)
			ctx.Stop()
		})
		wg.Wait()
	})

	t.Run("Err1", func(t *testing.T) {
		const key = "test_key2"
		runWatch := func(wantErr error, allowEmpty bool) {
			var ctx = util.NewCancelCtx()
			err := etcd.Watch(ctx, etcd.KeyOption(key).
				Bind(etcd.BindValue(allowEmpty, func(str string) error {
					t.Logf("val: %v", str)
					return nil
				})).
				Action(func(response clientv3.WatchResponse) {
					logWatchEvent(t, response)
				}),
			)
			require.ErrorIs(t, err, wantErr)
			ctx.Stop()
		}

		runWatch(etcd.EmptyKvErr, false)
		runWatch(nil, true)

		time.Sleep(time.Second)
	})

	t.Run("BindErr", func(t *testing.T) {

		const key = "test_obj_bind_err"
		var data = &TestData{
			Name:    "123",
			Age:     199,
			Address: "5656",
		}
		require.NoError(t, etcd.SyncObj(key, data))

		runBind := func(bind etcd.IBind, dir string, wantErr error) {
			var ctx = util.NewCancelCtx()
			err := etcd.Watch(ctx, etcd.DirOption(dir).
				Bind(bind),
			)
			require.ErrorIs(t, err, wantErr)
			ctx.Stop()
		}

		runBind(etcd.BindValue(false, ""), key, etcd.ErrInvalidBindFunc)
		runBind(etcd.BindValue(false, func(string) {}), key, etcd.ErrInvalidBindFunc)
		runBind(etcd.BindValue(false, func(string) error { return nil }), key, nil)
		runBind(etcd.BindValue(false, func(v TestData) error {
			require.Equal(t, data, &v)
			return nil
		}), key, nil)
		var someError = errors.New("123")
		runBind(etcd.BindValue(false, func(v TestData) error {
			require.Equal(t, data, &v)
			return someError
		}), key, someError)
		runBind(etcd.BindValue(false, func(v *TestData) error {
			require.Equal(t, data, v)
			return nil
		}), key, nil)

		runBind(etcd.BindValue(false, func(v map[string]string) error {
			t.Logf("map: %v", v)
			return nil
		}), key, nil)

		runBind(etcd.BindValue(false, func(m map[string]string) error {
			t.Logf("map: %v", m)
			return nil
		}), key+"_not_found", etcd.EmptyKvErr)

		runBind(etcd.BindValue(false, func(c chan int) error {
			return nil
		}), key, nil)

	})

	t.Run("ErrNoAction", func(t *testing.T) {
		const key = "test_key3"

		var ctx = util.NewCancelCtx()
		var wg util.WaitGroupWrapper
		err := etcd.Watch(ctx, etcd.KeyOption(key).
			WaitGroup(&wg))
		require.NoError(t, err)
		wg.Wait()
	})

}

// 这个测试 case 一定得放在最后. 因为它会关闭 etcd 连接
func TestClose(t *testing.T) {
	const key = "test_key3"
	var ctx = util.NewCancelCtx()
	var wg util.WaitGroupWrapper
	require.NoError(t, etcd.Watch(ctx, etcd.KeyOption(key).
		WaitGroup(&wg).
		Action(func(response clientv3.WatchResponse) {
			logWatchEvent(t, response)
		}),
	))
	etcd.CloseEtcd()
	time.Sleep(time.Second)
	wg.Wait()
}
