package redis_helper

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/gomodule/redigo/redis"
	"github.com/nghichtu91/platform/share/planx/redispool"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type ManualDBDump interface {
	// val是自身，返回序列化后的基础类型
	ToDB(val interface{}) interface{}
	// data是序列化后的基础类型，obj是自身的value
	FromDB(data interface{}, obj reflect.Value) error
}

// ///////////////
// Helper function for DumpToHashDB of Redis
func init() {
}

type XArgs struct {
	redis.Args
}

func (args XArgs) Add(value ...interface{}) XArgs {
	args.Args = args.Args.Add(value...)
	return args
}

func (args XArgs) AddFlat(v interface{}) XArgs {
	args.Args = args.Args.AddFlat(v)
	return args
}

// 语意上这个函数是只为了HMSET而存在的
func (args XArgs) AddHash(v interface{}, dirtyHash map[string]interface{}) (
	res XArgs, newDirtyHash map[string]interface{}, chged []string) {

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			tilogs.L().Warnf("AddHash Ptr rv %v is Nil!", rv)
			return args, nil, nil
		} else if !rv.IsValid() {
			tilogs.L().Warnf("AddHash Ptr rv %v is not Valid!", rv)
			return args, nil, nil
		} else {
			rv = rv.Elem()
		}
	}

	switch rv.Kind() {
	case reflect.Struct:
		// tilogs.L().Debugf("AddHash Struct!")
		args.Args, newDirtyHash, chged = flattenStruct(args.Args, rv, dirtyHash)
	case reflect.Map:
		// tilogs.L().Debugf("AddHash Map!")
		args.Args, newDirtyHash, chged = flattenMap(args.Args, rv, dirtyHash)
	default:
		args.Args = append(args.Args, v)
	}
	return args, newDirtyHash, chged
}

// 语意上这个函数是只为了HDEL map而存在的
func (args XArgs) DelHash(v interface{}, dirtyHash map[string]interface{}) (
	res XArgs, chged []string) {

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			tilogs.L().Warnf("DelHash Ptr rv %v is Nil!", rv)
			return args, nil
		} else if !rv.IsValid() {
			tilogs.L().Warnf("DelHash Ptr rv %v is not Valid!", rv)
			return args, nil
		} else {
			rv = rv.Elem()
		}
	}

	if rv.Kind() != reflect.Map {
		return args, nil
	}
	// tilogs.L().Debugf("DelHash Map!")
	args.Args, chged = flattenDelMap(args.Args, rv, dirtyHash)
	return args, chged
}

func needJson(v reflect.Value) bool {
	switch v.Interface().(type) {
	case string, bool, byte, uint16, uint32, uint64, uint, int8, int16, int32, int64, int, float32, float64, nil, []byte:
		return false
	default:
		return true
	}
}

func checkMapKeyType(v interface{}) bool {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Map {
		return true
	}
	switch rv.Type().Key().Kind() {
	case reflect.String, reflect.Bool,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func ToDB(val interface{}) interface{} {
	rval := reflect.ValueOf(val)
	if rval.CanAddr() {
		if structConvert, ok := rval.Addr().Interface().(ManualDBDump); ok {
			return structConvert.ToDB(val)
		}
	}

	if structConvert, ok := rval.Interface().(ManualDBDump); ok {
		return structConvert.ToDB(val)
	} else {
		return ToDB_default(val)
	}
}

func ToDB_default(val interface{}) interface{} {
	rval := reflect.ValueOf(val)
	is_need_json := needJson(rval)
	if is_need_json {
		jval, err := json.Marshal(val)
		if err != nil {
			panic(err)
		}
		return jval
	} else {
		return val
	}
}

func FromDB(src_data interface{}, dest reflect.Value) error {
	if dest.CanAddr() {
		if structConvert, ok := dest.Addr().Interface().(ManualDBDump); ok {
			return structConvert.FromDB(src_data, dest)
		}
	}

	if structConvert, ok := dest.Interface().(ManualDBDump); ok {
		return structConvert.FromDB(src_data, dest)
	} else {
		return FromDB_default(src_data, dest)
	}
}

func FromDB_default(src_data interface{}, dest reflect.Value) error {
	is_need_json := needJson(dest)
	if is_need_json {
		data, ok := src_data.([]byte)
		if !ok {
			return errors.New("fromDB data is not []byte!")
		}

		res := reflect.New(dest.Type()).Interface()

		err := json.Unmarshal(data, &res)
		if err != nil {
			return err
		}
		if res != nil {
			// return fmt.Errorf("fromDB failed unMarshal json from %s", data)
			dest.Set(reflect.ValueOf(res).Elem())
		}
		return nil
	} else {
		return errors.New("Can not json!")
	}
}

func GetInItProfileDBKey(key string) string {
	dbkey, err := db.ParseProfileDbKey(key)
	if err != nil {
		tilogs.L().Errorf("key %s format error", key)
		return ""
	}
	dbkey.Account.UserId = db.NewUserIDWithName("init")

	return fmt.Sprintf("init:%s", dbkey)
}

func GenDirtyHash(v interface{}) map[string]interface{} {
	_, ret, _ := XArgs{}.AddHash(v, nil)
	return ret
}

/**
将数据序列化到指定的CmdBuffer中
特别说明，struct中可以定义map类型的数据结构, json序列化map的时候，是有序的，所以相同的map每次序列化的结果一样
*/
func DumpToHashDBCmcBufferCheckDirty(conn redis.Conn, key string,
	v interface{}, dirtyHash map[string]interface{}) (error, map[string]interface{}, []string) {
	// 检查v是map类型的key，是否是基本类型
	if !checkMapKeyType(v) {
		tilogs.L().Errorf("Dump To CmdBuffer failed, v's key type isn't base type")
		return fmt.Errorf("Dump To CmdBuffer failed, v's key type isn't base type"), nil, nil
	}
	chged := make([]string, 0, 16)
	// HMSET
	argsSet, newDirtyHash, chgedSet := XArgs{}.Add(key).AddHash(v, dirtyHash)
	cmdsSet := argsSet.Args
	chged = append(chged, chgedSet...)
	// HDEL[map]
	argsDel, chgedDel := XArgs{}.Add(key).DelHash(v, dirtyHash)
	cmdsDel := argsDel.Args
	chged = append(chged, chgedDel...)
	if len(cmdsSet) <= 1 && len(cmdsDel) <= 1 {
		return nil, newDirtyHash, chged
	}
	if len(cmdsSet) > 1 {
		err := conn.Send("HMSET", cmdsSet...)
		if err != nil {
			tilogs.L().Errorf("Dump To CmdBuffer failed, key:%s. err:%s, cmds: %v", key, err.Error(), cmdsSet)
			return err, nil, nil
		}
	}
	if len(cmdsDel) > 1 {
		err := conn.Send("HDEL", cmdsDel...)
		if err != nil {
			tilogs.L().Errorf("Dump To CmdBuffer failed, key:%s. err:%s, cmds: %v", key, err.Error(), cmdsDel)
			return err, nil, nil
		}
	}
	tilogs.L().Debugf("Dump To %s CmdBuffer success.", key)
	return nil, newDirtyHash, chged
}

func DumpToHashDBCmcBuffer(conn redis.Conn, key string, v interface{}) error {
	args, _, _ := XArgs{}.Add(key).AddHash(v, nil)
	cmds := args.Args
	err := conn.Send("HMSET", cmds...)
	if err != nil {
		tilogs.L().Errorf("Dump To CmdBuffer failed, key:%s. err:%s, cmds: %v", key, err.Error(), cmds)
		return err
	}
	// tilogs.L().Debugf("Dump To %s CmdBuffer success.", key)
	return nil
}

// func DumpToHashDB(db redis.Conn, key string, v interface{}) error {
//	if db == nil {
//		return fmt.Errorf("DumpToHashDB db nil")
//	}
//	cmds := XArgs{}.Add(key).AddHash(v).Args
//	_, err := redis.String(db.Do("HMSET", cmds...))
//	if err != nil {
//		tilogs.L().Errorf("Dump To %s failed. err:%s, cmds: %v", key, err.Error(), cmds)
//		return err
//	}
//	tilogs.L().Debugf("Dump To %s success.", key)
//	return nil
// }

// RESTORE_ERR_Profile_No_Data 表示玩家第一次登陆游戏，没有存档，这不视为Bug
// 外面的逻辑需要根据此判断是否是第一次登陆游戏
var RESTORE_ERR_Profile_No_Data = errors.New("Profile_No_Data")

func RestoreFromHashDB(db redis.Conn, key string, v interface{}, isNeedInit, logInfo bool) error {
	if db == nil {
		return fmt.Errorf("RestoreFromHashDB db nil")
	}
	// T990 TASK 增加服务器拷贝存档功能
	// 初始化存档时如果是开发模式的话，会以“init:XXXX:0:0:1”中的数据作为初始数据
	// 如果“init:XXXX:0:0:1”没有的话就还是用默认空存档
	if isNeedInit {
		key = GetInItProfileDBKey(key)
	}

	reply, err := db.Do("HGETALL", key)
	if err != nil {
		tilogs.L().Warnf("Restore To %s failed. err:%s", key, err.Error())
		return err
	}
	return RestoreFromReply(key, reply, v, logInfo)
}

// 返回的map表示每个key是否有值不为空，不再使用error
func RestoreFromHashDBBatch(db redis.Conn, keys []string, vs []interface{}, logInfo bool) (error, map[string]bool) {
	if len(keys) != len(vs) {
		tilogs.L().Warnf("RestoreFromHashDBBatch key.length[%d] != v.length[%d]", len(keys), len(vs))
		return fmt.Errorf("RestoreFromHashDBBatch key.length[%d] != v.length[%d]", len(keys), len(vs)), nil
	}
	for _, k := range keys {
		err := db.Send("HGETALL", k)
		if err != nil {
			return err, nil
		}
	}

	results, err := redis.Values(db.Do(""))
	if err != nil {
		return err, nil
	}
	if len(results) != len(vs) {
		tilogs.L().Warnf("RestoreFromHashDBBatch results.length[%d] != v.length[%d]", len(results), len(vs))
		return fmt.Errorf("RestoreFromHashDBBatch results.length[%d] != v.length[%d]", len(results), len(vs)), nil
	}

	rt := make(map[string]bool, 64)
	for i, result := range results {
		key := keys[i]
		v := vs[i]
		err := RestoreFromReply(key, result, v, logInfo)
		if err != nil {
			if err == RESTORE_ERR_Profile_No_Data {
				rt[key] = false
				continue
			}
			return err, nil
		}
		rt[key] = true
	}
	return nil, rt
}

func RestoreFromReply(key string, reply interface{}, v interface{}, logInfo bool) error {
	values, err := redis.Values(reply, nil)
	if err != nil {
		tilogs.L().Warnf("Restore To %s failed. err:%s", key, err.Error())
		return err
	}

	if len(values) == 0 {
		// 初始化存档不在这里实现了
		// if (!is_need_init) && devMode {
		// 初始化存档时如果是开发模式的话，会以“init:XXXX:0:0:1”中的数据作为初始数据
		//	RestoreFromHashDB(db, key, v, true)
		//	return RESTORE_ERR_Profile_No_Data
		// } else {
		// 如果“init:XXXX:0:0:1”没有的话就还是用默认空存档
		return RESTORE_ERR_Profile_No_Data
		// }
	}

	tilogs.L().Debugf("Restore To %s", key)
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	} else {
		tilogs.L().Errorf("RestoreFromHashDB need a Ptr %s", key)
		return errors.New("RestoreFromHashDB need a Ptr")
	}

	// 结构升级(小)
	// 这里要现根据需要升级存档字符串
	//

	switch rv.Kind() {
	case reflect.Map:
		err := scanMap(values, rv)
		if err != nil {
			tilogs.L().Warnf("Restore To %s failed in ScanMap. err:%s", key, err.Error())
			return err
		}
	default:
		err := scanStruct(values, rv)
		if err != nil {
			tilogs.L().Warnf("Restore To %s failed in ScanStruct. err:%s", key, err.Error())
			return err
		}
	}
	if logInfo {
		tilogs.L().Debugf("Restore To %s success.", key)
	}
	return nil
}

func RedisSaveDataToOther(db redispool.RedisPoolConn, from, to string) bool {
	tilogs.L().Debugf("RedisSaveDataToOtherKey : %s -> %s", from, to)

	lua_src := `
		local from = KEYS[1]
		local to = KEYS[2]

		if redis.call("EXISTS", to) == 1 then
		  redis.call("DEL", to)
		end

		redis.call("RESTORE", to, 0, redis.call("DUMP", from))

		return "OK"
	`

	rc := db.RawConn()
	save_data_to_other_src := redis.NewScript(2, lua_src)
	save_data_to_other_src.Load(rc)

	res, err := redis.String(save_data_to_other_src.Do(rc, from, to))
	tilogs.L().Warnf("RedisSaveDataToOtherKey Re : %v  %v", err, res)

	return res == "OK"
}

func PanicIfErr(err error) bool {
	if err == nil {
		return false
	}

	if err == RESTORE_ERR_Profile_No_Data {
		return true
	}

	// NewAccount数据加载后如何处理玩家数据加载错误,
	// 如果玩家数据不存在是不会引发错误的。这里的错误应该是数据库自身的错误。
	// 此外此函数是通过GetMux|Player playerProcessor调用，是用户goroutine级别的错误，不会影响其他玩家
	panic(err)
}

func ParseFromHMGET(src []interface{}, v interface{}, fields ...string) error {
	values, err := buildHMGETValues(src, fields...)

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	} else {
		return errors.New("ParseFromHMGET need a Ptr")
	}

	switch rv.Kind() {
	case reflect.Map:
		err = scanMap(values, rv)
		if err != nil {
			tilogs.L().Warnf("ParseFromHMGET To %s failed in ScanMap. err:%s", err.Error())
			return err
		}
	default:
		err = scanStruct(values, rv)
		if err != nil {
			tilogs.L().Warnf("ParseFromHMGET To %s failed in ScanStruct. err:%s", err.Error())
			return err
		}
	}
	return nil
}

func buildHMGETValues(src []interface{}, fields ...string) ([]interface{}, error) {
	tempSrc := make([]interface{}, 0)
	if len(src) != len(fields) {
		return nil, fmt.Errorf("src and fields length not equal")
	}
	for i := range src {
		tempSrc = append(tempSrc, []byte(fields[i]), src[i])
	}
	return tempSrc, nil
}
