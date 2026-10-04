package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/nghichtu91/platform/share/planx/arrayhelper"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	ParseTag = "etcd3"
)

var (
	EmptyKvErr        = errors.New("etcd Bind, Get kvs is empty")
	ErrTypeNotSupport = errors.New("only support struct or struct ptr")

	ErrInvalidValueInt    = errors.New("int: none or multiple value")
	ErrInvalidValueString = errors.New("string: none or multiple value")
	ErrInvalidValueStruct = errors.New("struct: none or multiple value")
)

/*
*
TODO 文档 + 注释 + 单元测试
Bind parses the data got from etcd and stores the result
in the value pointed to by v.

识别struct中的tag `etcd`, 比如 `etcd:"update"`

支持一下类型的序列化
int系列
string
struct, ptr 会将etcd上的值以json格式反序列化到这些结构里面
map, 将 目录和符合相应结构的值反序列化到这些机构里面， 以最后一格目录名为key

举例说明
现有etcd目录结构如下
- /hotdata(热更, gm写入， 其他服务器监听)
  - /15001
  - /modulex
  - update
  - current
  - /gamex
  - update
  - current
  - /15002
  - /其他需要热更的服务

解析代码如下

	type HotDataDetailInfo struct {
		Current int                       `etcd:"current"`
		Update  gamedata.HotUpdateVersion `etcd:"update" json:"update"`
	}

Infos := make(map[string]map[string]HotDataDetailInfo)
err := etcd.Bind("/hotdata", Infos)

bind接口的实现基于etcd3
bind函数只与etcd服务器发生一次连接
*/
func Bind(etcdPath string, v interface{}) error {
	_, err := BindWithRev(etcdPath, v)
	return err
}

func BindWithRev(etcdPath string, v interface{}) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time_out)
	defer cancel()
	resp, err := GetEtcd().Get(ctx, etcdPath,
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return 0, err
	}
	revision := resp.Header.GetRevision()
	if len(resp.Kvs) <= 0 {
		return revision, EmptyKvErr
	}
	return revision, BindKvs(etcdPath, v, resp.Kvs)
}

func BindKvs(etcdPath string, v interface{}, kvs []*mvccpb.KeyValue) error {
	kvs = filterKeys(etcdPath, kvs)
	err := parse(etcdPath, kvs, reflect.ValueOf(v), false)
	if err != nil {
		return err
	}
	return nil
}

func filterKeys(etcdPath string, kvs []*mvccpb.KeyValue) []*mvccpb.KeyValue {
	ret := make([]*mvccpb.KeyValue, 0)
	for _, v := range kvs {
		str := string(v.Key)
		str = str[len(etcdPath):]
		if str == "" || str[:1] == "/" {
			ret = append(ret, v)
		}
	}
	return ret
}

func parse(etcdPath string, kvs []*mvccpb.KeyValue, v reflect.Value, isJson bool) error {
	etcdPath = strings.TrimRight(etcdPath, "/")
	switch v.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint,
		reflect.Uint32, reflect.Uint64, reflect.Int16, reflect.Uint16, reflect.Int8, reflect.Uint8:
		if len(kvs) != 1 {
			return ErrInvalidValueInt
		}
		intV, err := strconv.Atoi(string(kvs[0].Value))
		if err != nil {
			return err
		}
		v.SetInt(int64(intV))
	case reflect.String:
		if len(kvs) != 1 {
			return ErrInvalidValueString
		}
		v.SetString(string(kvs[0].Value))
	case reflect.Struct:
		if isJson {
			if len(kvs) != 1 {
				return ErrInvalidValueStruct
			}
			res := reflect.New(v.Type()).Interface()
			err := json.Unmarshal(kvs[0].Value, &res)
			if err != nil {
				return err
			}
			v.Set(reflect.ValueOf(res).Elem())
		} else {
			t := v.Type()
			for i := 0; i < t.NumField(); i++ {
				etcdTag := t.Field(i).Tag.Get("etcd3")
				if etcdTag != "" {
					tags := strings.Split(etcdTag, ",")
					prefix := etcdPath + "/" + tags[0]
					cNodes := findKeysFromNodes(kvs, prefix)
					if len(cNodes) > 0 {
						err := parse(prefix, cNodes, v.Field(i), arrayhelper.ContainsString(tags, "json"))
						if err != nil {
							tilogs.L().Errorf("parse error %s, %v", prefix, err)
						}
					}
				}
			}
		}
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if isJson {
			if len(kvs) != 1 {
				return ErrInvalidValueStruct
			}
			res := v.Interface()
			err := json.Unmarshal(kvs[0].Value, &res)
			if err != nil {
				return err
			}
			v.Set(reflect.ValueOf(res).Elem())
		} else {
			t := v.Elem()
			if t.Kind() == reflect.Struct {
				for i := 0; i < t.NumField(); i++ {
					etcdTag := t.Type().Field(i).Tag.Get("etcd3")
					if etcdTag != "" {
						tags := strings.Split(etcdTag, ",")
						prefix := etcdPath + "/" + tags[0]
						cNodes := findKeysFromNodes(kvs, prefix)
						if len(cNodes) > 0 {
							err := parse(prefix, cNodes, t.Field(i), arrayhelper.ContainsString(tags, "json"))
							if err != nil {
								tilogs.L().Errorf("parse error %s, %v", prefix, err)
							}
						}
					}
				}
			} else {
				err := parse(etcdPath, kvs, t, false)
				if err != nil {
					tilogs.L().Errorf("parse pointer %v", err)
				}
			}

		}
	case reflect.Map:
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
		kvvs := classifyKvs(etcdPath, kvs)
		for k, vvs := range kvvs {
			mKey := reflect.ValueOf(splitKey(k))
			elemType := v.Type().Elem()
			mValue := reflect.New(elemType).Elem()
			parse(k, vvs, mValue, false)
			v.SetMapIndex(mKey, mValue)
		}
	}
	return nil
}

func findKeysFromNodes(kvs []*mvccpb.KeyValue, key string) []*mvccpb.KeyValue {
	ret := make([]*mvccpb.KeyValue, 0)
	for _, node := range kvs {
		nodeKey := string(node.Key)
		if strings.HasPrefix(nodeKey, key) {
			str := nodeKey[len(key):]
			if str == "" || str[:1] == "/" {
				ret = append(ret, node)
			}
		}
	}
	return ret
}

func splitKey(key string) string {
	str := strings.Split(key, "/")
	return str[len(str)-1]
}

// 向后延迟一格分类
func classifyKvs(etcdPath string, kvs []*mvccpb.KeyValue) map[string][]*mvccpb.KeyValue {
	res := make(map[string][]*mvccpb.KeyValue, 4)
	n := len(strings.Split(etcdPath, "/"))
	for i := range kvs {
		k := string(kvs[i].Key)
		ss := strings.Split(k, "/")
		if len(ss)-n > 1 {
			dk := strings.Join(ss[:n+1], "/")
			sl, ok := res[dk]
			if !ok {
				sl = make([]*mvccpb.KeyValue, 0, 16)
				res[dk] = sl
			}
			sl = append(sl, kvs[i])
			res[dk] = sl
		}
		if len(ss)-n == 1 {
			res[ss[len(ss)-1]] = append(res[ss[len(ss)-1]], kvs[i])
		}
	}
	return res
}

/*
*
文档+注释 // TODO 能否减少set的次数
sync 将v内容写到etcd上

识别 `etcd` tag

支持以下类型 string, int系列

举例

	type GMConfig struct {
		GmDbDriver   string `etcd:"GmDbDriver"`
		GmDbUser     string `etcd:"GmDbUser"`
		GmDbPwd      string `etcd:"GmDbPwd"`
		GmDbDatabase string `etcd:"GmDbDatabase"`
		GmDbUrl      string `etcd:"GmDbUrl"`
		Url          string `etcd:"Url"`
	}

gmCfg := &GMConfig{}
gmCfg.Url = ":8808"
gmCfg.GmDbDriver = "mysql"
gmCfg.GmDbUser = "root"
gmCfg.GmDbPwd = "mypwd"
gmCfg.GmDbDatabase = "gmtool"
gmCfg.GmDbUrl = "tcp(10.0.1.106:3306)"

err = etcd3.Sync("/a4k/150/gmtools", gmCfg)

sync 基于etcd3实现
sync 每个值的写入都会调用一次set，效率不高
*/
func Sync(etcdPath string, v interface{}) error {
	// 获取每个字段，转成字符串
	valueFiled := reflect.ValueOf(v).Elem()
	typeFiled := reflect.TypeOf(v).Elem()
	for i := 0; i < valueFiled.NumField(); i++ {
		v := valueFiled.Field(i)
		etcdValue := ""
		switch v.Type().Kind() {
		case reflect.String:
			etcdValue = v.String()
		case reflect.Int, reflect.Int32, reflect.Int64, reflect.Int8, reflect.Int16:
			etcdValue = fmt.Sprint(v.Int())
		case reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uint16, reflect.Uint8:
			etcdValue = fmt.Sprint(v.Uint())
		}

		tag := typeFiled.Field(i).Tag.Get("etcd3")
		if tag != "" {
			key := fmt.Sprintf("%s/%s", etcdPath, tag)
			err := Put(key, etcdValue)
			if err != nil {
				tilogs.L().Errorf("sync to etcd err %v", err)
			} else {
				tilogs.L().Debugf("sync to etcd %s=%s", key, etcdValue)
			}
		}
	}
	return nil
}

// SyncObj 通用的写入etcd方法
// 由于需要struct tag，仅接收struct作为参数
// 如果有"etcd3" tag，使用tag作为目录名，如果tag为"-"，忽略此结构
// 如果没有tag，使用变量名作为目录名
// 如果结构里包括其他struct，且存在json tag，将其解析成json字符串并写入etcd
// 其他struct如果没有json tag则递归写入
// TODO: etcd tag支持options
func SyncObj(etcdPath string, v interface{}) error {
	vt := reflect.TypeOf(v)
	vv := reflect.ValueOf(v)

	// 替换指针为对应的Value
	if vt.Kind() == reflect.Ptr {
		vv = vv.Elem()
		vt = reflect.TypeOf(vv.Interface())
	}

	switch vt.Kind() {
	case reflect.Struct:
	default:
		return ErrTypeNotSupport
	}

	// 获取每个字段，转成字符串
	for i := 0; i < vv.NumField(); i++ {
		f := vv.Field(i)

		// 跳过exported
		if !f.CanInterface() {
			continue
		}

		var etcdKey string
		// 使用变量名作为目录，如果有tag则替换为tag
		// 如果类型为struct且有json tag，保存为json
		etcdKey = vt.Field(i).Name
		tag, ok := vt.Field(i).Tag.Lookup(ParseTag)
		if ok {
			if tag == "-" {
				continue
			}
			etcdKey = tag
		}
		_, hasJsonTag := vt.Field(i).Tag.Lookup("json")

		path := fmt.Sprintf("%s/%s", etcdPath, etcdKey)

		switch f.Type().Kind() {
		case reflect.Ptr, reflect.Struct:
			vi := reflect.Indirect(f).Interface()
			if hasJsonTag {
				etcdValue, err := json.Marshal(vi)
				if err != nil {
					return err
				}
				err = Put(path, string(etcdValue))
				if err != nil {
					return err
				}
			} else {
				err := SyncObj(path, vi)
				if err != nil {
					return err
				}
			}
			continue
		case reflect.Map, reflect.Array, reflect.Slice, reflect.Func, reflect.Chan: // 暂时还不支持这些
			continue
		}

		err := Put(path, fmt.Sprintf("%v", f.Interface()))
		if err != nil {
			tilogs.L().Errorf("sync to etcd err %v", err)
			return err
		}
	}

	return nil
}
