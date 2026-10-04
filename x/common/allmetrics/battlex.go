package allmetrics

import (
	"fmt"
	"sync"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	battlexGid               uint
	_battle_ccu              gm.Counter
	_conn_num                gm.Counter
	_room_ccu                gm.Counter
	_rev                     gm.Counter
	_send                    gm.Counter
	_willSend                gm.Counter
	_rev_size                gm.Counter
	_send_size               gm.Counter
	_maxRS                   gm.Gauge
	_minRS                   gm.Gauge
	_midRS                   gm.Gauge
	_avgRS                   gm.Gauge
	_kcpRs                   gm.Histogram
	_readCost                gm.Histogram
	_writeCost               gm.Histogram
	_battleTypeRoomCCU       map[string]gm.Counter
	_battleTypeRoomCCULock   sync.RWMutex
	_battleTypePlayerCCU     map[string]gm.Counter
	_battleTypePlayerCCULock sync.RWMutex
	_battleLocalRecordNum    gm.Counter
	_battleLocalRecordBytes  gm.Counter
)

func PrefixBattleMetrics(gid uint, serverId string) string {
	battlexGid = gid
	dataprefix := fmt.Sprintf("%s.%d.%s", BattlePrefix, gid, serverId)
	return dataprefix
}

func InitBattleMetrics() {
	_battle_ccu = metrics.NewCustomCounter("ccu")
	_conn_num = metrics.NewCustomCounter("connnum")
	_room_ccu = metrics.NewCustomCounter("roomnum")
	_rev = metrics.NewCustomCounter("rev")
	_send = metrics.NewCustomCounter("send")
	_willSend = metrics.NewCustomCounter("willsend")
	_rev_size = metrics.NewCustomCounter("revsize")
	_send_size = metrics.NewCustomCounter("sendsize")
	_readCost = metrics.NewCustomHistogram("readcost")
	_writeCost = metrics.NewCustomHistogram("writecost")
	_kcpRs = metrics.NewCustomHistogram("kcprs")
	_battleTypeRoomCCU = make(map[string]gm.Counter, 4)
	_battleTypePlayerCCU = make(map[string]gm.Counter, 4)
	_battleLocalRecordNum = metrics.NewCustomCounter("battlerecord")
	_battleLocalRecordBytes = metrics.NewCustomCounter("battlerecordbyte")
}

func PrefixBattleRobotMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", BattlePrefix, gid, serverId)
}

func InitBattleRobotMetrics() {
	_maxRS = metrics.NewCustomGauge("maxrs")
	_minRS = metrics.NewCustomGauge("minrs")
	_midRS = metrics.NewCustomGauge("midrs")
	_avgRS = metrics.NewCustomGauge("avgrs")
	_rev = metrics.NewCustomCounter("rev")
	_send = metrics.NewCustomCounter("send")
	_rev_size = metrics.NewCustomCounter("revsize")
	_send_size = metrics.NewCustomCounter("sendsize")
	_kcpRs = metrics.NewCustomHistogram("kcprs")
}

func GetBattlexDBStatPrefix(key, oper string) string {
	return fmt.Sprintf("%s.%d.%s.%s", BattlePrefix, battlexGid, key, oper)
}

func AddBattleCCU(battleType string) {
	if battleType == etcd.Server_ServerCheck {
		return
	}
	if _battle_ccu != nil {
		_battle_ccu.Inc(1)
	}
	k := battleTypePlayerCCUPrefix(battleType)
	_battleTypePlayerCCULock.RLock()
	v, ok := _battleTypePlayerCCU[k]
	_battleTypePlayerCCULock.RUnlock()
	if !ok {
		_battleTypePlayerCCULock.Lock()
		v, ok = _battleTypePlayerCCU[k]
		if !ok {
			v = metrics.NewCounter(k)
			_battleTypePlayerCCU[k] = v
		}
		_battleTypePlayerCCULock.Unlock()
	}
	v.Inc(1)
}

func ReduceBattleCCU(battleType string, n int64) {
	if battleType == etcd.Server_ServerCheck {
		return
	}
	if _battle_ccu != nil {
		_battle_ccu.Dec(n)
	}
	k := battleTypePlayerCCUPrefix(battleType)
	_battleTypePlayerCCULock.RLock()
	v, ok := _battleTypePlayerCCU[k]
	_battleTypePlayerCCULock.RUnlock()
	if !ok {
		return
	}
	v.Dec(n)
}

func AddBattleConnNum() {
	if _conn_num != nil {
		_conn_num.Inc(1)
	}
}

func ReduceBattleConnNum() {
	if _conn_num != nil {
		_conn_num.Dec(1)
	}
}

func GetBattleCCUCount() int64 {
	if _battle_ccu != nil {
		return _battle_ccu.Count()
	}
	return 0
}

func AddBattleRoomNum(battleType string) {
	if _room_ccu != nil {
		_room_ccu.Inc(1)
	}
	k := battleTypeRoomNumPrefix(battleType)
	_battleTypeRoomCCULock.RLock()
	v, ok := _battleTypeRoomCCU[k]
	_battleTypeRoomCCULock.RUnlock()
	if !ok {
		_battleTypeRoomCCULock.Lock()
		v, ok = _battleTypeRoomCCU[k]
		if !ok {
			v = metrics.NewCounter(k)
			_battleTypeRoomCCU[k] = v
		}
		_battleTypeRoomCCULock.Unlock()
	}
	v.Inc(1)
}

func AddBattleLocalRecordNum() {
	_battleLocalRecordNum.Inc(1)
}

func AddBattleLocalRecordBytes(size int64) {
	_battleLocalRecordBytes.Inc(size)
}

func ReduceBattleRoomNum(battleType string) {
	if _room_ccu != nil {
		_room_ccu.Dec(1)
	}
	k := battleTypeRoomNumPrefix(battleType)
	_battleTypeRoomCCULock.RLock()
	v, ok := _battleTypeRoomCCU[k]
	_battleTypeRoomCCULock.RUnlock()
	if !ok {
		return
	}
	v.Dec(1)
}

func OnBattleRev(n int) {
	if _rev != nil {
		_rev.Inc(1)
	}
	if _rev_size != nil {
		_rev_size.Inc(int64(n))
	}
}

func OnBattleSend(n int) {
	if _send != nil {
		_send.Inc(1)
	}
	if _send_size != nil {
		_send_size.Inc(int64(n))
	}
}

func OnBattleWillSend() {
	if _willSend != nil {
		_willSend.Inc(1)
	}
}

func UpdateBattleRobotRS(max, min, mid, avg int64) {
	if _maxRS != nil && max > 0 {
		_maxRS.Update(max)
	}
	if _minRS != nil && min > 0 {
		_minRS.Update(min)
	}
	if _midRS != nil && mid > 0 {
		_midRS.Update(mid)
	}
	if _avgRS != nil && avg > 0 {
		_avgRS.Update(avg)
	}
}

func UpdateReadCost(v int64) {
	if _readCost != nil {
		_readCost.Update(v)
	}
}

func UpdateBattleRobotKcpRS(v int64) {
	if _kcpRs != nil {
		_kcpRs.Update(v)
	}
}

func UpdateWriteCost(v int64) {
	if _writeCost != nil {
		_writeCost.Update(v)
	}
}

func battleTypeRoomNumPrefix(battleType string) string {
	return fmt.Sprintf("battletyperoomnum.%s", battleType)
}
func battleTypePlayerCCUPrefix(battleType string) string {
	return fmt.Sprintf("battletypeplayernum.%s", battleType)
}

func AddBattleAgentCCU() {
	_battle_ccu.Inc(1)
}

func ReduceBattleAgentCCU() {
	_battle_ccu.Dec(1)
}

func OnBattleAgentRev(n int) {
	if _rev != nil {
		_rev.Inc(1)
	}
	//if _rev_agent_size != nil {
	//	_rev_agent_size.Inc(int64(n))
	//}
}

func OnBattleAgentSend(n int) {
	if _send != nil {
		_send.Inc(1)
	}
	//if _send_agent_size != nil {
	//	_send_agent_size.Inc(int64(n))
	//}
}
