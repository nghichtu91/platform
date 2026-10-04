package mergeinfo

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/util/collection"
)

const (
	EndpointsEnv  = "ETCDCTL_ENDPOINTS"
	DevOpsRootEnv = "OPS_ROOT"
	ServerRootEnv = "SRV_ROOT"

	DefaultEndpoint   = "http://localhost:2379"
	DefaultDevOpsRoot = "root/devops"
	DefaultServerRoot = "root/server"
)

// GOOS=linux go test -c -o show_test
// ./show_test -test.v -test.run ^TestShowManagerInfo$
func TestShowManagerInfo(t *testing.T) {
	// 使用环境变量 EndpointsEnv 初始化 endpoints 默认值 DefaultEndpoint
	endpoints := DefaultEndpoint
	if envEndpoints := os.Getenv(EndpointsEnv); envEndpoints != "" {
		endpoints = envEndpoints
	}

	// init serverRoot by env ServerRootEnv or DefaultServerRoot
	serverRoot := DefaultServerRoot
	if envServerRoot := os.Getenv(ServerRootEnv); envServerRoot != "" {
		serverRoot = envServerRoot
	}

	// init devOpsRoot by env DevOpsRootEnv or DefaultDevOpsRoot
	devOpsRoot := DefaultDevOpsRoot
	if envDevOpsRoot := os.Getenv(DevOpsRootEnv); envDevOpsRoot != "" {
		devOpsRoot = envDevOpsRoot
	}

	// Initialize etcd.InitEtcd using the endpoints shardInfo and require no error
	if err := etcd.InitEtcd([]string{endpoints}); err != nil {
		t.Fatal(err)
	}

	// create *EtcdMgr by devOpsRoot and serverRoot
	m := NewEtcdMgr(devOpsRoot, serverRoot)

	// check m.Init() no error
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}

	// 强制转换 m.mainShards 到 collection.IntSet
	mainShards := collection.IntSet(m.mainShards)
	// 强制转换 m.subShards 到 collection.IntSet
	subShards := collection.IntSet(m.subShards)

	// 依次迭代 m.shardInfo 的 key
	//		如果 mainShards 包含 sid  中, 就需要保证 shardInfo.SID == key
	//		如果 mainShards 不包含 sid 中, 就需要保证 sid 在 subShards 中
	for sid, shardInfo := range m.shardInfo {
		if mainShards.Contain(sid) {
			if shardInfo.SID != sid {
				t.Fatalf("mainShard %d not match %d", sid, shardInfo.SID)
			}
			// 迭代 shardInfo.ShardIDs
			for _, subShardID := range shardInfo.ShardIDs {
				// 如果 subShardID == sid, 就跳过
				if subShardID == sid {
					continue
				}
				// 检查 subShardID 在 subShards 中
				if !subShards.Contain(subShardID) {
					t.Fatalf("subShard %d not found in set", subShardID)
				}
				// 从 m.shardInfo 中取出 subShardID 对应的 ShardInfo, 并保证不为空
				subShardInfo := m.shardInfo[subShardID]
				if subShardInfo == nil {
					t.Fatalf("subShard %d not found in shard", subShardID)
				}
				// 检查 subShardInfo.SID == sid
				if subShardInfo.SID != sid {
					t.Fatalf("subShard %d not match %d", subShardID, sid)
				}
				// 使用 require.Equal 检查 subShardInfo.ShardIDs == shardInfo.ShardIDs
				require.Equal(t, subShardInfo.ShardIDs, shardInfo.ShardIDs)
			}
		} else {
			if !subShards.Contain(sid) {
				t.Fatalf("subShard %d not found", sid)
			}
		}
	}

	// make *ShardInfo slice from m.shardInfo shardInfo
	shards := make([]*ShardInfo, 0, len(m.shardInfo))
	for _, shard := range m.shardInfo {
		shards = append(shards, shard)
	}
	// sort shards by SID
	sort.Slice(shards, func(i, j int) bool {
		a, b := shards[i], shards[j]
		return a.SID < b.SID
	})

	// 打印 shards 中的每一个 shard
	for _, shard := range shards {
		t.Logf("%+v", shard)
	}

}
