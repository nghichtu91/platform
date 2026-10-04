package merge_util

import (
	"errors"

	"github.com/nghichtu91/platform/share/planx/mergeinfo"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

var (
	ErrManagerNotInit = errors.New("merge info manager not init")
	ErrShardNotExist  = errors.New("shard id not found")
)

const (
	NotExistsID = mergeinfo.NotExistsID
)

var (
	mm mergeinfo.IManager
)

// InitByEtcd 使用etcd管理合服
func InitByEtcd(devOpsPath, SrvPath string) (mergeinfo.IManager, error) {
	mgr := mergeinfo.NewEtcdMgr(devOpsPath, SrvPath)
	err := mgr.Init()
	if err != nil {
		return nil, err
	}
	SetMergeManager(mgr)
	return mm, nil
}

func SetMergeManager(mgr mergeinfo.IManager) {
	mm = mgr
}

// GetGIDBySID 根据SID查询所属GID
func GetGIDBySID(sid int) (int, error) {
	if mm == nil {
		return 0, ErrManagerNotInit
	}
	gid := mm.GetGIDBySID(sid)
	if gid == mergeinfo.NotExistsID {
		return 0, ErrShardNotExist
	}
	return gid, nil
}

// GetRealSIDBySID 根据SID查询主服SID
func GetRealSIDBySID(sid int) (int, error) {
	if mm == nil {
		return 0, ErrManagerNotInit
	}
	realSid := mm.GetRealSIDBySID(sid)
	if realSid == mergeinfo.NotExistsID {
		return 0, ErrShardNotExist
	}
	return realSid, nil
}

func GetRealSIDByAcid(acid string) (int, error) {
	sid, err := GetRealSIDBySID(util.ParseShardIDFromAcid(acid))
	if err != nil {
		tilogs.L().Debugf("get real sid %d %v", sid, err)
	}
	return sid, err
}

// IsSameRealSID 判断两个SID是否属于同一个主服
// 只有两个SID都存在主ID，且主ID一致的情况下返回true
// 如果任意id不存在或有错误，均返回false
func IsSameRealSID(sid1, sid2 int) (bool, error) {
	if mm == nil {
		return false, ErrManagerNotInit
	}

	s1, s2 := mm.GetRealSIDBySID(sid1), mm.GetRealSIDBySID(sid2)

	return s1 == s2 && s1 != mergeinfo.NotExistsID, nil
}

// GetSIDsFromReal 查询指定主服ID包括了哪些服务器ID
func GetSIDsFromReal(realSids ...int) ([]int, error) {
	if mm == nil {
		return nil, ErrManagerNotInit
	}

	ret := make([]int, 0, len(realSids)*4)
	for _, realSid := range realSids {
		si := mm.GetShardInfoBySID(realSid)
		if si == nil {
			tilogs.L().Warnf("GetSIDsFromReal realSid %d not found", realSid)
			continue
		}
		ret = append(ret, si.ShardIDs...)
	}

	return ret, nil
}

// GetSIDsFromRealOrSelf 查询指定主服ID包括了哪些服务器ID
// 如果传进来的不是主服id，则会返回服务器id本身
func GetSIDsFromRealOrSelf(realSids ...int) ([]int, error) {
	if mm == nil {
		return nil, ErrManagerNotInit
	}

	ret := make([]int, 0, len(realSids)*4)
	for _, realSid := range realSids {
		si := mm.GetShardInfoBySID(realSid)
		if si == nil { // 查询不到合服相关服务器信息，则是还没有合服，返回自身即可
			ret = append(ret, realSid)
			continue
		}
		ret = append(ret, si.ShardIDs...)
	}

	return ret, nil
}

// GetSIDsFromRealUint32 查询指定主服ID包括了哪些服务器ID，这个接口的输入和返回都是uint32
// 如果服务器id中传入了非主服id，会跳过
func GetSIDsFromRealUint32(shardIDs ...uint32) ([]uint32, error) {
	if mm == nil {
		return nil, ErrManagerNotInit
	}

	ret := make([]uint32, 0, len(shardIDs)*4)
	for _, realSid := range shardIDs {
		if !IsMainShard(int(realSid)) {
			continue
		}
		si := mm.GetShardInfoBySID(int(realSid))
		if si == nil {
			ret = append(ret, realSid)
			// tilogs.L().Warnf("GetSIDsFromReal realSid %d not found", realSid)
			continue
		}
		for i := 0; i < len(si.ShardIDs); i++ {
			ret = append(ret, uint32(si.ShardIDs[i]))
		}
	}

	return ret, nil
}

// IsMainShard 判断shard id是否为主服
func IsMainShard(sid int) bool {
	if mm == nil {
		return false
	}

	return mm.GetRealSIDBySID(sid) == sid
}
