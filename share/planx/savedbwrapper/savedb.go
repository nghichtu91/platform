package savedbwrapper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/planx/metrics"

	"github.com/golang/protobuf/proto"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/planx/redispool"
)

// NewSaveDB 收集可能的变化，并定时发给存db的goroutine进行异步存db操作
// 使用环境为：单goroutine内，不是线程安全的
func NewSaveDB(serTyp, moduleName, dbKeyName string, dbPool redispool.IPool, devMode string) *SaveDB {
	var db = &SaveDB{
		serTyp:        serTyp,
		moduleName:    moduleName,
		dbKeyName:     dbKeyName,
		dbPool:        dbPool,
		changed:       make(map[string][][]*DBMsgInfo, 16),
		saveCh:        make(chan map[string][][]*DBDataInfo, 4), // 没给太大的缓存空间，如果数据库出问题，都会缓存在changed里
		changeChASync: make(chan *changeInfo, 1024),
		devMode:       devMode,
		flag:          fmt.Sprintf("<SaveDB><%s>", dbKeyName),
	}
	db.StartSaveDB = sync.OnceFunc(db.startInner)
	return db
}

type SaveDB struct {
	serTyp     string
	moduleName string // 仅生成metrics统计用，不影响逻辑
	dbKeyName  string // 仅日志使用，不影响逻辑
	devMode    string
	dbPool     redispool.IPool

	changed map[string][][]*DBMsgInfo // 因为有删表操作，所以用了个[][]双切片结构，删表在第一个切片里记录，其他操作在最后一个切片里记录

	saveCh          chan map[string][][]*DBDataInfo
	batchChanged    []*DBDataInfoWithKey // save db goroutine db key 缓存的操作，用于按时间批量操作
	lastBatchInitTS int64
	lastBatchIndex  int
	wait            util.WaitGroupWrapper

	// async
	changeChASync chan *changeInfo

	// 用于日志
	flag string

	// 存储类型是否为自动过期
	autoExpire bool

	// 绑定了哪些key
	// 会在quit时，从用于校验的全局变量里删除对应的key
	bindKeys []string

	// StartSaveDB是否已经启动
	saveDBStart bool

	// sync.OnceFunc 包一下
	StartSaveDB func()
}

func (save *SaveDB) startInner() {
	save.saveDBStart = true
	if batchSaveDB {
		save.startSaveDBBatch()
		return
	}
	save.startSaveDB()
}

func (save *SaveDB) startSaveDB() {
	save.wait.Wrap(func() {
		tilogs.L().Infof("%s StartSaveDB start", save.logFlag())

		_debugMsg := make([]string, 0, debugLogLen)

		for {
			select {
			case needSave, ok := <-save.saveCh:
				if !ok {
					tilogs.L().Infof("%s SaveDB.StartSaveDB close", save.logFlag())
					return
				}
				/*
					执行存db操作，若执行期间出错，则踢玩家下线，并一直重试直到成功
					若数据库有比较长的时间挂掉了，则玩家可能还能再登录上来，就会再次被踢
				*/
				_needSave := needSave
				for {
					re_err := timeutil.RetryIfErr(func() (er error) {
						defer func() {
							if err := recover(); err != nil {
								errmsg := fmt.Sprintf("%s, err:%v", "saveDB panic", err)
								tilogs.L().With("accountid", save.dbKeyName).Errorf(errmsg)
								er = fmt.Errorf("%v", err)
							}
							if er != nil {
								// 如果存db失败了，这里只是会一直卡着尝试存储，直到存成功
							}
						}()

						err := timeutil.RetryIfErr(func() error {
							conn := save.dbPool.GetDBConn()
							if conn.Err() != nil {
								tilogs.L().Errorf("%s RetryIfErr GetDBConn err %s", save.logFlag(), conn.Err())
								return conn.Err()
							}
							defer conn.Close()

							i := -1
							hgetallInfos := make([]*hgetallInfo, 0, 2)
							_debugMsg = _debugMsg[:0]
							for k, subs := range _needSave {
								// 标记key是否有需要更新过期时间的操作
								// 执行DEL和EXPIRE时，不重复执行EXPIRE
								var updated bool
								for _, sub := range subs {
									for _, e := range sub {
										i++
										switch e.DBCmd {
										case DBCmdGetAll:
											hgetallInfos = append(hgetallInfos, &hgetallInfo{
												i:       i,
												chReply: e.ChReply,
											})
											conn.Send("HGETALL", k)
											_debugMsg = append(_debugMsg, fmt.Sprintf("HGETALL %s,", k))
										case DBCmdAddUpdate:
											conn.Send("HSET", k, e.SubKey, e.Data)
											_debugMsg = append(_debugMsg, fmt.Sprintf("HSET %s %s,", k, e.SubKey))
											updated = true
										case DBCmdDel:
											conn.Send("HDEL", k, e.SubKey)
											_debugMsg = append(_debugMsg, fmt.Sprintf("HDEL %s %s,", k, e.SubKey))
											updated = true
										case DBCmdDelTable:
											conn.Send("DEL", k)
											_debugMsg = append(_debugMsg, fmt.Sprintf("DEL %s,", k))
										case DBCmdExpire:
											conn.Send("EXPIRE", k, e.Params[0])
											_debugMsg = append(_debugMsg, fmt.Sprintf("EXPIRE %s %s,", k, e.Params[0]))
										}
									}
								}

								// JWS2-39177 增加自动设置过期时间功能
								if save.autoExpire && updated {
									conn.Send("EXPIRE", k, autoExpireDefaultSeconds)
									_debugMsg = append(_debugMsg, fmt.Sprintf("autoExpire EXPIRE %s %d,", k, autoExpireDefaultSeconds))
								}
							}

							// 注意：pika数据库不支持事务，只支持pipeline，所以DoCmdBuffer参数只能用false
							r, err := conn.DoCmdBuffer(metrics.GetDBStatPrefix(save.serTyp, save.moduleName,
								"SaveDB"), false)
							if err != nil {
								tilogs.L().Errorf("%s DoCmdBuffer err %s", save.logFlag(), err.Error())
								return err
							}
							if len(hgetallInfos) > 0 {
								reply, ok := r.([]interface{})
								if !ok {
									tilogs.L().Errorf("%s DoCmdBuffer reply is not []interface{}, r %v",
										save.logFlag(), r)
									return nil
								}
								for _, info := range hgetallInfos {
									if len(reply) <= info.i {
										tilogs.L().Errorf("%s DoCmdBuffer reply len %d, index %d", save.logFlag(), len(reply), info.i)
										continue
									}
									info.chReply <- reply[info.i]
								}
							}
							tilogs.L().Debugf("%s save success %v", save.logFlag(), _debugMsg)
							return nil
						}, timeutil.NewBackOffWithMaxElapsedTime(5*time.Minute))
						return err
					}, timeutil.NewBackOffWithMaxElapsedTime(5*time.Minute))
					if re_err == nil {
						break
					}
					time.Sleep(time.Millisecond * 10)
				}
			}
		}
	})
}

func (save *SaveDB) SetChanged(dbCmd DBCmd, dbFieldName, subDbFieldName string, data proto.Message, params ...string) {
	save.SendSetChanged(dbCmd, dbFieldName, subDbFieldName, "", nil, data, params...)
}

func (save *SaveDB) SendSetChanged(dbCmd DBCmd, dbFieldName, subDbFieldName string, acid string, hooker IMarshalHook, data proto.Message, params ...string) {
	v, ok := save.changed[dbFieldName]
	if !ok {
		v = make([][]*DBMsgInfo, 0, 2)
		v = append(v, make([]*DBMsgInfo, 0, defaultChangeLen))
		save.changed[dbFieldName] = v
	}
	// 若是删表操作，则清除此表的其他已有修改
	if dbCmd == DBCmdDelTable {
		// 此时
		// v[0] = {DBCmdDelTable}
		// v[1] = {DBCmdAddUpdate, DBCmdDel, DBCmdExpire}
		v = make([][]*DBMsgInfo, 0, 2)
		v = append(v, []*DBMsgInfo{
			{
				DBCmd: dbCmd,
			},
		})
		v = append(v, make([]*DBMsgInfo, 0, defaultChangeLen))
		save.changed[dbFieldName] = v
		return
	}
	// 若是表过期
	lastCmds := v[len(v)-1]
	if dbCmd == DBCmdExpire {
		if len(params) < 1 {
			tilogs.L().Errorf("%s %s Expire lack param", save.logFlag(), dbFieldName)
			return
		}
		lastCmds = append(lastCmds, &DBMsgInfo{
			DBCmd:  dbCmd,
			Params: params,
		})
		v[len(v)-1] = lastCmds
		return
	}
	// 更新或添加子键值
	// 2022.10.24 暂时注释掉合并逻辑
	// 国服官职结算时，人数较多，短时间存入大量cmd
	// 每次存入时，下面的逻辑会遍历所有acid的cmd，反而导致CPU占用特别高
	// for _, od := range lastCmds {
	// 	if od.SubKey == subDbFieldName {
	// 		od.DBCmd = dbCmd
	// 		od.Data = data
	// 		od.Params = params
	// 		return
	// 	}
	// }
	lastCmds = append(lastCmds, &DBMsgInfo{
		DBCmd:  dbCmd,
		SubKey: subDbFieldName,
		Data:   data,
		Params: params,
		acid:   acid,
		hooker: hooker,
	})
	v[len(v)-1] = lastCmds
}

/*
需要和SetMaybeChange同一goroutine中执行
定时执行，如：30s
*/
func (save *SaveDB) SaveDB(quit bool) (hasChg, success bool) {
	if batchSaveDB {
		return save.saveDBBatch(quit)
	}
	return save.saveDB(quit)
}

func (save *SaveDB) saveDB(quit bool) (hasChg, success bool) {
	defer func() {
		if quit {
			tilogs.L().Debugf("%s rev quit", save.logFlag())
			close(save.saveCh)
			save.wait.Wait()
			save.RemoveKeysOnQuit()
		}
	}()
	if len(save.changed) <= 0 {
		return false, true
	}
	//
	_needSave := make(map[string][][]*DBMsgInfo, len(save.changed))
	for k, subs := range save.changed {
		nsSubs, ok := _needSave[k]
		if !ok {
			nsSubs = make([][]*DBMsgInfo, 0, len(subs))
			_needSave[k] = nsSubs
		}
		for _, sub := range subs {
			var expire *DBMsgInfo
			nsSubMap := make(map[string]*DBMsgInfo, len(sub))
			nsSubs = append(nsSubs, make([]*DBMsgInfo, 0, len(sub)))
			for _, e := range sub {
				if e.DBCmd == DBCmdExpire {
					if expire == nil {
						expire = &DBMsgInfo{
							DBCmd:  DBCmdExpire,
							Params: e.Params,
							acid:   e.acid,
							hooker: e.hooker,
						}
					} else {
						expire.Params = e.Params
					}
					continue
				}
				if e.SubKey != "" {
					// 合并相同子键值
					s, exists := nsSubMap[e.SubKey]
					if exists {
						s.DBCmd = e.DBCmd
						s.Data = e.Data
						s.Params = e.Params
						continue
					}
				}

				data := &DBMsgInfo{
					DBCmd:  e.DBCmd,
					SubKey: e.SubKey,
					Data:   e.Data,
					Params: e.Params,
					acid:   e.acid,
					hooker: e.hooker,
				}

				if e.SubKey != "" {
					nsSubMap[e.SubKey] = data
				}
				nsSubs[len(nsSubs)-1] = append(nsSubs[len(nsSubs)-1], data)
			}
			if expire != nil {
				nsSubs[len(nsSubs)-1] = append(nsSubs[len(nsSubs)-1], expire)
			}
		}
		_needSave[k] = nsSubs
	}
	if len(_needSave) <= 0 {
		// 清空maybeChange
		save.changed = make(map[string][][]*DBMsgInfo, 16)
		return false, true
	}
	// 对去过重的db改动进行marshal
	needSave := make(map[string][][]*DBDataInfo, len(_needSave))
	for k, subs := range _needSave {
		nsSubs, ok := needSave[k]
		if !ok {
			nsSubs = make([][]*DBDataInfo, 0, len(subs))
			needSave[k] = nsSubs
		}
		for _, sub := range subs {
			nsSubs = append(nsSubs, make([]*DBDataInfo, 0, len(sub)))
			for _, e := range sub {
				var dataByte []byte
				if e.Data != nil {
					if e.hooker != nil {
						e.hooker.BeforeMarshal(e.acid)
					}
					pm, err := Marshal(e.Data, save.devMode)
					if e.hooker != nil {
						e.hooker.AfterMarshal(e.acid)
					}
					if err != nil {
						tilogs.L().Errorf("%s %s, proto.Marshal err:%s, msg: %s", save.logFlag(), k, err.Error(), e.Data.String())
						continue
					}
					dataByte = pm
				}
				data := &DBDataInfo{
					DBCmd:  e.DBCmd,
					SubKey: e.SubKey,
					Data:   dataByte,
					Params: e.Params,
				}
				nsSubs[len(nsSubs)-1] = append(nsSubs[len(nsSubs)-1], data)
			}
			needSave[k] = nsSubs
		}
	}

	// 传给save的goroutine
	tilogs.L().Debugf("%s will save key count %d", save.logFlag(), len(_needSave))
	if quit {
		save.saveCh <- needSave // 无限等待
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), util.ASyncCmdTimeOut)
		defer cancel()
		select {
		// 异步存db
		case save.saveCh <- needSave:
			// 清空maybeChange。channel成功清空，若失败则不清，下次接着存
			save.changed = make(map[string][][]*DBMsgInfo, 16)
		case <-ctx.Done():
			tilogs.L().Errorf("%s channel full, put timeout", save.logFlag())
			return true, false
		}
	}
	return true, true
}

func (save *SaveDB) logFlag() string {
	return save.flag
}

func Marshal(msg proto.Message, devMode string) ([]byte, error) {
	// if planx.IsRunLocal(devMode) {
	//	return json.Marshal(msg)
	// } else {
	return proto.Marshal(msg)
	// }
}

func Unmarshal(buf []byte, pb proto.Message, devMode string) error {
	// if planx.IsRunLocal(devMode) {
	//	return json.Unmarshal(buf, pb)
	// } else {
	return proto.Unmarshal(buf, pb)
	// }
}

// SetAutoExpire 开启存储键值自动过期
// 除了DEL和EXPIRE，其他命令更新的键值，都会自动设置过期时间
// Note: 这里没有对autoExpire做原子操作，请务必在StartSaveDB之前调用
func (save *SaveDB) SetAutoExpire() {
	save.autoExpire = true
}
