package mergeinfo

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func init() {
	// 测试的时候不跳过
	skipOldMergeCheck = false
}

func TestManagerV1(t *testing.T) {
	// 准备数据
	s2gRaw := `[{"gid":16, "sss":"160001,160002,160099-160999"},{"gid":17, "sss":"170001,170002,170099-170999"},{"gid":99, "sss":"99999"}]`
	s2g, err := NewSid2GidInfoFromJson([]byte(s2gRaw))
	assert.Nil(t, err)

	sidInfoRaw := `{"sid": 160001, "sss":"160001,160099-160999", "status":1}`
	si, err := NewShardInfoFromJson([]byte(sidInfoRaw))
	assert.Nil(t, err)

	m := NewManagerV1()

	t.Run("Init", func(t *testing.T) {
		assert.Nil(t, m.SetInfo(s2g, si))
		assert.Nil(t, m.SetInfo(s2g, si)) // 重复init

		info := m.GetVerInfo()
		t.Logf("info %v", string(info))
	})

	t.Run("Init Special", func(t *testing.T) {
		assert.Equal(t, ErrInvalidSid2GidInfo, m.SetInfo(nil, nil))
		assert.Equal(t, nil, m.SetInfo(s2g, nil))
		assert.Equal(t, nil, m.SetInfo(s2g, si, nil, nil))

		// 主ID重复
		si2, err := NewShardInfoFromJson([]byte(`{"sid": 160001, "sss":"160001,160099-160999", "status":1}`))
		assert.Nil(t, err)
		assert.Equal(t, ErrDuplicateMainSID, m.SetInfo(s2g, si, si2))

		// 子ID重复
		si3, err := NewShardInfoFromJson([]byte(`{"sid": 160002, "sss":"160002,160099-160999", "status":1}`))
		assert.Nil(t, err)
		assert.Equal(t, ErrDuplicateSubSID, m.SetInfo(s2g, si, si3))

		// 还是重复
		si4, err := NewShardInfoFromJson([]byte(`{"sid": 160099, "sss":"160099-160999", "status":1}`))
		assert.Nil(t, err)
		assert.Equal(t, ErrDuplicateMainSID, m.SetInfo(s2g, si, si4))

		// 越界
		si5, err := NewShardInfoFromJson([]byte(`{"sid": 160003, "sss":"160003,180099-180999", "status":1}`))
		assert.Nil(t, err)
		assert.Equal(t, ErrShardIDHasNoGID, m.SetInfo(s2g, si, si5))
	})

	t.Run("GetGID", func(t *testing.T) {
		assert.Nil(t, m.SetInfo(s2g, si))

		tests := []struct {
			sid int
			gid int
		}{
			{160001, 16},
			{160099, 16},
			{160100, 16},
			{170001, 17}, // 分配了但未启动
			{999, -1},    // 没有
		}

		for i, test := range tests {
			failMsg := "test " + strconv.Itoa(i) + " failed"
			assert.Equal(t, test.gid, m.GetGIDBySID(test.sid), failMsg)
		}
	})

	t.Run("GetSID", func(t *testing.T) {
		assert.Nil(t, m.SetInfo(s2g, si))

		tests := []struct {
			sid  int
			rSid int
		}{
			{160001, 160001},
			{160099, 160001},
			{160100, 160001},
			{170001, 170001}, // 分配了但未启动
			{999, -1},        // 没有
		}

		for i, test := range tests {
			failMsg := "test " + strconv.Itoa(i) + " failed"
			assert.Equal(t, test.rSid, m.GetRealSIDBySID(test.sid), failMsg)
		}
	})

	t.Run("UpdateSid2Gid", func(t *testing.T) {
		assert.Nil(t, m.UpdateSid2Gid(s2g))

		// 正常增
		g1 := append(s2g, &Sid2Gid{GID: 111, Shards: Shards{
			ShardsStr: "11101,11103,11105",
			ShardIDs:  []int{11101, 11103, 11105},
		}})
		assert.True(t, g1.isValid())
		assert.Nil(t, m.UpdateSid2Gid(g1))
		assert.Equal(t, 1810, len(m.sid2gid))

		// 正常改
		g1[len(g1)-1].GID = 121
		g1[len(g1)-1].ShardsStr = "12101,12103"
		g1[len(g1)-1].ShardIDs = []int{12101, 12103}
		assert.True(t, g1.isValid())
		assert.Nil(t, m.UpdateSid2Gid(g1))
		assert.Equal(t, 1809, len(m.sid2gid))

		// 正常删
		g3 := g1[:len(g1)-1]
		assert.True(t, g3.isValid())
		assert.Nil(t, m.UpdateSid2Gid(g3))
		assert.Equal(t, 1807, len(m.sid2gid))
	})

	t.Run("UpdateShard", func(t *testing.T) {
		sid := si.SID
		nsi := si.DeepClone()

		// 关多人 -> 离线
		nsi.Status = SrvStatusOffline
		assert.Nil(t, m.UpdateShard(nsi))
		// 2023年04月06日 离线状态保留合服信息
		assert.NotNil(t, m.GetShardInfoBySID(sid))

		// 离线 -> 关多人
		nsi = si.DeepClone()
		nsi.Status = SrvStatusMultiOff
		assert.Nil(t, m.UpdateShard(nsi))
		assert.NotNil(t, m.GetShardInfoBySID(sid))
		assert.Equal(t, SrvStatusMultiOff, m.GetShardInfoBySID(sid).Status)

		// 关多人 -> 开多人
		nsi = si.DeepClone()
		nsi.Status = SrvStatusMultiOn
		assert.Nil(t, m.UpdateShard(nsi))
		assert.NotNil(t, m.GetShardInfoBySID(sid))
		assert.Equal(t, SrvStatusMultiOn, m.GetShardInfoBySID(sid).Status)

		// 开多人 -> 开多人 报错
		nsi = si.DeepClone()
		nsi.Status = SrvStatusMultiOn
		assert.Equal(t, ErrTargetShardIsOnline, m.UpdateShard(nsi))
		assert.NotNil(t, m.GetShardInfoBySID(sid))
		assert.Equal(t, SrvStatusMultiOn, m.GetShardInfoBySID(sid).Status)

		// 开多人 -> 离线
		nsi = si.DeepClone()
		nsi.Status = SrvStatusOffline
		assert.Nil(t, m.UpdateShard(nsi))
		// 2023年04月06日 离线状态保留合服信息
		assert.NotNil(t, m.GetShardInfoBySID(sid))
	})
}

func TestManagerV1_SetInfoV2(t *testing.T) {
	// 准备数据
	s2gRaw := `[{"gid":1, "sss":"1-50"},{"gid":16, "sss":"160001-160050"},{"gid":17, "sss":"170001,170002,170010-170050"},{"gid":99, "sss":"99999"}]`
	s2g, err := NewSid2GidInfoFromJson([]byte(s2gRaw))
	require.Nil(t, err)

	si, err := NewShardInfoFromJson([]byte(
		`{"sid": 160001, "sss":"160001-160010", "status":1}`,
	))
	require.Nil(t, err)

	m := NewManagerV1()
	m2 := NewManagerV1()

	t.Run("Init", func(t *testing.T) {
		assert.Nil(t, m.SetInfo(s2g, si))
		assert.Nil(t, m.SetInfo(s2g, si))    // 重复init		assert.Nil(t, m.SetInfoV2(s2g, si))
		assert.Nil(t, m2.SetInfoV2(s2g, si)) // 重复init		assert.Nil(t, m.SetInfoV2(s2g, si))
		assert.Nil(t, m2.SetInfoV2(s2g, si)) // 重复init

		t.Logf("info %v, info %v", string(m.GetVerInfo()), string(m2.GetVerInfo()))
	})

	gen := func(str string) *ShardInfo {
		info, _ := NewShardInfoFromJson([]byte(str))
		return info
	}

	t.Run("Set", func(t *testing.T) {

		// 模拟乱序的情况
		allShards := []*ShardInfo{
			gen(`{"sid": 160047, "sss":"160047-160048", "status":0}`),
			gen(`{"sid": 160007, "sss":"160007", "status":0}`),
			gen(`{"sid": 160004, "sss":"160004-160008", "status":0}`),
			gen(`{"sid": 160001, "sss":"160001-160008", "status":1}`),
			gen(`{"sid": 160005, "sss":"160005", "status":0}`),
			gen(`{"sid": 160006, "sss":"160006-160008", "status":0}`),
			gen(`{"sid": 160040, "sss":"160040-160044", "status":1}`),
			gen(`{"sid": 160043, "sss":"160043-160044", "status":0}`),
			gen(`{"sid": 160045, "sss":"160045-160048", "status":1}`),
		}

		require.Nil(t, m.SetInfo(s2g, allShards...))
		require.Nil(t, m2.SetInfoV2(s2g, allShards...))
		require.Equal(t, m, m2)
	})

	pack := func(shards ...*ShardInfo) []*ShardInfo {
		return shards
	}

	t.Run("SetErr", func(t *testing.T) {
		type checkCase struct {
			name      string
			allShards []*ShardInfo
			wantErr   bool
		}

		for _, cc := range []checkCase{
			{
				name: "err1:子服大于主服",
				allShards: pack(
					gen(`{"sid": 160047, "sss":"160046-160047", "status":0}`),
				),
				wantErr: true,
			},
			{
				name: "err2:子服大于主服2",
				allShards: pack(
					gen(`{"sid": 5, "sss":"1-6", "status":0}`),
				),
				wantErr: true,
			},
			{
				name: "err2:合服不完全",
				allShards: pack(
					gen(`{"sid": 3, "sss":"3-4", "status":0}`),
					gen(`{"sid": 1, "sss":"1,2,4", "status":0}`),
				),
				wantErr: true,
			},
			{
				name: "err2:主服ID异常",
				allShards: pack(
					gen(`{"sid": 5, "sss":"1-6", "status":0}`),
					gen(`{"sid": 1, "sss":"1-4", "status":0}`),
				),
				wantErr: true,
			},
			{
				name: "err2:区间重复",
				allShards: pack(
					gen(`{"sid": 1, "sss":"1-4", "status":0}`),
					gen(`{"sid": 3, "sss":"3-6", "status":0}`),
				),
				wantErr: true,
			},
			{
				name: "err2:多次出现",
				allShards: pack(
					gen(`{"sid": 1, "sss":"1-2", "status":0}`),
					gen(`{"sid": 3, "sss":"3-4", "status":0}`),
					gen(`{"sid": 2, "sss":"2-3", "status":0}`),
				),
				wantErr: true,
			},
		} {
			setErr := m2.SetInfoV2(s2g, cc.allShards...)
			t.Logf("set error %v", setErr)
			require.Equal(t, setErr != nil, cc.wantErr, cc.name)
		}
	})

}

func BenchmarkManagerV1(b *testing.B) {
	m := NewManagerV1()
	s2gRaw := `[
		{"gid":1, "sss":"1-9999"},
		{"gid":2, "sss":"10000-19999"},
		{"gid":3, "sss":"20000-29999"},
		{"gid":4, "sss":"30000-39999"},
		{"gid":5, "sss":"40000-49999"},
		{"gid":6, "sss":"50000-59999"},
		{"gid":7, "sss":"60000-69999"},
		{"gid":8, "sss":"70000-79999"},
		{"gid":9, "sss":"80000-89999"},
		{"gid":10, "sss":"90000-99999"}]`
	s2g, err := NewSid2GidInfoFromJson([]byte(s2gRaw))
	if err != nil {
		b.Fail()
	}
	_ = m.SetInfo(s2g)

	size := 99999

	si := make([]*ShardInfo, 0, size)
	for i := 0; i < size; i++ {
		sid := rand.Intn(size) + 1
		info, err := NewShardInfoFromJson([]byte(fmt.Sprintf("{\"sid\": %d, \"sss\":\"%d\", \"status\":%d}", sid, sid, i%2)))
		if err != nil {
			b.Fail()
		}
		si = append(si, info)
	}

	b.Run("Set", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err = m.UpdateShard(si[i%size])
		}
	})

	for i := 0; i < size; i++ {
		err = m.UpdateShard(si[i])
	}

	b.Logf("%d shards remaining", len(m.shardInfo))

	b.Run("Get", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			m.GetShardInfoBySID(si[i%size].SID)
		}
	})
}
