package mergeinfo

import (
	"encoding/json"
	"errors"
	"sort"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

var (
	ErrEtcdRootNil = errors.New("etcd root is nil")
)

type EtcdMgr struct {
	*ManagerV1

	// etcd相关路径
	etcdDevops string
	etcdServer string
	pathG2S    string
	pathShards string

	wg   *util.WaitGroupWrapper
	quit chan struct{}
}

// actHandler 执行对应etcd键值的函数
type actHandler func(k, v []byte) error

// NewEtcdMgr 新建基于etcd数据的Manager
func NewEtcdMgr(etcdDevOps, etcdSrv string) *EtcdMgr {
	return &EtcdMgr{
		ManagerV1:  NewManagerV1(),
		etcdDevops: etcdDevOps,
		etcdServer: etcdSrv,
		pathG2S:    etcdDevOps + "/" + etcd.GID2ShardID,
		pathShards: etcdSrv + "/" + etcd.MergedInfo,
		wg:         new(util.WaitGroupWrapper),
		quit:       make(chan struct{}),
	}
}

// Init 初始化对应数据
func (m *EtcdMgr) Init() error {
	if m.etcdDevops == "" || m.etcdServer == "" {
		return ErrEtcdRootNil
	}

	revision, err := m.parseAll()
	if err != nil {
		return err
	}

	m.watch(
		m.pathG2S,
		revision.G2SRev,
		false,
		m.updateG2S,
		nil,
	)

	m.watch(
		m.pathShards,
		revision.ShardsRev,
		true,
		m.updateShard,
		nil,
	)

	tilogs.L().Infof("etcd manager inited. devops root %s, server root %s, revsion %+v",
		m.etcdDevops, m.etcdServer, revision)

	return nil
}

// Stop 关闭Manager
func (m *EtcdMgr) Stop() {
	close(m.quit)
	tilogs.L().Infof("mergeinfo etcd manager stopped")
}

type MergeRevision struct {
	G2SRev    int64
	ShardsRev int64
}

// parseAll 使用目前etcd键值设置合服信息
func (m *EtcdMgr) parseAll() (info MergeRevision, err error) {
	// g2s
	g2sJson, g2sRev, err := etcd.GetWithRev(m.pathG2S)
	if err != nil {
		return
	}

	tilogs.L().Infof("merge_manager, parseAll g2sJson:%s, pathG2S:%s", g2sJson, m.pathG2S)

	g2s, err := NewSid2GidInfoFromJson([]byte(g2sJson))
	if err != nil {
		return
	}

	tilogs.L().Infof("merge_manager, parseAll g2s:%+v", g2s)

	// shards
	all, shardsRev, err := etcd.GetSubRecursiveWithRev(m.pathShards)
	if err != nil {
		return
	}
	shards := make([]*ShardInfo, 0, len(all))
	for _, shardJson := range all {
		// tilogs.L().Infof("merge_manager adding shardJson %s", shardJson)
		si, err := NewShardInfoFromJson([]byte(shardJson))
		if err != nil {
			// 暂时先跳过
			tilogs.L().Errorf("NewShardInfoFromJson err %s, shardJson %s", err.Error(), shardJson)
			continue
		}
		shards = append(shards, si)
	}

	tilogs.L().Infof("merge_manager, parseAll shards:%v", shards)

	sort.Slice(shards, func(i, j int) bool {
		a, b := shards[i], shards[j]
		return a.SID < b.SID
	})
	err = m.SetInfoV2(g2s, shards...)
	if err != nil {
		return
	}
	info.ShardsRev = shardsRev
	info.G2SRev = g2sRev
	return
}

// watch 监视对应键值，并绑定handler
func (m *EtcdMgr) watch(path string, rev int64, watchAllSub bool, put, del actHandler) {
	etcd.WatchWithRevNoPrevRetry(path, rev, watchAllSub, m.quit, m.wg, func(resp clientv3.WatchResponse) {
		for _, event := range resp.Events {
			if event.Kv == nil {
				continue
			}
			switch event.Type {
			case clientv3.EventTypePut:
				if put == nil {
					continue
				}
				if err := put(event.Kv.Key, event.Kv.Value); err != nil {
					tilogs.L().Errorf("failed while %s put, %v", path, err)
				}
				tilogs.L().Infof("merge etcd watch path:%s, kv:%v", path, event.Kv)
			case clientv3.EventTypeDelete:
				if del == nil {
					continue
				}
				if err := del(event.Kv.Key, event.Kv.Value); err != nil {
					tilogs.L().Errorf("failed while %s del, %v", path, err)
				}
			}
		}
	})
}

func (m *EtcdMgr) updateG2S(k, v []byte) error {
	g2i := NewSid2Gid()
	err := json.Unmarshal(v, g2i)
	if err != nil {
		return err
	}

	if !g2i.isValid() {
		return ErrInvalidSid2GidInfo
	}

	tilogs.L().Infof("updateG2S g2i：%v", g2i)

	return m.UpdateSid2Gid(*g2i)
}

func (m *EtcdMgr) updateShard(k, v []byte) error {
	si := NewShardInfo()
	err := json.Unmarshal(v, si)
	if err != nil {
		return err
	}

	if !si.isValid() {
		return ErrInvalidShardInfo
	}

	tilogs.L().Infof("updateG2S si：%v", si)

	return m.UpdateShard(si)
}

// Reset 重置所有合服信息为当前etcd状态
func (m *EtcdMgr) Reset() error {
	_, err := m.parseAll()
	return err
}
