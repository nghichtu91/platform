package client

import (
	"fmt"
	"strings"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// list a directory
func (clt *EtcdHRCHYClient) List(key string) ([]*Node, error) {
	key, _, err := clt.ensureKey(key)
	if err != nil {
		return nil, err
	}
	// directory start with /
	dir := key + "/"

	txn := clt.client.Txn(clt.ctx)
	// make sure the list key is a directory

	txn.If().Then(
		clientv3.OpGet(dir, clientv3.WithPrefix()),
	)

	txnResp, err := txn.Commit()
	if err != nil {
		return nil, err
	}
	if !txnResp.Succeeded {
		return nil, ErrorListKey
	} else {
		if len(txnResp.Responses) > 0 {
			rangeResp := txnResp.Responses[0].GetResponseRange()
			return clt.list(dir, rangeResp.Kvs)
		} else {
			// empty directory
			return []*Node{}, nil
		}
	}
}

// pick key/value under the dir
func (clt *EtcdHRCHYClient) list(dir string, kvs []*mvccpb.KeyValue) ([]*Node, error) {
	nodes := []*Node{}

	// 处理所有的一级值
	for _, kv := range kvs {
		name := strings.TrimPrefix(string(kv.Key), dir)
		if strings.Contains(name, "/") {
			// secondary directory
			continue
		}
		nodes = append(nodes, clt.createNode(kv))
	}

	// 处理所有的目录
	dirs := checkDirs(dir, kvs)
	fmt.Println("****", clt.rootKey)
	for _, kv := range dirs {
		fmt.Println("*****", dir, string(kv.Key))
		name := strings.TrimPrefix(string(kv.Key), dir)
		if strings.Contains(name, "/") {
			// secondary directory
			continue
		}
		nodes = append(nodes, clt.createDirNode(kv.KeyValue))
	}
	return nodes, nil
}

// root/server/150/shards/10/displayName
// root   root/server   root/server/150
func checkDirs(dir string, kvs []*mvccpb.KeyValue) (ret []*Node) {
	fmt.Println("=====kvs", kvs)
	dirsMap := make(map[string]*Node, 64)
	for _, node := range kvs {
		strK := ""
		keys := strings.Split(string(node.Key), "/")
		for i, k := range keys[:len(keys)-1] {
			if i == 0 {
				strK = k
			} else {
				strK = strK + "/" + k
			}

			if _, ok := dirsMap[strK]; !ok {
				dirsMap[strK] = &Node{
					KeyValue: &mvccpb.KeyValue{
						Key: []byte(strK),
					},
					IsDir: true,
				}
			}
		}
	}
	for _, node := range dirsMap {
		if !strings.HasPrefix(string(node.Key), dir) {
			continue
		}
		ret = append(ret, node)
	}
	fmt.Println("====== dirs", ret)
	return
}
