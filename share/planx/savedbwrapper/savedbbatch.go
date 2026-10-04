package savedbwrapper

import (
	"context"
	"fmt"
	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/util"
	"math"
	"os"
	"time"

	"github.com/nghichtu91/platform/share/planx/metrics"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/planx/redispool"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var batchSaveDB bool

// Batch NewSaveDB后调用，batch模式启动save db
func Batch(b bool) {
	batchSaveDB = b
}

func (save *SaveDB) IsSaveDBStart() bool {
	return save.saveDBStart
}

func (save *SaveDB) startSaveDBBatch() {
	save.wait.Wrap(func() {
		tilogs.L().Infof("%s StartSaveDB start", save.logFlag())
		_debugMsg := make([]string, 0, debugLogLen)
		timer1s := timeutil.TimerSec.After(time.Second)

		saveBatchChangedAll := func() {
			// 全部落库，重置缓存和上次下标
			if len(save.batchChanged) > 0 {
				save.save(save.batchChanged, _debugMsg)
				save.batchChanged = save.batchChanged[:0]
				save.lastBatchIndex = 0
			}
		}

		for {
			select {
			case needSave, ok := <-save.saveCh:
				if !ok {
					// 退出时，将缓存全部数据落库
					saveBatchChangedAll()
					tilogs.L().Infof("%s SaveDB.StartSaveDB close", save.logFlag())
					return
				}
				if len(save.batchChanged) == 0 {
					save.batchChanged = make([]*DBDataInfoWithKey, 0, defaultChangeLen)
					save.lastBatchInitTS = timeutil.Now().Unix()
				}
				var haveCmdGetAll bool
				for k, subs := range needSave {
					for _, sub := range subs {
						for _, e := range sub {
							save.batchChanged = append(save.batchChanged, &DBDataInfoWithKey{
								Key:        k,
								DBDataInfo: e,
							})
							if e.DBCmd == DBCmdGetAll {
								haveCmdGetAll = true
							}
						}
					}
				}
				// 有DBCmdGetAll全部执行
				if haveCmdGetAll {
					saveBatchChangedAll()
				}

			case <-timer1s:
				l := len(save.batchChanged)
				// 有落库数据
				if l > 0 {
					// 不应该出现这种情况
					if l <= save.lastBatchIndex {
						errStr := fmt.Sprintf("%s SaveDB.StartSaveDB batchChanged len %d, lastBatchIndex %d", save.logFlag(), l, save.lastBatchIndex)
						if !planx.IsRunProd(save.devMode) {
							tilogs.L().Error(errStr)
							os.Exit(1)
						} else {
							tilogs.L().Alarm(errStr)
						}
					}
					// 剩余数和最大批数取小
					currBatch := int(math.Min(float64(l-save.lastBatchIndex), float64(defaultBatchCount)))
					// 本次要落库的数据
					batchChange := save.batchChanged[save.lastBatchIndex : save.lastBatchIndex+currBatch]
					// 落库
					save.save(batchChange, _debugMsg)
					// 更新lastBatchIndex
					save.lastBatchIndex = save.lastBatchIndex + currBatch
					// 如果已经全部落库，清空队列重置下标
					if save.lastBatchIndex >= l {
						save.batchChanged = save.batchChanged[:0]
						save.lastBatchIndex = 0
						// 队列过长，1分钟后缩容
						now := timeutil.Now().Unix()
						if l > defaultChangeLen && now-save.lastBatchInitTS > timeutil.MinSec {
							save.batchChanged = make([]*DBDataInfoWithKey, 0, defaultChangeLen)
							save.lastBatchInitTS = now
						}
					}
				}
				timer1s = timeutil.TimerSec.After(time.Second)
			}
		}
	})
}

// saveDBBatch 需要和SetMaybeChange同一goroutine中执行
// 定时执行，如：30s
func (save *SaveDB) saveDBBatch(quit bool) (hasChg, success bool) {
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

func (save *SaveDB) SyncGetAll(dbFieldName string) (interface{}, error) {
	if len(save.changed) > 0 {
		panic(fmt.Errorf("%s SyncGetAll changed not empty", save.logFlag()))
	}
	retCh := make(chan interface{}, 1)
	needGet := make(map[string][][]*DBDataInfo, 1)
	needGet[dbFieldName] = [][]*DBDataInfo{[]*DBDataInfo{&DBDataInfo{DBCmd: DBCmdGetAll, ChReply: retCh}}}
	save.saveCh <- needGet

	ctx, cancel := context.WithTimeout(context.Background(), redispool.DefaultReadTimeout*2)
	defer cancel()
	select {
	case ret := <-retCh:
		return ret, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("%s SyncGetAll retCh channel full, ret timeout", save.logFlag())
	}
}

func (save *SaveDB) save(needSave []*DBDataInfoWithKey, _debugMsg []string) {
	/*
		执行存db操作，若执行期间出错，则踢玩家下线，并一直重试直到成功
		若数据库有比较长的时间挂掉了，则玩家可能还能再登录上来，就会再次被踢
	*/
	_needSave := needSave
	for {
		reErr := timeutil.RetryIfErr(func() (er error) {
			defer func() {
				if err := recover(); err != nil {
					errMsg := fmt.Sprintf("%s, err:%v", "saveDB panic", err)
					tilogs.L().With("accountid", save.dbKeyName).Errorf(errMsg)
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
				// 标记key是否有需要更新过期时间的操作
				// 执行DEL和EXPIRE时，不重复执行EXPIRE
				var updated bool
				for _, e := range _needSave {
					i++

					k := e.Key
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
				tilogs.L().Debugf("%s save success, cmd len %v debug msg %v", save.logFlag(), len(_needSave), _debugMsg)
				return nil
			}, timeutil.NewBackOffWithMaxElapsedTime(5*time.Minute))
			return err
		}, timeutil.NewBackOffWithMaxElapsedTime(5*time.Minute))
		if reErr == nil {
			break
		}
	}
}
