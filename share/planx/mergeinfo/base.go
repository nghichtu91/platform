package mergeinfo

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
)

const (
	SrvStatusOffline  = iota // 离线
	SrvStatusMultiOff        // 多人玩法离线，需要跨服服务的玩法不可用
	SrvStatusMultiOn         // 多人玩法在线，可以进行所有多人玩法
)

const (
	defaultSIDSize = 64 // 默认分配的合服区间大小
)

// ShardInfo 服务器状态
// 如果MainSID为0，表示这个服没有合服状态，认定其为离线
// 也用于Unmarshal etcd上的json字符串
type ShardInfo struct {
	// GID            int `json:"gid,omitempty" etcd3:"gid"` // 大区ID
	SID            int `json:"sid" etcd3:"sid"` // 对应主服ID
	*ShardBaseInfo     // 子服的指针指向主服
}

// ShardBaseInfo 服务器的合服状态数据
type ShardBaseInfo struct {
	Shards
	Status int `json:"status" etcd3:"status"` // 服务器当前状态，默认0为离线状态
}

// Shards 服务器区间的string形式和int slice形式
type Shards struct {
	ShardsStr string `json:"sss" etcd3:"sss"` // shard区间的字符串形式
	ShardIDs  []int  `json:"-" etcd3:"-"`     // 包括的服务器列表
}

// Sid2GidMapInfo ShardID和GID的映射总表
// 这个结构主要用于Unmarshal etcd上的json字符串
type Sid2GidInfo []*Sid2Gid

// Sid2Gid ShardID和GID的映射结构
// 支持区间形式
type Sid2Gid struct {
	GID int `json:"gid"` // 大区ID
	Shards
}

// I2S 服务列表int slice转字符串形式
func (s *Shards) I2S() string {
	s.ShardsStr = ""
	for i := 0; i < len(s.ShardIDs); i++ {
		if i != 0 {
			s.ShardsStr += ","
		}
		s.ShardsStr += strconv.Itoa(s.ShardIDs[i])
	}

	return s.ShardsStr
}

// S2I 服务列表字符串形式转int slice
func (s *Shards) S2I() ([]int, error) {
	ss, err := ParseRangeStr(s.ShardsStr)
	if err != nil {
		return nil, err
	}

	// 生成sid序列
	s.ShardIDs = make([]int, 0, defaultSIDSize)
	for _, r := range ss {
		if len(r) != 2 {
			return nil, fmt.Errorf("invalid range %v from %s", ss, s.ShardsStr)
		}
		st, et := r[0], r[1]
		if st > et {
			return nil, fmt.Errorf("invalid range %v from %s", ss, s.ShardsStr)
		}
		if st == et {
			s.ShardIDs = append(s.ShardIDs, st)
		} else {
			for ; st < et+1; st++ {
				s.ShardIDs = append(s.ShardIDs, st)
			}
		}
	}

	return s.ShardIDs, nil
}

// NewSid2Gid 新建
func NewSid2Gid() *Sid2GidInfo {
	return new(Sid2GidInfo)
}

// NewSid2GidInfoFromJson 使用json内容新建
// 会进行有效性检查，并解析对应数据
func NewSid2GidInfoFromJson(v []byte) (Sid2GidInfo, error) {
	sg := Sid2GidInfo{}
	err := sg.SetJsonValue(v)
	if err != nil {
		return nil, err
	}
	return sg, nil
}

// SetJsonValue 使用json内容赋值
// 会进行有效性检查，并解析对应数据
func (info *Sid2GidInfo) SetJsonValue(v []byte) error {
	err := json.Unmarshal(v, &info)
	if err != nil {
		return err
	}
	if !info.isValid() {
		return ErrInvalidSid2GidInfo
	}
	return nil
}

// s2i 所有服务列表字符串形式转int slice
func (info *Sid2GidInfo) s2i() error {
	for i := 0; i < len(*info); i++ {
		_, err := (*info)[i].S2I()
		if err != nil {
			return err
		}
	}
	return nil
}

// validate 检验数据是否存在冲突，并根据shard string赋值
// gid和sid均不能重复
func (info *Sid2GidInfo) isValid() bool {
	if info == nil || len(*info) == 0 {
		return false
	}

	if err := info.s2i(); err != nil {
		return false
	}

	gm := make(map[int]struct{}, defaultGroupSize)
	sm := make(map[int]struct{}, defaultShardSize)

	for _, ii := range *info {
		// gid是否重复
		if _, ok := gm[ii.GID]; ok {
			return false
		}
		gm[ii.GID] = struct{}{}

		// sid是否重复
		for _, sid := range ii.Shards.ShardIDs {
			if _, ok := sm[sid]; ok {
				return false
			}
			sm[sid] = struct{}{}
		}
	}

	return true
}

// NewShardInfo 新建Shard信息结构
func NewShardInfo() *ShardInfo {
	return &ShardInfo{
		ShardBaseInfo: new(ShardBaseInfo),
	}
}

// NewShardInfoFromJson
// 从json字符串新建Shard信息结构，可以用于Unmarshal
// 会进行有效性检查，并解析对应数据
func NewShardInfoFromJson(v []byte) (*ShardInfo, error) {
	si := NewShardInfo()
	err := si.SetJsonValue(v)
	if err != nil {
		return nil, err
	}
	return si, nil
}

// SetJsonValue 使用json内容赋值
// 会进行有效性检查，并解析对应数据
func (si *ShardInfo) SetJsonValue(v []byte) error {
	err := json.Unmarshal(v, si)
	if err != nil {
		return err
	}
	if !si.isValid() {
		return ErrInvalidShardInfo
	}
	return nil
}

func (si *ShardInfo) isValid() bool {
	if si.SID == 0 {
		return false
	}
	if err := si.s2i(); err != nil {
		return false
	}
	// 需要包括自身
	for _, sid := range si.ShardIDs {
		if si.SID == sid {
			return true
		}
	}
	return false
}

// s2i 所有服务列表字符串形式转int slice
func (si *ShardInfo) s2i() error {
	_, err := si.Shards.S2I()
	return err
}

// ToJson 返回shard info的json格式
func (si ShardInfo) ToJson() []byte {
	raw, _ := json.Marshal(si)
	return raw
}

func (si ShardInfo) String() string {
	return string(si.ToJson())
}

// ShallowClone 返回一个复制的ShardInfo
// 其中ShardBaseInfo指针指向同一个值
func (si ShardInfo) ShallowClone() *ShardInfo {
	return &ShardInfo{
		// GID:           si.GID,
		SID:           si.SID,
		ShardBaseInfo: si.ShardBaseInfo,
	}
}

// DeepClone 返回一个复制的ShardInfo
// ShardBaseInfo为新值
func (si ShardInfo) DeepClone() *ShardInfo {
	nsi := &ShardInfo{
		// GID: si.GID,
		SID: si.SID,
		ShardBaseInfo: &ShardBaseInfo{
			Status: si.Status,
		},
	}

	nsi.ShardsStr = si.ShardsStr
	nsi.ShardIDs = make([]int, len(si.ShardIDs))
	copy(nsi.ShardIDs, si.ShardIDs)

	return nsi
}

// genMD5 将任意类型结构转为json，并生成对应md5值
func genMD5(i interface{}) string {
	if i == nil {
		return ""
	}

	raw, err := json.Marshal(i)
	if err != nil {
		return ""
	}

	h := md5.New()
	h.Write(raw)

	return hex.EncodeToString(h.Sum(nil))
}
