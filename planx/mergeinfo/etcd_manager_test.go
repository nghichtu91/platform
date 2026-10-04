package mergeinfo

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nghichtu91/platform/share/planx/etcd"
)

func TestEtcdManagerV1(t *testing.T) {
	// var err error
	m := NewEtcdMgr("root/devops/test_emv1", "root/server/test_emv1")
	assert.NotNil(t, m)
	assert.NotNil(t, m.GetVerInfo())

	// 没有键值的时候启动
	assert.NotNil(t, m.Init())

	// 只有G2S
	assert.Nil(t, etcd.Put(m.pathG2S, `[{"gid":12, "sss":"1,2,3-5"},{"gid":32, "sss":"64,65,66-68"}]`))
	assert.Nil(t, m.Init())

	// 非法ShardInfo
	assert.Nil(t, etcd.Put(m.pathShards+"/64", `"sid": 64, "sss":"64", "status":1`))
	time.Sleep(1 * time.Second)
	assert.Nil(t, m.GetShardInfoBySID(64))

	// 合法
	assert.Nil(t, etcd.Put(m.pathShards+"/64", `{"sid": 64, "sss":"64", "status":1}`))
	time.Sleep(1 * time.Second)
	assert.NotNil(t, m.GetShardInfoBySID(64))
	r, _ := json.Marshal(m.GetShardInfoBySID(64))
	assert.Equal(t, `{"sid":64,"sss":"64","status":1}`, string(r))

	// 修改
	assert.Nil(t, etcd.Put(m.pathShards+"/64", `{"sid": 64, "sss":"64", "status":2}`))
	time.Sleep(1 * time.Second)
	assert.NotNil(t, m.GetShardInfoBySID(64))
	r, _ = json.Marshal(m.GetShardInfoBySID(64))
	assert.Equal(t, `{"sid":64,"sss":"64","status":2}`, string(r))

	// 清理
	assert.Nil(t, etcd.DeleteRecursive("root/devops/test_emv1"))
	assert.Nil(t, etcd.DeleteRecursive("root/server/test_emv1"))
}
