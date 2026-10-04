package mergeinfo

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util/collection"
)

var (
	ErrInvalidSid2GidInfo  = errors.New("invalid sid to gid info") // sid到gid的映射表数据有错误
	ErrInvalidShardInfo    = errors.New("invalid shard info")      // shard信息有错误
	ErrShardIDHasNoGID     = errors.New("shard id has no gid")     // sid到gid的映射表里没有这个服
	ErrDuplicateMainSID    = errors.New("duplicate main sid")      // 发现重复的主服sid
	ErrDuplicateSubSID     = errors.New("duplicate sub sid")       // 发现重复的子服sid
	ErrTargetShardIsOnline = errors.New("target shard is online")  // 如果目标shard未关闭多人玩法或离线，不允许更新合服信息

	ErrMainBiggerThanSub = errors.New("main shard > sub shard") // 主服的shard > 子服的shard
)

const (
	NotExistsID = -1
)

var (
	skipOldMergeCheck = true // 是否跳过强校验
)

// ManagerV1 用于管理合服信息
// V1版本使用锁进行并发管理，支持所有数据的热加/热减
// 会根据逻辑决定是否能进行数据修改
type ManagerV1 struct {
	// mainShards 主服列表
	// subShards 被合服列表
	// 主要用于做逻辑判断，是否有sid冲突
	mainShards map[int]struct{}
	subShards  map[int]struct{}

	// sid2gid 保存当前的gid和sid映射关系
	// 用于映射关系变化时快捷对比哪些shard发生了变化
	sid2gid map[int]int

	// 合服关系信息
	shardInfo map[int]*ShardInfo

	// 更新状态跟踪
	verInfo VerInfo

	mutex sync.RWMutex
}

// NewManagerV1 创建Manager
func NewManagerV1() *ManagerV1 {
	return &ManagerV1{
		mainShards: make(map[int]struct{}, defaultShardSize),
		subShards:  make(map[int]struct{}, defaultShardSize),
		sid2gid:    make(map[int]int, defaultShardSize),
		shardInfo:  make(map[int]*ShardInfo, defaultShardSize),
	}
}

// VerInfo 用于同步当前数据版本信息
type VerInfo struct {
	G2STag    string `json:"g2s_tag"` // gid2sid当前tag
	G2STs     int64  `json:"g2s_ts"`  // gid2sid更新时间戳
	ShardsTag string `json:"ss_tag"`  // 服务信息tag
	ShardsTs  int64  `json:"ss_ts"`   // 服务信息更新时间戳
}

// GetShardInfoBySID 查询对应SID的信息
func (m *ManagerV1) GetShardInfoBySID(sid int) *ShardInfo {
	if m == nil {
		return nil
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.shardInfo[sid]
}

// GetGIDBySID 根据SID查询所属GID
// 不存在或未启动的Shard返回-1
// 未启动的Shard返回GID2SID配置内的GID
func (m *ManagerV1) GetGIDBySID(sid int) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	gid, ok := m.sid2gid[sid]
	if !ok {
		return NotExistsID
	}
	return gid
}

// GetRealSIDBySID 根据SID查询主服SID
// 不存在的Shard返回-1
// 未启动的Shard返回GID2SID配置内的SID
func (m *ManagerV1) GetRealSIDBySID(sid int) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	si, ok := m.shardInfo[sid]
	if !ok || si == nil {
		if _, ok := m.sid2gid[sid]; !ok {
			return NotExistsID
		}
		return sid
	}
	return si.SID
}

// GetVerInfo 当前合服信息
// 由于每个Manager实现不一样，使用通用的json格式作为返回值，方便解析
// Etcd版本返回两个watch键值的相关信息
func (m *ManagerV1) GetVerInfo() []byte {
	raw, _ := json.Marshal(m.verInfo)
	return raw
}

// SetInfo 更新所有的合服信息
// 这个方法会丢弃原来的合服数据，替换为当前信息
// 初始化或需要重设信息时，使用此方法
func (m *ManagerV1) SetInfo(s2g Sid2GidInfo, si ...*ShardInfo) error {
	// 没变化，不用更新
	if genMD5(s2g) == m.verInfo.G2STag && genMD5(si) == m.verInfo.ShardsTag {
		return nil
	}

	// 将数据全部转换并校验后，没有问题再替换Manager内变量
	nsi := make(map[int]*ShardInfo, defaultShardSize)
	nms := make(map[int]struct{}, defaultShardSize)
	nns := make(map[int]struct{}, defaultShardSize)
	ns2g := make(map[int]int, defaultShardSize)

	// shard to group
	if !s2g.isValid() {
		return ErrInvalidSid2GidInfo
	}

	for _, info := range s2g {
		for _, sid := range info.ShardIDs {
			ns2g[sid] = info.GID
		}
	}

	tilogs.L().Debugf("%d shards added", len(ns2g))
	tilogs.L().Infof("merge_manager, SetInfo ns2g:%v", ns2g)

	// 更新每个服的状态
	// 由于被合的服会留下最后一次关闭时的状态，初始化时需要忽略这部分shard
	var err error
	for _, info := range si {
		if info == nil || info.SID == 0 || info.Status == SrvStatusOffline {
			continue
		}

		// tilogs.L().Infof("merge_manager, mainShard Check, sid:%d", info.SID)

		// 主服检查
		if _, ok := ns2g[info.SID]; !ok {
			err = ErrShardIDHasNoGID
		}
		if _, ok := nsi[info.SID]; ok {
			err = ErrDuplicateMainSID
		}
		if _, ok := nms[info.SID]; ok {
			err = ErrDuplicateMainSID
		}

		if err != nil {
			tilogs.L().Errorf("shard %d added as main shard failed", info.SID)
			return err
		}

		// info.GID = ns2g[info.SID]
		nsi[info.SID] = info
		nms[info.SID] = struct{}{}

		// 刷新子服状态
		if len(nsi[info.SID].ShardIDs) > 1 {
			tilogs.L().Infof("merge_manager, childShard Check, sid:%v", nsi[info.SID].ShardIDs)
			for _, subID := range nsi[info.SID].ShardIDs {
				if subID == info.SID {
					continue
				}

				// 子服检查
				if _, ok := ns2g[subID]; !ok {
					err = ErrShardIDHasNoGID
				}
				if _, ok := nsi[subID]; ok {
					err = ErrDuplicateSubSID
				}
				if _, ok := nns[subID]; ok {
					err = ErrDuplicateSubSID
				}

				if err != nil {
					tilogs.L().Errorf("shard %d added as sub shard failed", subID)
					return err
				}

				nns[subID] = struct{}{}
				// 更新子服对应信息为主服
				if nsi[subID] == nil {
					nsi[subID] = NewShardInfo()
				}
				// nsi[subID].GID = ns2g[subID]
				nsi[subID].SID = info.SID
				nsi[subID].ShardBaseInfo = info.ShardBaseInfo
			}
		}
	}

	// 到这里还没有错误，说明数据有效
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.shardInfo = nsi
	m.mainShards = nms
	m.subShards = nns
	m.sid2gid = ns2g

	m.saveS2GInfo(s2g)
	m.saveSInfo(si...)

	tilogs.L().Infof("merge manager update all success, %d shards(%d main and %d sub) info updated, ver %+v",
		len(nsi), len(nms), len(nns), m.verInfo)

	return nil
}

func (m *ManagerV1) SetInfoV2(s2g Sid2GidInfo, shards ...*ShardInfo) error {

	// 没变化，不用更新
	if genMD5(s2g) == m.verInfo.G2STag && genMD5(shards) == m.verInfo.ShardsTag {
		return nil
	}

	// shard to group
	if !s2g.isValid() {
		return ErrInvalidSid2GidInfo
	}

	shardsInfo := make(map[int]*ShardInfo, len(shards)) // 所有shard -> ShardInfo的映射
	shard2GID := make(map[int]int, defaultShardSize)    // shard -> GID
	subShards := collection.NewIntSetSize(len(shards))  // 所有子服的集合

	// 构建shard并查集
	var shardUFSet = NewIntUFSet(len(shards) / 8)        // 这个8没啥意义, 纯粹就是瞎写的. 经验值
	var shardSet = collection.NewIntSetSize(len(shards)) // 所有服务器集合

	// 搭建 Shard -> Gid 的映射
	for _, info := range s2g {
		for _, sid := range info.ShardIDs {
			shard2GID[sid] = info.GID
		}
	}
	tilogs.L().Debugf("%d shards added", len(shard2GID))
	tilogs.L().Infof("merge_manager, SetInfo shard2GID:%v", shard2GID)

	var err error

	// 一个 ShardInfo.ShardIDs 构建的集合, 减少内存分配次数, 做到空间的复用
	var subSet = collection.NewIntSet()

	/*
		正常的合服, 只能是如下的情况
		1. 只能高Shard合并到低Shard
		2. 被合并后, 主ShardInfo.ShardIDs 中包括所有的 子 Shard

		比如: [1,2合并] -> [3,4合并] -> [1,2,3,4合并] 后, 对应的 ShardInfo 应该是这样的:
		{SID:1, ShardIDs: [1,2,3,4]}
		{SID:2, ShardIDs: [2]}
		{SID:3, ShardIDs: [3,4]}
		{SID:4, ShardIDs: [4]}
	*/

	/*
		相较于v1:
		1. 无视服务器的停服状态(如果后续有特殊判定, 可以继续修改)
		2. 允许任意服务器多次出现, 通过并查集确定最终一致性
		3. 允许无序迭代, 允许 ShardInfo.ShardIDs 乱序
	*/

	// 合并子服->主服映射关系
	mergeSubToRoot := func(sub, root int) error {
		// 保证主服和子服之间的大小关系
		if checkErr := checkSubBiggerMain(sub, root); checkErr != nil {
			return checkErr
		}
		// 主服必须已经存在
		mainInfo := shardsInfo[root]
		if mainInfo == nil {
			return ErrInvalidShardInfo
		}
		// 设置 子服的root为主服的root
		shardUFSet.SetRoot(sub, root)
		// 设置子服标记. 只要是被标记为子服了, 就永远不可能再成为主服了
		subShards.AddNoCheck(sub)
		// 设置子服信息等同于主服
		subInfo := shardsInfo[sub]
		if subInfo == nil {
			// 没有就new一个, 有就原地更新
			subInfo = NewShardInfo()
			shardsInfo[sub] = subInfo
		}
		// 子服的信息以主服为准
		subInfo.SID = mainInfo.SID
		subInfo.ShardBaseInfo = mainInfo.ShardBaseInfo
		return nil
	}

	// 一次迭代所有的服务器信息
	for _, shard := range shards {
		// TODO 如果后续存在特殊的状态判定, 从这里入手
		if shard == nil || shard.SID == 0 {
			continue
		}

		// 主服唯一性验证
		if shardSet.Contain(shard.SID) {
			tilogs.L().Errorf("find duplicate main shard %v", shard.SID)
			err = ErrDuplicateMainSID
			return err
		}
		shardSet.Add(shard.SID)

		// 将所有子服(包括自身)添加到集合中, 应对下边的检查
		subSet.Clear()

		// 检查GID, 添加到 子服 set 中
		for _, sub := range shard.ShardIDs {
			if _, ok := shard2GID[shard.SID]; !ok {
				err = ErrShardIDHasNoGID
				tilogs.L().Errorf("cannot find main %v shard %v gid", shard.SID, sub)
				return err
			}
			subSet.Add(sub)
		}

		// 子服队列中一定要包含自身
		if !subSet.Contain(shard.SID) {
			tilogs.L().Errorf("%v cannot find main in shardIDs", shard.String())
			return ErrInvalidShardInfo
		}

		// 获取主服的root
		mainRoot, mainIsNew := shardUFSet.GetRoot(shard.SID)
		if mainIsNew {

			// 绑定 shard -> ShardInfo
			shardsInfo[shard.SID] = shard.DeepClone()

			// 主服是一个新增节点,
			//		如果子服中存在老节点, 必须要保证所有的老节点对应的根节点都在 当前节点的 ShardInfo.ShardIDs 中

			// 迭代所有的子服
			for _, sub := range shard.ShardIDs {
				if sub == shard.SID {
					// 跳过自己
					continue
				}
				// 判断子服是不是一个新节点
				subRoot, subIsNew := shardUFSet.GetRoot(sub)
				if !subIsNew {
					// 如果子服不是一个新增节点

					// 老节点需要保证原来的根不能小于现在的根
					if err = checkSubBiggerMain(subRoot, mainRoot); err != nil {
						// 错误情况: 主服不应该大于子服
						tilogs.L().Errorf(err.Error())
						return err
					}

					if !subSet.Contain(subRoot) {
						// 错误情况: 迭代顺序 {SID: 3, ShardIDs: [3,4]} -> {SID: 1, ShardIDs: [1,2,4]}
						// 先迭代 [3,4], 此时 shard -> root 的关系为 [3:3, 4:3]
						// 当迭代到 [1,2,4]时, 4不是一个新节点, 但是[1,2,4]中不包括3, 不符合预期

						// 为啥要通过set判断呢? 为了防止潜在的乱序
						// 比如, 如果 存储的顺序变成了 [4,1,2,3], 不通过set就会出现误判(排个序应该也行(?))
						err = invalidMergeErr(shard.SID, sub)
						tilogs.L().Errorf("%v. shard info: %v. sub %v root %v not in ShardIDs",
							err, shard.String(), sub, subRoot)
						return err
					}
				}
				// 如果子服是一个新节点, 可以直接合
				if err = mergeSubToRoot(sub, mainRoot); err != nil {
					tilogs.L().Errorf("shardUFSet sub %v root %v error %v", sub, mainRoot, err)
					return err
				}
			}
		} else {
			// 主服不是一个新增节点, 则其对应的所有子服都应该不是新增的节点, 并且所有子服都应该指向相同的根节点

			// TODO 如果可以保证过往数据不出问题的话, 这段检查可以直接跳过
			// 	比如先迭代 {SID:1, ShardIDs:[1,2,3,4]},
			//  再迭代的 {SID:3, ShardIDs:[3,4]}, 此时3的root为1, 可以跳过相关的检查
			// 	这一块相当于对整体的合法性进行了强校验

			if err = checkSubBiggerMain(shard.SID, mainRoot); err != nil {
				// 错误情况: 当前服务器有根节点, 就需要保证根节点一定是小于自身的
				tilogs.L().Errorf(err.Error())
				return err
			}

			if skipOldMergeCheck {
				// 暂时跳过检查
				continue
			}

			for _, sub := range shard.ShardIDs {
				if sub == shard.SID {
					// 跳过自己
					continue
				}
				subRoot, subIsNew := shardUFSet.GetRoot(sub)

				if subIsNew {
					// 子服不应该是一个新节点, 因为应该在之前某次迭代时添加进去了
					// 错误情况: 迭代顺序 {SID: 1, [1,2,3,4]} -> {SID: 3, [3,4,5,6]},
					// 先迭代 [1,2,3,4], 此时 shard -> root 的关系为 [1:1, 2:1, 3:1, 4:1]
					// 当迭代到 [3,4,5,6]时, 3的根为1, 不是新增的节点; 但是 [5,6] 均为新增的节点, 不符合预期
					err = invalidMergeErr(shard.SID, sub)
					tilogs.L().Errorf("%v. shard info: %v. sub is new node, but main is old node",
						err, shard.String())
					return err
				}
				if subRoot != mainRoot {
					// 子服的root必须要和主服的root是一致的
					// 错误情况: 迭代顺序 {SID: 1, [1,2]} -> {SID: 3, [3,4]} -> {SID: 2, [2,3]}
					// 先迭代 [1,2], 此时 shard -> root 的关系为 [1:1, 2:1]
					// 再迭代 [3,4], 此时 shard -> root 的关系为 [1:1, 2:1, 3:3, 4:3]
					// 当迭代到 [2,3]时, 2的根节点是1, 3的根节点3, 不符合预期
					err = invalidMergeErr(shard.SID, sub)
					tilogs.L().Errorf("%v. shard info: %v. main %v root %v, sub %v root %v not equal",
						err, shard.String(), shard.SID, mainRoot, sub, subRoot)
					return err
				}

				// 设置 子服的root指向主服的root
				if err = mergeSubToRoot(sub, mainRoot); err != nil {
					tilogs.L().Errorf("shardUFSet sub %v root %v error %v", sub, mainRoot, err)
					return err
				}
			}
		}
	}

	mainShards := shardUFSet.GetRoots() // 所有的主服. 就是并查集最后所有的根节点

	// 主服集合 和 子服集合 二者的交集应该为空
	if inter := collection.IntSet(mainShards).Intersection(subShards); inter.Len() != 0 {
		tilogs.L().Errorf("main shards intersection sub shards %v not  empty", inter)
		return ErrInvalidShardInfo
	}

	// 到这里还没有错误，说明数据有效
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.shardInfo = shardsInfo
	m.mainShards = mainShards
	m.subShards = subShards
	m.sid2gid = shard2GID

	m.saveS2GInfo(s2g)
	m.saveSInfo(shards...)

	tilogs.L().Infof("merge manager update all success, %d shards(%d main and %d sub) info updated, ver %+v",
		len(shardsInfo), len(mainShards), len(subShards), m.verInfo)

	return nil
}

// UpdateSid2Gid 更新GID映射信息
// 如果开启状态检查，则状态发生变化的shard必须为关闭或多人玩法离线
func (m *ManagerV1) UpdateSid2Gid(s2g Sid2GidInfo) error {
	// 和上次一样，不需要更新
	if genMD5(s2g) == m.verInfo.G2STag {
		return nil
	}

	if !s2g.isValid() {
		return ErrInvalidSid2GidInfo
	}

	// 都是 sid:gid 的形式
	adds := make(map[int]int)                // 增
	dels := make([]int, 0, defaultGroupSize) // 删
	changes := make(map[int]int)             // 改
	na := make(map[int]struct{})             // new all shards

	m.mutex.RLock()
	for _, info := range s2g {
		ng := info.GID
		for _, ns := range info.ShardIDs {
			na[ns] = struct{}{}
			g, ok := m.sid2gid[ns]
			if ok {
				if g == ng {
					continue
				} else {
					// 改
					changes[ns] = ng
				}
			} else {
				// 增
				adds[ns] = ng
			}
		}
	}

	// 删
	for s := range m.sid2gid {
		if _, ok := na[s]; !ok {
			dels = append(dels, s)
		}
	}

	// 检查删和改的sid状态是否为在线
	var isInvalid bool
	for _, sid := range dels {
		if m.shardInfo[sid] != nil && m.shardInfo[sid].Status == SrvStatusMultiOn {
			isInvalid = true
			tilogs.L().Errorf("shard %d del from group failed, multiplay on")
		}
	}
	for _, sid := range changes {
		if m.shardInfo[sid] != nil && m.shardInfo[sid].Status == SrvStatusMultiOn {
			isInvalid = true
			tilogs.L().Errorf("shard %d change gid failed, multiplay on")
		}
	}
	m.mutex.RUnlock()

	if isInvalid {
		return ErrTargetShardIsOnline
	}

	// 下面开始写操作
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 增
	for s, g := range adds {
		m.sid2gid[s] = g
	}

	// 改
	for s, g := range changes {
		m.changeInMutex(s, g)
	}

	// 删
	for _, s := range dels {
		m.delInMutex(s)
	}

	tilogs.L().Infof("updateSid2Gid success, total shards is %d now, %d added, %d deleted and %d changed",
		len(na), len(adds), len(dels), len(changes))

	m.saveS2GInfo(s2g)

	return nil
}

// UpdateShard 更新单个Shard信息
func (m *ManagerV1) UpdateShard(info *ShardInfo) error {
	if !m.isUpdateValid(info) {
		return ErrTargetShardIsOnline
	}
	if err := m.setShardInfo(info); err != nil {
		return err
	}
	return nil
}

// Clone 复制单个Manager
// 用于检查下一批数据正确性等不能改变原数据的情况
// 复制出的Manager所有数据都与原Manager独立
func (m *ManagerV1) Clone() IManager {
	nm := NewManagerV1()

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for k, v := range m.sid2gid {
		nm.sid2gid[k] = v
	}
	for k := range m.mainShards {
		nm.mainShards[k] = struct{}{}
	}
	for k := range m.subShards {
		nm.subShards[k] = struct{}{}
	}
	for k, v := range m.shardInfo {
		nm.shardInfo[k] = v.DeepClone()
	}

	nm.verInfo = m.verInfo

	return nm
}

// isUpdateValid 检查当前更新操作是否合法
// 如果某个服状态是MultiOn，不应该继续进行更新流程
func (m *ManagerV1) isUpdateValid(info *ShardInfo) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if info == nil {
		return false
	}

	if info.Status == SrvStatusMultiOn {
		for _, sid := range info.ShardIDs {
			if m.shardInfo[sid].Status == SrvStatusMultiOn {
				tilogs.L().Errorf("sid %d is multi on", sid)
				return false
			}
		}
	}

	return true
}

// setShardInfo 更新shard信息
// 如果shard状态为Offline，会删除对应的Shard信息
// 其他状态则更新
func (m *ManagerV1) setShardInfo(info *ShardInfo) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	tilogs.L().Infof("setShardInfo info:%v", info)
	// Offline
	if info.Status == SrvStatusOffline {
		// 停服不需要删除合服关系
		//for _, sid := range info.ShardIDs {
		//	if sid == info.SID {
		//		continue
		//	}
		//	delete(m.subShards, sid)
		//	delete(m.shardInfo, sid)
		//}
		//
		//// 删主服
		//delete(m.mainShards, info.SID)
		//delete(m.shardInfo, info.SID)
	} else {
		// 主服
		m.shardInfo[info.SID] = info
		m.mainShards[info.SID] = struct{}{}
		delete(m.subShards, info.SID)

		// 子服
		for _, sid := range info.ShardIDs {
			if sid == info.SID {
				continue
			}
			m.shardInfo[sid] = info
			delete(m.mainShards, sid)
			m.subShards[sid] = struct{}{}
		}
	}

	m.saveSInfo(info)

	return nil
}

func (m *ManagerV1) saveS2GInfo(s2g Sid2GidInfo) {
	m.verInfo.G2STag = genMD5(s2g)
	m.verInfo.G2STs = time.Now().Unix()
}

func (m *ManagerV1) saveSInfo(si ...*ShardInfo) {
	m.verInfo.ShardsTag = genMD5(si)
	m.verInfo.ShardsTs = time.Now().Unix()
}

// delInMutex 在锁内进行的删除操作
func (m *ManagerV1) delInMutex(sid int) {
	delete(m.sid2gid, sid)
	delete(m.mainShards, sid)
	delete(m.subShards, sid)
	// 如果某个服需要被删除并可以删除，那所有包括的sid应该都被删掉
	if m.shardInfo[sid] != nil {
		for _, ss := range m.shardInfo[sid].ShardIDs {
			delete(m.sid2gid, ss)
			delete(m.shardInfo, ss)
			delete(m.mainShards, ss)
			delete(m.subShards, ss)
		}
		delete(m.shardInfo, sid)
	}
}

// changeInMutex 在锁内进行的修改操作
func (m *ManagerV1) changeInMutex(sid, gid int) {
	m.sid2gid[sid] = gid
}

// Stop 实现接口
func (m *ManagerV1) Stop() {
	tilogs.L().Infof("mergeinfo manager v1 stopped")
}

// 检查是否会出现 mainShard > subShard 的情况
func checkSubBiggerMain(sub, main int) error {
	if main > sub {
		return fmt.Errorf("%w, main shard %v, sub shard %v", ErrMainBiggerThanSub, main, sub)
	}
	return nil
}

// 通用的错误包装
func invalidMergeErr(main, sub int) error {
	return fmt.Errorf("invalid merge: main:%v, sub:%v", main, sub)
}
