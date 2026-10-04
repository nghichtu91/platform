package etcd

import (
	"context"
	"fmt"
	"strings"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	// /Root/gid/ser_ver
	KeyPathVersion   = "ser_ver"
	KeySerVer        = "version"
	KeySerDataVer    = "data"
	KeySerHotDataVer = "hotdata"
)

type serVer struct {
	paths []string
}

// 给gm工具用，获取大区gid下所有服务版本信息
// typ->id->key:value
func GetServersVer(etcd_root, gid string) (map[string]map[string]map[string]string, error) {
	path := fmt.Sprintf("%s/%s/%s", etcd_root, gid, KeyPathVersion)
	resp, err := GetEtcd().Get(context.Background(), path, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	res := make(map[string]map[string]map[string]string, 8)
	for _, _v := range resp.Kvs {
		k := string(_v.Key)
		v := string(_v.Value)
		ss := strings.Split(k, "/")
		if len(ss) < 7 || v == "" {
			continue
		}
		//gid := ss[2]
		typ := ss[4]
		id := ss[5]
		key := ss[6]
		ids, ok := res[typ]
		if !ok {
			ids = make(map[string]map[string]string, 4)
			res[typ] = ids
		}
		ks, ok := ids[id]
		if !ok {
			ks = make(map[string]string, 2)
			ids[id] = ks
		}
		ks[key] = v
	}
	return res, nil
}

func NewRegServer(etcd_root string, gids []string, typ ServiceTypeId, Id string) *serVer {
	s_typ, ok := SerTyp2Name[typ]
	if !ok {
		panic(fmt.Errorf("NewRegServer typ not found  %v", typ))
	}
	_path := make([]string, 0, len(gids))
	for _, gid := range gids {
		_path = append(_path, fmt.Sprintf("%s/%s/%s/%s/%s",
			etcd_root, gid, KeyPathVersion, s_typ, Id))
	}
	return &serVer{
		paths: _path,
	}
}

// NewRegServers 生成一组ServerID对应的serVer，gamex使用此函数便于合服后处理相关内容
func NewRegServers(etcdRoot string, gids []string, typ ServiceTypeId, Ids ...uint) *serVer {
	sTyp, ok := SerTyp2Name[typ]
	if !ok {
		panic(fmt.Errorf("NewRegServers typ not found  %v", typ))
	}

	sv := &serVer{
		paths: make([]string, 0, len(gids)*len(Ids)),
	}

	for _, gid := range gids {
		for _, sid := range Ids {
			sv.paths = append(sv.paths, fmt.Sprintf("%s/%s/%s/%s/%d", etcdRoot, gid, KeyPathVersion, sTyp, sid))
		}
	}

	return sv
}

func (v *serVer) RegServerVer(ver string) error {
	for _, path := range v.paths {
		key := fmt.Sprintf("%s/%s", path, KeySerVer)
		if err := Put(key, ver); err != nil {
			return err
		}
	}
	return nil
}

func (v *serVer) RegServerData(dataVer string) error {
	for _, path := range v.paths {
		key := fmt.Sprintf("%s/%s", path, KeySerDataVer)
		if err := Put(key, dataVer); err != nil {
			return err
		}
	}
	return nil
}

func (v *serVer) RegServerHotData(dataVer string) error {
	for _, path := range v.paths {
		key := fmt.Sprintf("%s/%s", path, KeySerHotDataVer)
		if err := Put(key, dataVer); err != nil {
			return err
		}
	}
	return nil
}

func (v *serVer) UnRegServerVer() {
	defer tilogs.PanicCatcher("UnRegServerVer Panic")

	for _, path := range v.paths {
		key := fmt.Sprintf("%s/%s", path, KeySerVer)
		if err := Delete(key); err != nil {
			tilogs.L().Errorf("UnRegServerVer Delete %s failed, err %v", key, err)
			return
		}
		key = fmt.Sprintf("%s/%s", path, KeySerDataVer)
		if err := Delete(key); err != nil {
			tilogs.L().Errorf("UnRegServerVer Delete %s failed, err %v", key, err)
			return
		}
	}
}
