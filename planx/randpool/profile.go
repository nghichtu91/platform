package randpool

/*
	Profile管理Counter
*/

// IRandomProfile 管理Counter
type IRandomProfile interface {
	GetCounter(id string) RandomCounter
	SetCounter(id string, rc RandomCounter)
	DelCounter(id string)
	CleanUpCounters(timeNow int64)
}

// DefaultRandomProfile 默认profile，无锁，跨线程不安全
// 如果有多线程使用需求，需要自己实现SafeRandomProfile
type DefaultRandomProfile struct {
	SavedCounters map[string]RandomCounter
}

// DefaultProfile 生成一个默认的profile
func DefaultProfile() IRandomProfile {
	return &DefaultRandomProfile{
		SavedCounters: make(map[string]RandomCounter),
	}
}

// GetCounter 获取一个id对应的counter
func (rp *DefaultRandomProfile) GetCounter(id string) RandomCounter {
	if c, ok := rp.SavedCounters[id]; ok {
		return c
	}
	return nil
}

// SetCounter 存储一个id对应的counter
func (rp *DefaultRandomProfile) SetCounter(id string, c RandomCounter) {
	rp.SavedCounters[id] = c
}

// DelCounter 删除一个counter
func (rp *DefaultRandomProfile) DelCounter(id string) {
	delete(rp.SavedCounters, id)
}

// CleanUpCounters 遍历当前所有counter，并删除已经过期的counter
func (rp *DefaultRandomProfile) CleanUpCounters(timeNow int64) {
	for id, c := range rp.SavedCounters {
		if !c.IsAvailable(timeNow) {
			delete(rp.SavedCounters, id)
		}
	}
}
