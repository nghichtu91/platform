package allmetrics

import (
	"fmt"
	"sync"

	gm "github.com/rcrowley/go-metrics"
	"github.com/nghichtu91/platform/share/planx/metrics"
)

var (
	_sceneLineCount map[string]gm.Gauge
	_slcLock        sync.RWMutex

	_sceneMapWithLineCount map[string]gm.Counter
	_smlLock               sync.RWMutex

	_sceneNPCMap map[string]gm.Counter
	_snmLock     sync.RWMutex
)

func PrefixSceneMetrics(gid, sid uint) string {
	return fmt.Sprintf("%s.%d.%d", ScenePrefix, gid, sid)
}

func InitSceneMetrics() {
	_sceneLineCount = make(map[string]gm.Gauge, 64)
	_sceneMapWithLineCount = make(map[string]gm.Counter, 64)
	_sceneNPCMap = make(map[string]gm.Counter, 4)
}

func sceneLineCountPrefix(key string) string {
	return fmt.Sprintf("scenelinecount.%s", key)
}

func AddSceneMetric(key string) {
	key = sceneLineCountPrefix(key)
	_slcLock.RLock()
	_, ok := _sceneLineCount[key]
	_slcLock.RUnlock()
	if !ok {
		_slcLock.Lock()
		if _, ok := _sceneLineCount[key]; !ok {
			_sceneLineCount[key] = metrics.NewGauge(key)
		}
		_slcLock.Unlock()
	}
}

func UpdateSceneMetric(key string, count int64) {
	key = sceneLineCountPrefix(key)
	_slcLock.RLock()
	defer _slcLock.RUnlock()
	sl, ok := _sceneLineCount[key]
	if !ok {
		return
	}

	sl.Update(count)
}

func sceneMapWithLineCountPrefix(key string) string {
	return fmt.Sprintf("scenemaplinecount.%s", key)
}

func AddSceneMapMetric(key string) {
	key = sceneMapWithLineCountPrefix(key)
	_smlLock.RLock()
	sl, ok := _sceneMapWithLineCount[key]
	_smlLock.RUnlock()
	if !ok {
		_smlLock.Lock()
		if sl, ok = _sceneMapWithLineCount[key]; !ok {
			sl = metrics.NewCounter(key)
			_sceneMapWithLineCount[key] = sl
		}
		_smlLock.Unlock()
	}
	sl.Inc(1)
}

func ReduceSceneMapMetric(key string) {
	key = sceneMapWithLineCountPrefix(key)
	_smlLock.RLock()
	defer _smlLock.RUnlock()
	sl, ok := _sceneMapWithLineCount[key]
	if !ok {
		return
	}
	sl.Dec(1)
}

func sceneNPCMapPrefix(key string) string {
	return fmt.Sprintf("scenenpcmap.%s", key)
}

func AddSceneNPCMapMetric(key string) {
	key = sceneNPCMapPrefix(key)
	_snmLock.RLock()
	sl, ok := _sceneNPCMap[key]
	_snmLock.RUnlock()
	if !ok {
		_snmLock.Lock()
		if sl, ok = _sceneNPCMap[key]; !ok {
			sl = metrics.NewCounter(key)
			_sceneNPCMap[key] = sl
		}
		_snmLock.Unlock()
	}
	sl.Inc(1)
}

func ReduceNPCSceneMapMetric(key string) {
	key = sceneNPCMapPrefix(key)
	_snmLock.RLock()
	defer _snmLock.RUnlock()
	sl, ok := _sceneNPCMap[key]
	if !ok {
		return
	}
	sl.Dec(1)
}
