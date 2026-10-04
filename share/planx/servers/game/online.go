package game

import (
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"sync/atomic"

	"github.com/siddontang/go/timingwheel"
	"github.com/nghichtu91/platform/share/planx/util"
)

var (
	dbMux      sync.RWMutex
	dbReady    map[string]*atomic.Value
	dbTimer    *timingwheel.TimingWheel
	stopAssist *StopAssist
)

func init() {
	dbReady = make(map[string]*atomic.Value)
	dbTimer = timingwheel.NewTimingWheel(100*time.Millisecond, 10*60)
	stopAssist = newStopAssist()
}

func getDBStatusInUse(accountId string) *atomic.Value {
	dbMux.RLock()
	defer dbMux.RUnlock()
	tilogs.L().Debugf("DBStatus InUse:%s", accountId)
	return dbReady[accountId]
}

func markDBStatusRelease(accountId string) {
	dbMux.Lock()
	defer dbMux.Unlock()
	tilogs.L().Debugf("DBStatus Release:%s", accountId)
	delete(dbReady, accountId)
}

func isDBStatusReadyAndMark(accountId string) (*atomic.Value, bool) {
	dbMux.RLock()
	v, ok := dbReady[accountId]
	if ok {
		dbMux.RUnlock()
		tilogs.L().Debugf("DBStatus Check failed:%s", accountId)
		return v, false
	}
	dbMux.RUnlock()

	dbMux.Lock()
	defer dbMux.Unlock()
	v, ok = dbReady[accountId]
	if ok {
		tilogs.L().Debugf("DBStatus Check failed:%s", accountId)
		return v, false
	}
	tilogs.L().Debugf("DBStatus Check ok and InUse:%s", accountId)
	newV := &atomic.Value{}
	dbReady[accountId] = newV
	return newV, true
}

func IsAllAccountOffLine() int {
	dbMux.RLock()
	defer dbMux.RUnlock()
	return len(dbReady)
}

func GetAllAccountOffLine() []string {
	dbMux.RLock()
	defer dbMux.RUnlock()
	res := make([]string, 0, len(dbReady))
	for k := range dbReady {
		res = append(res, k)
	}
	return res
}

/*
	stopAssist此类用来辅助停服
	gamex停服一般分两步，首先所有玩家下线存db，然后所有modules在stop
	但当有些玩家因为bug停不掉的时候，就不能在往后进行了，这时若强杀gamex，则modules里的东西又会丢失
	所以stopAssist的作用是这样：
	首先停服的时候开始计时5s, 若这期间所有玩家都正常退出了，则正常执行停modules
	若5s后还有玩家没有退出，则1s检查一次当前未退出的玩家数量是否变化，若没有退出的玩家数量5s都没有变化，则放弃这些玩家，停modules关服
*/
func newStopAssist() *StopAssist {
	return &StopAssist{
		beginStopCh:         make(chan struct{}, 1),
		playerStopSuccessCh: make(chan struct{}, 1),
	}
}

func GetStopAssist() *StopAssist {
	return stopAssist
}

type StopAssist struct {
	beginStopCh         chan struct{}
	playerStopSuccessCh chan struct{}
	waitGroup           util.WaitGroupWrapper
}

func (h *StopAssist) Stop() {
	h.beginStopCh <- struct{}{}
}

func (h *StopAssist) PlayerStopSuccess() {
	h.playerStopSuccessCh <- struct{}{}
	tilogs.L().Infof("PlayerStopSuccess...")
}

func (h *StopAssist) GetWait() *util.WaitGroupWrapper {
	return &h.waitGroup
}

func (h *StopAssist) Run(after_player_stop func()) {
	h.waitGroup.Wrap(func() {
		<-h.beginStopCh
		tilogs.L().Infof("StopAssist begin stop")

		is_player_stop_success := false
		wait_player_stop := time.After(time.Second * 2)
		select {
		case <-wait_player_stop:
		case <-h.playerStopSuccessCh:
			is_player_stop_success = true
		}

		if is_player_stop_success {
			tilogs.L().Infof("after player stop...")
			after_player_stop()
			return
		}

		// timeout
		n := 0
		no_stop_player := IsAllAccountOffLine()
		tilogs.L().Infof("%d players stop failed...", no_stop_player)
		n = 0
		t := time.After(time.Second)
		for {
			select {
			case <-t:
				if no_stop_player != IsAllAccountOffLine() {
					no_stop_player = IsAllAccountOffLine()
					tilogs.L().Infof("%d players stop failed...", no_stop_player)
					n++
					if n < 5 { // 若5s后还有玩家没有退出，则放弃这些玩家
						t = time.After(time.Second)
						continue
					}
				}
				tilogs.L().Infof("%d players stop failed...", no_stop_player)
				after_player_stop()
				tilogs.L().Infof("Exit with %d players no stop, %v",
					no_stop_player, GetAllAccountOffLine())
				return
			}
		}
	})
}

func (h *StopAssist) Wait() {
	h.waitGroup.Wait()
}
