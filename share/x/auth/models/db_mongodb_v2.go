package models

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/x/auth/config_data"

	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/timongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

const (
	MongoConnectTimeOut     = 3 * time.Second // 连接超时
	DefaultMultiSliceLength = 16              // 默认slice长度
	URLDefaultPrefix        = "mongodb://"
)

const (
	// 返利类型。
	RewardAll          = 0 // 全部（仅支持查询）。
	RewardPay          = 1 // 内测充值返利（仅支持领取，需要其余操作随时扩展）。
	RewardLoginAndRank = 2 // 登陆和排名返利（仅支持领取，需要其余操作随时扩展）。

	// 操作类型。
	RewardOptClaim = "claim" // 领取返利。
	RewardOptQuery = "query" // 查询返利。

	// 返利操作结果反馈。
	RewardCodeOK     = 1 // 操作成功。
	RewardCodeError  = 2 // 操作异常。
	RewardCodeRepeat = 3 // 重复触发。
)

// Collection 类型枚举
// 不在此范围内的类型会报错
const (
	_ = iota
	ColTypeDevices
	// ColTypeUserInfos
	ColTypeUserShardInfos
	ColTypeSdkIdToUid
	ColTypeWhiteList
	ColTypeHistoryShard
)

// Collection 表名枚举
const (
	ColNameDevices = "Devices"
	// ColNameUserInfos      = "UserInfos"
	ColNameUserShardInfos = "UserShardInfos"
	ColNameSdkIdToUid     = "SdkIdToUid"
	ColNameWhiteList      = "WhiteList"
	ColNameHistoryShard   = "HistoryShard"
)

// Collection 第一主键枚举
const (
	KeyNameDeviceID = "id"
	KeyNameUserID   = "user_id"
	KeyNameSDKID    = "sdk_id"
	// KeyNameAuthToken      = "authtoken"
	KeyNameShardID        = "shard_id"
	KeyNamePlayerNumberID = "current_player_number"
)

// 索引枚举
// 使用索引进行查询性能较好
// 用之外的key进行查询需要注意性能问题
const (
	IdxUserID  = KeyNameUserID
	IdxGidSid  = "gid_sid"
	IdxDevice  = KeyNameDeviceID
	IdxDevice2 = "name"
	// IdxToken    = "create_at"
	IdxHistory  = KeyNameUserID
	IdxHistory2 = KeyNameShardID
)

var (
	ErrNoValidTableType  = errors.New("invalid table type")
	ErrNilSetData        = errors.New("nil set data")
	ErrInvalidSetData    = errors.New("invalid set data")
	ErrNilCollection     = errors.New("nil mongo collection")
	ErrNilPrimalKeyValue = errors.New("primal key value is nil") // 主键对应的值为空
	ErrInvalidUserID     = errors.New("invalid user id")
	ErrInvalidDeviceID   = errors.New("invalid device id")
	ErrInvalidName       = errors.New("invalid name")
)

type DBByMongoDBV2 struct {
	db     *mongo.Database
	cm     *Collections  // 本地
	client *mongo.Client // 关闭用
	mutex  sync.Mutex
}

// Collections 所有Collection
type Collections struct {
	Devices *mongo.Collection
	// UserInfos      *mongo.Collection
	UserShardInfos *mongo.Collection
	SdkIdToUid     *mongo.Collection
	WhiteList      *mongo.Collection
	HistoryShard   *mongo.Collection
}

func (mdb *DBByMongoDBV2) Init(cfg DBConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), MongoConnectTimeOut)
	defer cancel()

	// 兼容不写prefix的情况
	uri := cfg.MongoDBUrl
	if !strings.HasPrefix(uri, URLDefaultPrefix) {
		uri = URLDefaultPrefix + cfg.MongoDBUrl
	}
	opt := options.Client().ApplyURI(uri)
	// opt.SetMaxPoolSize(0) // 如果没有特殊需求，不需要调整这个选项

	client, err := mongo.Connect(ctx, opt)
	if err != nil {
		tilogs.L().Errorf("auth mongo connect fail, %v", err)
		return err
	}

	mdb.client = client

	mdb.db = client.Database(cfg.MongoDBName,
		options.Database().SetReadConcern(readconcern.Local()),
		options.Database().SetReadPreference(readpref.Primary()),
		options.Database().SetWriteConcern(writeconcern.New(writeconcern.WMajority())))

	mdb.cm = new(Collections)
	mdb.cm.Devices = mdb.db.Collection(ColNameDevices)
	// mdb.cm.UserInfos = mdb.db.Collection(ColNameUserInfos)
	mdb.cm.UserShardInfos = mdb.db.Collection(ColNameUserShardInfos)
	mdb.cm.SdkIdToUid = mdb.db.Collection(ColNameSdkIdToUid)
	mdb.cm.WhiteList = mdb.db.Collection(ColNameWhiteList)
	mdb.cm.HistoryShard = mdb.db.Collection(ColNameHistoryShard)

	// indexes
	if err := timongo.CreateSingleIndex(mdb.cm.Devices, IdxUserID, true); err != nil {
		return err
	}
	if err := timongo.CreateSingleIndex(mdb.cm.Devices, IdxDevice, true); err != nil {
		return err
	}
	if err := timongo.CreateSingleIndex(mdb.cm.Devices, IdxDevice2, true); err != nil {
		return err
	}

	// if err := timongo.CreateSingleIndex(mdb.cm.UserInfos, IdxUserID, true); err != nil {
	//	return err
	// }
	// expire类型的没有封装，单独写下
	// idxOpt := options.Index().SetExpireAfterSeconds(AUTHTOKEN_TIMEOUT)
	// if err := timongo.CreateSingleIndexWithOpts(mdb.cm.UserInfos, IdxToken, idxOpt); err != nil {
	//	return err
	// }

	// if err := timongo.CreateSingleIndex(mdb.cm.UserShardInfos, IdxUserID, true); err != nil {
	// 	return err
	// }

	// 升级旧表索引，删除原来的单一索引，建立双索引
	// TODO 后续可以删除这行逻辑
	if err := timongo.DropSingleIndexWithOpts(mdb.cm.UserShardInfos, "user_id_1"); err != nil {
		tilogs.L().Warnf("drop index for %s with err %s", IdxUserID, err)
	}

	if err := timongo.CreateCompoundIndexes(mdb.cm.UserShardInfos, []string{IdxUserID, IdxGidSid}, false); err != nil {
		return err
	}
	if err := timongo.CreateCompoundIndexes(mdb.cm.HistoryShard, []string{IdxHistory, IdxHistory2}, false); err != nil {
		return err
	}

	return nil
}

// Close 释放client连接
func (mdb *DBByMongoDBV2) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), MongoConnectTimeOut)
	defer cancel()

	if err := mdb.client.Disconnect(ctx); err != nil {
		tilogs.L().Errorf("close client connection failed, %v", err)
	}
}

// getPreset 获取预设的auth各张表的mongo键值
// pk是默认带索引的键值，查询性能较好
func (mdb *DBByMongoDBV2) getPreset(tableTyp int) (c *mongo.Collection, pk string, err error) {
	switch tableTyp {
	case ColTypeDevices:
		c = mdb.cm.Devices
		pk = KeyNameUserID
	// case ColTypeUserInfos:
	// 	c = mdb.cm.UserInfos
	// 	pk = KeyNameUserID
	case ColTypeUserShardInfos:
		c = mdb.cm.UserShardInfos
		pk = KeyNameUserID
	case ColTypeSdkIdToUid:
		c = mdb.cm.SdkIdToUid
		pk = KeyNameUserID
	case ColTypeWhiteList:
		c = mdb.cm.WhiteList
		pk = KeyNameUserID
	case ColTypeHistoryShard:
		c = mdb.cm.HistoryShard
		pk = KeyNameUserID
	default:
		err = ErrNoValidTableType
	}

	if c == nil {
		err = ErrNilCollection
	}

	return
}

// getPresetData 获取预设的auth各张表的data类型
// 对应GET和SET拆出data，减少内存使用
func (mdb *DBByMongoDBV2) getPresetData(tableTyp int) (data interface{}, err error) {
	switch tableTyp {
	case ColTypeDevices:
		data = new(MongoAuth)
	// case ColTypeUserInfos:
	// 	data = new(MongoTokens)
	case ColTypeUserShardInfos:
		data = new(MongoUserShardInfoV2)
	case ColTypeSdkIdToUid:
		data = new(SdkIdToUidInfos)
	case ColTypeWhiteList:
		data = new(WhiteListInfo)
	case ColTypeHistoryShard:
		data = new(MongoHistoryShard)
	default:
		err = ErrNoValidTableType
	}

	return
}

// getDataPrimKey 获得data的主键
func (mdb *DBByMongoDBV2) getDataPrimKey(data interface{}) (pk string, err error) {
	if data == nil {
		err = ErrNilSetData
		return
	}

	switch data.(type) {
	case *MongoAuth:
		pk = data.(*MongoAuth).DeviceID
	case *MongoTokens:
		pk = data.(*MongoTokens).UserID
	case *MongoUserShardInfoV2:
		pk = data.(*MongoUserShardInfoV2).UserID
	case *SdkIdToUidInfos:
		pk = data.(*SdkIdToUidInfos).SdkId
	case *WhiteListInfo:
		pk = data.(*WhiteListInfo).SdkId
	case *MongoHistoryShard:
		pk = data.(*MongoHistoryShard).UserId
	default:
		err = ErrInvalidSetData
	}

	return
}

// getSetMKey 生成用于set的MKey
// 同时对主键做了校验，不能传入空值
func (mdb *DBByMongoDBV2) getSetMKey(data interface{}) (mKey *bson.M, err error) {
	if data == nil {
		err = ErrNilSetData
		return
	}

	var k, v string
	switch data.(type) {
	case *MongoAuth:
		k = KeyNameDeviceID
		v = data.(*MongoAuth).DeviceID
	case *MongoTokens:
		k = KeyNameUserID
		v = data.(*MongoTokens).UserID
	case *MongoUserShardInfoV2:
		mKey = &bson.M{KeyNameUserID: data.(*MongoUserShardInfoV2).UserID, IdxGidSid: data.(*MongoUserShardInfoV2).AuthUserShardInfo.GidSid}
		return
	case *SdkIdToUidInfos:
		k = KeyNameSDKID
		v = data.(*SdkIdToUidInfos).SdkId
	case *WhiteListInfo:
		k = KeyNameSDKID
		v = data.(*WhiteListInfo).SdkId
	case *MongoHistoryShard:
		mKey = &bson.M{KeyNameUserID: data.(*MongoHistoryShard).UserId, KeyNameShardID: data.(*MongoHistoryShard).ShardId}
		return
	default:
		err = ErrInvalidSetData
		return
	}

	if v == "" {
		err = ErrNilPrimalKeyValue
		return
	}

	mKey = &bson.M{k: v}

	return
}

// set 写入逻辑
func (mdb *DBByMongoDBV2) set(c *mongo.Collection, data interface{}) error {
	out, err := bson.Marshal(data)
	if err != nil {
		return err
	}

	mKey, err := mdb.getSetMKey(data)
	if err != nil {
		return err
	}

	result := c.FindOneAndReplace(context.Background(),
		*mKey,
		out,
		options.FindOneAndReplace().SetUpsert(true))

	// 这里参考Err的文档，如果成功执行命令且不返回Document，会报下面这个错，特殊处理下
	if result.Err() == mongo.ErrNoDocuments {
		return nil
	}

	return result.Err()
}

// Set 全量写入
// 注意Set只依赖于主键，如果双主键表用Set，会写入多个值
// 如果需要持续更新，请使用Update
func (mdb *DBByMongoDBV2) Set(tableTyp int, data interface{}) error {
	c, _, err := mdb.getPreset(tableTyp)
	if err != nil {
		return err
	}

	return mdb.set(c, data)
}

// update 差量更新逻辑
// 包含主键的data使用这个接口进行更新
func (mdb *DBByMongoDBV2) update(c *mongo.Collection, data interface{}, upsert bool) error {
	out, err := bson.Marshal(data)
	if err != nil {
		return err
	}

	update := new(bson.M)
	err = bson.Unmarshal(out, update)
	if err != nil {
		return err
	}

	mKey, err := mdb.getSetMKey(data)
	if err != nil {
		return err
	}

	result := c.FindOneAndUpdate(context.Background(),
		*mKey,
		bson.M{"$set": *update},
		options.FindOneAndUpdate().SetUpsert(upsert))

	if result.Err() == mongo.ErrNoDocuments {
		if upsert {
			return nil
		}
		return XErrDBNotFound
	}

	return result.Err()
}

// updateWith 差量更新逻辑
// 不包含主键的data使用这个接口进行更新，需要额外指定主键名称和值
func (mdb *DBByMongoDBV2) updateWith(c *mongo.Collection, k, v string, data interface{}, upsert bool) error {
	out, err := bson.Marshal(data)
	if err != nil {
		return err
	}

	update := new(bson.M)
	err = bson.Unmarshal(out, update)
	if err != nil {
		return err
	}

	result := c.FindOneAndUpdate(context.Background(),
		bson.M{k: v},
		bson.M{"$set": *update},
		options.FindOneAndUpdate().SetUpsert(upsert))

	if result.Err() == mongo.ErrNoDocuments {
		if !upsert {
			return XErrDBNotFound
		}
	}

	return result.Err()
}

// updateAtomicWith 确保一致性的差量更新接口
// 需要额外指定主键名称和值，以及版本号
// ver为0时，可以自由更新文档
// 非0时，必须匹配版本号才能更新
// 无法通过这个方法新建文档
func (mdb *DBByMongoDBV2) updateAtomicWith(c *mongo.Collection, k, v string, ver int, data interface{}) error {
	out, err := bson.Marshal(data)
	if err != nil {
		return err
	}

	update := new(bson.M)
	err = bson.Unmarshal(out, update)
	if err != nil {
		return err
	}

	var filter bson.M
	if ver > 0 {
		filter = bson.M{k: v, "version": ver}
	} else {
		filter = bson.M{k: v}
	}

	result := c.FindOneAndUpdate(context.Background(),
		filter,
		bson.M{"$set": *update, "$inc": bson.M{"version": 1}})

	if result.Err() == mongo.ErrNoDocuments {
		return XErrDBNotFound
	}

	return result.Err()
}

// 更新数据并返回更新前的数据
func (mdb *DBByMongoDBV2) UpdateAndReturnOld(tableTyp int, k, v string, data interface{}, upsert bool) (bson.Raw, error) {
	c, _, err := mdb.getPreset(tableTyp)
	if err != nil {
		return nil, err
	}
	out, err := bson.Marshal(data)
	if err != nil {
		return nil, err
	}

	update := new(bson.M)
	err = bson.Unmarshal(out, update)
	if err != nil {
		return nil, err
	}
	result := c.FindOneAndUpdate(context.Background(),
		bson.M{k: v},
		bson.M{"$set": *update},
		options.FindOneAndUpdate().SetUpsert(upsert))
	if result.Err() == mongo.ErrNoDocuments {
		return nil, XErrDBNotFound
	}

	return result.DecodeBytes()
}

// Update 差量更新
// 对应mongo的upsert模式
func (mdb *DBByMongoDBV2) Update(tableTyp int, data interface{}, upsert bool) error {
	c, _, err := mdb.getPreset(tableTyp)
	if err != nil {
		return err
	}

	return mdb.update(c, data, upsert)
}

// UpdateWith 指定键值的差量更新
// 对应mongo的upsert模式
func (mdb *DBByMongoDBV2) UpdateWith(tableTyp int, k, v string, data interface{}, upsert bool) error {
	c, _, err := mdb.getPreset(tableTyp)
	if err != nil {
		return err
	}

	return mdb.updateWith(c, k, v, data, upsert)
}

func (mdb *DBByMongoDBV2) UpdateAtomicWith(tableTyp int, k, v string, version int, data interface{}) error {
	c, _, err := mdb.getPreset(tableTyp)
	if err != nil {
		return err
	}
	return mdb.updateAtomicWith(c, k, v, version, data)
}

// getOne 读取单个值逻辑
func (mdb *DBByMongoDBV2) getOne(c *mongo.Collection, tableTyp int, k, v string) (interface{}, error) {
	data, err := mdb.getPresetData(tableTyp)
	if err != nil {
		return nil, err
	}

	raw, err := c.FindOne(context.Background(),
		bson.M{k: v},
	).DecodeBytes()

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, XErrDBNotFound
		}
		return nil, err
	}

	if err := bson.Unmarshal(raw, data); err != nil {
		return nil, err
	}

	return data, nil
}

// GetOne 通用本地读表方法
// 对于错误处理，统一使用了XErrDBNotFound替代mongo自己的ErrNoDocuments
func (mdb *DBByMongoDBV2) GetOne(tableTyp int, v string) (interface{}, error) {
	c, k, err := mdb.getPreset(tableTyp)
	if err != nil {
		return nil, err
	}

	return mdb.getOne(c, tableTyp, k, v)
}

// GetOneWith 查询非主键的值
// 对于错误处理，统一使用了XErrDBNotFound替代mongo自己的ErrNoDocuments
func (mdb *DBByMongoDBV2) GetOneWith(tableTyp int, k, v string) (interface{}, error) {
	c, _, err := mdb.getPreset(tableTyp)
	if err != nil {
		return nil, err
	}

	return mdb.getOne(c, tableTyp, k, v)
}

// getAll 读取所有元素逻辑
func (mdb *DBByMongoDBV2) getAll(c *mongo.Collection, tableTyp int, k, v string) ([]interface{}, error) {
	filter := bson.M{k: v}
	if k == "" && v == "" {
		filter = bson.M{}
	}

	return mdb.getAllByFilter(c, tableTyp, &filter)
}

// getAllByFilter 使用filter进行查询
func (mdb *DBByMongoDBV2) getAllByFilter(c *mongo.Collection, tableTyp int, filter *bson.M) ([]interface{}, error) {
	cursor, err := c.Find(context.Background(), filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, XErrDBNotFound
		}
		return nil, err
	}

	result := make([]interface{}, 0, DefaultMultiSliceLength)

	for cursor.Next(context.Background()) {
		data, err := mdb.getPresetData(tableTyp)
		if err != nil {
			return nil, err
		}

		err = cursor.Decode(data)
		if err != nil {
			return nil, err
		}

		result = append(result, data)
	}

	cursor.Close(context.TODO())

	return result, nil
}

// GetAll 读取所有元素
func (mdb *DBByMongoDBV2) GetAll(tableTyp int, v string) ([]interface{}, error) {
	c, k, err := mdb.getPreset(tableTyp)
	if err != nil {
		return nil, err
	}

	return mdb.getAll(c, tableTyp, k, v)
}

// GetAllByFilter 按filter读取所有元素
func (mdb *DBByMongoDBV2) GetAllByFilter(tableTyp int, filter *bson.M) ([]interface{}, error) {
	c, _, err := mdb.getPreset(tableTyp)
	if err != nil {
		return nil, err
	}

	return mdb.getAllByFilter(c, tableTyp, filter)
}

// GetColAll 读取整个col信息
func (mdb *DBByMongoDBV2) GetColAll(tableTyp int) ([]interface{}, error) {
	c, _, err := mdb.getPreset(tableTyp)
	if err != nil {
		return nil, err
	}

	return mdb.getAll(c, tableTyp, "", "")
}

// insert 写入文档
// 和上面区别在于如果已经存在对应的数据，会报错
func (mdb *DBByMongoDBV2) insert(c *mongo.Collection, data interface{}) error {
	out, err := bson.Marshal(data)
	if err != nil {
		return err
	}

	_, err = c.InsertOne(context.Background(), out)

	return err
}

// delOne 删除一条数据
func (mdb *DBByMongoDBV2) delOne(c *mongo.Collection, k, v string) error {
	return mdb.delOneWithFilter(c, &bson.M{k: v})
}

// delOneWithFilter 按照filter条件删除一条数据
func (mdb *DBByMongoDBV2) delOneWithFilter(c *mongo.Collection, filter *bson.M) error {
	_, err := c.DeleteOne(context.Background(), filter)
	return err
}

// DelOne 删除指定主键的值
func (mdb *DBByMongoDBV2) DelOne(tableTyp int, v string) error {
	c, k, err := mdb.getPreset(tableTyp)
	if err != nil {
		return err
	}

	return mdb.delOne(c, k, v)
}

// DelOneWith 删除指定键的指定值
func (mdb *DBByMongoDBV2) DelOneWith(tableTyp int, k, v string) error {
	c, _, err := mdb.getPreset(tableTyp)
	if err != nil {
		return err
	}

	return mdb.delOne(c, k, v)
}

// DelOneWithFilter 按照filter条件删除一条数据
func (mdb *DBByMongoDBV2) DelOneWithFilter(tableTyp int, filter *bson.M) error {
	c, _, err := mdb.getPreset(tableTyp)
	if err != nil {
		return err
	}

	return mdb.delOneWithFilter(c, filter)
}

/*
	下面开始接口实现
*/

// GetDeviceInfo 根据DeviceID查找是否存在已注册信息
func (mdb *DBByMongoDBV2) GetDeviceInfo(deviceID string, checkGM bool) (*DeviceUserInfo, error, bool) {
	var isGM bool
	r, err := mdb.GetOneWith(ColTypeDevices, KeyNameDeviceID, deviceID)
	if err != nil {
		return nil, err, false
	}
	ma := r.(*MongoAuth)

	if checkGM {
		gr, err := mdb.GetOneWith(ColTypeSdkIdToUid, KeyNameSDKID, deviceID)
		if err == nil {
			if gr != nil && gr.(*SdkIdToUidInfos).Uid != "" {
				ma.UserID = gr.(*SdkIdToUidInfos).Uid
				isGM = true
			}
		}
	}

	return ma.GetDeviceUserInfoPtr(), nil, isGM
}

// UpdateDeviceInfo 更新Device信息
func (mdb *DBByMongoDBV2) UpdateDeviceInfo(deviceID string, info *DeviceUserInfo) error {
	return mdb.UpdateWith(ColTypeDevices, KeyNameDeviceID, deviceID, &MongoAuth{
		DeviceID:     deviceID,
		LastAuthTime: time.Now().Unix(),
	}, true)
}

// SetDeviceInfo 设置Device信息
func (mdb *DBByMongoDBV2) SetDeviceInfo(deviceID, name string, uid db.UserID, channelID, device string) (*DeviceUserInfo, error) {
	ma := &MongoAuth{
		DeviceID:     deviceID,
		DisplayName:  "anonymous",
		UserID:       uid.String(),
		ChannelID:    channelID,
		Device:       device,
		LastAuthTime: time.Now().Unix(),
		CreateTime:   time.Now().Unix(),
		NameAuth:     name,
	}

	if err := mdb.Update(ColTypeDevices, ma, true); err != nil {
		return nil, err
	}

	tilogs.L().Debugf("DBByMongoDB.SetDeviceInfo %+v", ma)

	return ma.GetDeviceUserInfoPtr(), nil
}

// IsNameExist 判断玩家是否使用了用户名模式
func (mdb *DBByMongoDBV2) IsNameExist(name string) (int, error) {
	if name == "" {
		return -1, ErrInvalidName
	}
	v, err := mdb.GetOneWith(ColTypeDevices, IdxDevice2, name)
	switch err {
	case XErrDBNotFound:
	case nil:
		if v != nil {
			return 1, nil
		}
	default:
		return -1, err
	}
	return 0, nil
}

// GetUnKey 根据name查询uid
func (mdb *DBByMongoDBV2) GetUnKey(name string, checkGM bool) (db.UserID, error, bool) {
	if name == "" {
		return db.InvalidUserID, ErrInvalidName, checkGM
	}
	v, err := mdb.GetOneWith(ColTypeDevices, IdxDevice2, name)
	switch err {
	case XErrDBNotFound:
		err = XErrAuthUsernameNotFound
	case nil:
		ma := v.(*MongoAuth)
		// checkGM
		if checkGM {
			v2, err := mdb.GetOneWith(ColTypeSdkIdToUid, KeyNameSDKID, ma.DeviceID)
			if err == nil {
				info := v2.(*SdkIdToUidInfos)
				tilogs.L().Infof("成功替换uid, old uid is %s,new uid is %s", ma.UserID, info.Uid)
				return db.UserIDFromStringOrNil(info.Uid), nil, true
			}
		}
		return db.UserIDFromStringOrNil(ma.UserID), nil, false
	}
	return db.InvalidUserID, err, false
}

// GetUnKeyV2 根据name查询uid
func (mdb *DBByMongoDBV2) GetUnKeyV2(ctx context.Context, name string, checkGM bool) (db.UserID, error, bool) {
	span, _ := opentracing.StartSpanFromContext(ctx, "GetUnKey")
	span.SetTag("DBType", "MongoV2")
	span.SetTag("OpType", "GetOneWith")
	span.SetTag("ColName", ColNameDevices)
	defer span.Finish()

	return mdb.GetUnKey(name, checkGM)
}

// SetUnKey
// 对应doc找不到会返回ErrNoDoc
func (mdb *DBByMongoDBV2) SetUnKey(name string, uid db.UserID) error {
	if name == "" {
		return ErrInvalidName
	}
	if !uid.IsValid() {
		return ErrInvalidUserID
	}
	return mdb.UpdateWith(ColTypeDevices, IdxDevice2, name, &MongoAuth{
		UserID:   uid.String(),
		NameAuth: name,
	}, false)
}

// GetUnInfo 获取对应信息
func (mdb *DBByMongoDBV2) GetUnInfo(uid db.UserID) (*UserInfo, error) {
	if !uid.IsValid() {
		return nil, ErrInvalidUserID
	}
	v, err := mdb.GetOneWith(ColTypeDevices, IdxUserID, uid.String())
	if err != nil {
		return nil, err
	}

	return v.(*MongoAuth).GetUserInfo(), nil
}

// GetUnInfo 获取对应信息
func (mdb *DBByMongoDBV2) GetUnInfoV2(ctx context.Context, uid db.UserID) (*UserInfo, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "GetUnInfo")
	span.SetTag("DBType", "MongoV2")
	span.SetTag("OpType", "GetOneWith")
	span.SetTag("ColName", ColNameDevices)
	defer span.Finish()

	return mdb.GetUnInfo(uid)
}

// UpdateUnInfo 更新信息
func (mdb *DBByMongoDBV2) UpdateUnInfo(uid db.UserID, deviceID, authToken string) error {
	if deviceID != "" {
		return mdb.Update(ColTypeDevices, &MongoAuth{DeviceID: deviceID, AuthToken: authToken}, true)
	} else if uid.IsValid() {
		return mdb.UpdateWith(ColTypeDevices, IdxUserID, uid.String(), &MongoAuth{AuthToken: authToken}, true)
	}
	return nil
}

func (mdb *DBByMongoDBV2) UpdateUnInfoV2(ctx context.Context, uid db.UserID, deviceID, authToken string) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "UpdateUnInfo")
	span.SetTag("DBType", "MongoV2")
	span.SetTag("OpType", "Update")
	span.SetTag("ColName", ColNameDevices)
	defer span.Finish()

	return mdb.UpdateUnInfo(uid, deviceID, authToken)
}

// UpdateBanUn 更新封禁信息
func (mdb *DBByMongoDBV2) UpdateBanUn(uid string, timeToBan int64, reason string) error {
	if uid == "" {
		return ErrInvalidUserID
	}
	return mdb.UpdateWith(ColTypeDevices, IdxUserID, uid, &MongoAuth{
		BanTime:   time.Now().Unix() + timeToBan,
		BanReason: reason,
	}, false)
}

// 更新封禁角色信息
func (mdb *DBByMongoDBV2) UpdateBanPlayer(uid string, acid string, banTime int64, reason string) error {
	if uid == "" {
		return ErrInvalidUserID
	}
	v, err := mdb.GetOneWith(ColTypeDevices, IdxUserID, uid)
	if err != nil {
		return ErrInvalidUserID
	}
	mongoAuth := v.(*MongoAuth)
	if mongoAuth.BanPlayerInfos == nil {
		mongoAuth.BanPlayerInfos = make(map[string]BanPlayerInfo, 1)
	}
	info, ok := mongoAuth.BanPlayerInfos[acid]
	if !ok {
		info = BanPlayerInfo{
			UserID: uid,
			Acid:   acid,
		}
	}
	info.BanEndTime = banTime
	info.BanResult = reason
	mongoAuth.BanPlayerInfos[acid] = info

	return mdb.UpdateAtomicWith(ColTypeDevices, IdxUserID, uid, mongoAuth.Version, &MongoAuth{
		BanPlayerInfos: mongoAuth.BanPlayerInfos})
}

func (mdb *DBByMongoDBV2) GetPlayerBanInfo(uid string, acid string) *BanPlayerInfo {
	if uid == "" {
		return nil
	}
	v, err := mdb.GetOneWith(ColTypeDevices, IdxUserID, uid)
	if err != nil {
		return nil
	}
	mongoAuth := v.(*MongoAuth)
	if mongoAuth.BanPlayerInfos == nil {
		return nil
	}
	info, ok := mongoAuth.BanPlayerInfos[acid]
	if !ok {
		return nil
	}
	banInfo := &BanPlayerInfo{
		UserID:     info.UserID,
		Acid:       info.Acid,
		BanEndTime: info.BanEndTime,
		BanResult:  info.BanResult,
	}
	return banInfo
}

// UpdateGagUn 更新禁言信息
func (mdb *DBByMongoDBV2) UpdateGagUn(uid string, timeToGag int64) error {
	if uid == "" {
		return ErrInvalidUserID
	}
	return mdb.UpdateWith(ColTypeDevices, IdxUserID, uid, &MongoAuth{
		GagTime: time.Now().Unix() + timeToGag,
	}, false)
}

// SetUnInfo 设置信息
func (mdb *DBByMongoDBV2) SetUnInfo(uid db.UserID, name, deviceID, passwd, email, authToken string) error {
	return mdb.SetUnInfoPass(uid, name, deviceID,
		hex.EncodeToString(secure.DefaultEncode.PasswordForDB(passwd)),
		email, authToken)
}

// SetUnInfoPass GMTools中，用来乾坤大挪移。用来把一个accountid映射到一个有用户名密码的帐号下。方便查找问题
func (mdb *DBByMongoDBV2) SetUnInfoPass(uid db.UserID, name, deviceID, passwd, email, authToken string) error {
	return mdb.Update(ColTypeDevices, &MongoAuth{
		DeviceID:     deviceID,
		UserID:       uid.String(),
		NameAuth:     name,
		NameAuthPwd:  passwd,
		LastAuthTime: time.Now().Unix(),
		BanTime:      0,
		GagTime:      0,
	}, true)
}

/*
// JWS2-19885 token改用redis处理
// GetAuthToken 获取AuthToken
func (mdb *DBByMongoDBV2) GetAuthToken(authToken string) (db.UserID, string, error) {
	// TODO 这个判断是兼容旧接口，让Unity传空token可以进
	if authToken == "" {
		return db.InvalidUserID, "", nil
	}
	v, err := mdb.GetOneWith(ColTypeUserInfos, KeyNameAuthToken, authToken)
	switch err {
	case XErrDBNotFound:
		err = XErrLoginAuthtokenNotFound
	case nil:
		return db.UserIDFromStringOrNil(v.(*MongoTokens).UserID), v.(*MongoTokens).SdkDeviceId, nil
	}
	return db.InvalidUserID, "", err
}

// SetAuthToken 设置AuthToken
func (mdb *DBByMongoDBV2) SetAuthToken(authToken string, userID db.UserID, timeOut int64, sdkDeviceId string) error {
	return mdb.Update(ColTypeUserInfos, &MongoTokens{
		UserID:      userID.String(),
		AuthToken:   authToken,
		CreateAt:    time.Now(),
		SdkDeviceId: sdkDeviceId,
	}, true)
}
*/

// GetUserShardInfo 获取玩家在哪些服有账号信息
func (mdb *DBByMongoDBV2) GetUserShardInfo(uid string) ([]AuthUserShardInfo, error) {
	v, err := mdb.GetAll(ColTypeUserShardInfos, uid)
	if err != nil {
		return nil, err
	}
	si := make([]AuthUserShardInfo, len(v))
	for i := range v {
		si[i] = v[i].(*MongoUserShardInfoV2).AuthUserShardInfo
	}
	return si, nil
}

// SetUserShardInfo 设置玩家的账号信息
func (mdb *DBByMongoDBV2) SetUserShardInfo(uid, gidSidStr, roleName, roleLevel, mainHero, lastLoginTime, mainlineLevel, headIcon, playerId string) error {
	return mdb.Update(ColTypeUserShardInfos, &MongoUserShardInfoV2{
		UserID: uid,
		AuthUserShardInfo: AuthUserShardInfo{
			GidSid:        gidSidStr,
			RoleName:      roleName,
			RoleLevel:     roleLevel,
			MainHero:      mainHero,
			LastLoginTime: lastLoginTime,
			MainlineLevel: mainlineLevel,
			HeadIcon:      headIcon,
			PlayerId:      playerId,
		},
	}, true)
}

// GetUserHistoryShard 获取历史登录记录
func (mdb *DBByMongoDBV2) GetUserHistoryShard(uid string) ([]string, error) {
	if uid == "" {
		return nil, ErrInvalidUserID
	}
	v, err := mdb.GetAll(ColTypeHistoryShard, uid)
	if err != nil {
		return nil, err
	}
	si := make([]string, len(v))
	for i := range v {
		si[i] = v[i].(*MongoHistoryShard).ShardId
	}
	return si, nil
}

// SetUserHistoryShard 设置历史登录记录
func (mdb *DBByMongoDBV2) SetUserHistoryShard(uid, shard string) error {
	if uid == "" || shard == "" {
		return nil
	}
	return mdb.Update(ColTypeHistoryShard, &MongoHistoryShard{
		UserId:  uid,
		ShardId: shard,
	}, true)
}

// SetLastLoginShard 设置上次登录信息
// 如果uid不存在，报错
func (mdb *DBByMongoDBV2) SetLastLoginShard(uid string, gidsid string) error {
	tilogs.L().Debugf("SetLastLoginShard uid %s gidsid %s", uid, gidsid)
	if uid == "" || gidsid == "" {
		return nil
	}
	return mdb.UpdateWith(ColTypeDevices, IdxUserID, uid, &MongoAuth{UserID: uid, LastGidSid: gidsid}, false)
}

// GetLastLoginShard 获取上次登录信息
func (mdb *DBByMongoDBV2) GetLastLoginShard(uid string) (string, error) {
	if uid == "" {
		return "", nil
	}
	v, err := mdb.GetOneWith(ColTypeDevices, IdxUserID, uid)
	if err != nil {
		return "", err
	}
	return v.(*MongoAuth).LastGidSid, nil
}

// SetPayFeedBackIfNot 标记PayFeedBack为已领
// 只有数据存在且Payfeeback不为"got"，才会进行操作
func (mdb *DBByMongoDBV2) SetPayFeedBackIfNot(uid string) (error, bool) {
	if uid == "" {
		return ErrInvalidUserID, false
	}
	v, err := mdb.GetAllByFilter(ColTypeDevices, &bson.M{IdxUserID: uid})
	if err != nil {
		return err, false
	}
	if len(v) != 1 {
		return fmt.Errorf("SetPayFeedBackIfNot result found != 1"), false
	}

	if v[0].(*MongoAuth).Payfeeback != "got" {
		if err := mdb.UpdateWith(ColTypeDevices, IdxUserID, uid, &MongoAuth{
			UserID:     uid,
			Payfeeback: "got",
		}, false); err != nil {
			return err, false
		}
		return nil, true
	}
	return nil, false
}

/*
	没有逻辑，只是为了实现接口
*/

func (mdb *DBByMongoDBV2) SetRecallPlayerInfo(recallId, uid string, recallnum, gid int64) error {
	return nil
}
func (mdb *DBByMongoDBV2) UpdateRecallID(uid db.UserID, recallID string) error { return nil }
func (mdb *DBByMongoDBV2) MarketGetByHashM(recallId string) (bool, error)      { return false, nil }
func (mdb *DBByMongoDBV2) QueryRecallIdByAcid(accountId string) (map[string]interface{}, error) {
	return nil, nil
}

// 白名单操作

func (mdb *DBByMongoDBV2) AddWhiteList(ip, note string) error     { return nil }
func (mdb *DBByMongoDBV2) CheckWhiteList(ip string) (bool, error) { return false, nil }

// InsertOrUpdateWhiteList 添加一个白名单，返回所有白名单列表
func (mdb *DBByMongoDBV2) InsertOrUpdateWhiteList(infos *WhiteListInfo) ([]*WhiteListInfo, error) {
	err := mdb.Update(ColTypeWhiteList, infos, true)
	if err != nil {
		return nil, err
	}

	return QueryAllWhiteList()
}

// DeleteWhiteList 删除指定字段白名单，返回所有白名单列表
func (mdb *DBByMongoDBV2) DeleteWhiteList(content string, typ uint) ([]*WhiteListInfo, error) {
	err := mdb.DelOneWithFilter(ColTypeWhiteList, &bson.M{"typ": typ, "content": content})
	if err != nil {
		return nil, err
	}

	return QueryAllWhiteList()
}

// QueryAllWhiteList 查询所有白名单
func (mdb *DBByMongoDBV2) QueryAllWhiteList() ([]*WhiteListInfo, error) {
	v, err := mdb.GetColAll(ColTypeWhiteList)
	if err != nil {
		return nil, err
	}
	ret := make([]*WhiteListInfo, len(v))
	for i, vv := range v {
		ret[i] = vv.(*WhiteListInfo)
	}

	return ret, nil
}

// QueryWhiteList 按指定条件查询白名单
func (mdb *DBByMongoDBV2) QueryWhiteList(typ uint, param string) ([]*WhiteListInfo, error) {
	var filter bson.M
	if typ == 3 {
		filter = bson.M{"typ": typ, "sdk_id": param}
	} else {
		filter = bson.M{"typ": typ, "content": param}
	}
	v, err := mdb.GetAllByFilter(ColTypeWhiteList, &filter)
	if err != nil {
		return nil, err
	}
	ret := make([]*WhiteListInfo, len(v))
	for i, vv := range v {
		ret[i] = vv.(*WhiteListInfo)
	}

	return ret, nil
}

// GetSdkDeviceIdByUid 查询UID对应的SdkDeviceID
func (mdb *DBByMongoDBV2) GetSdkDeviceIdByUid(uid string) (string, error) {
	r, err := mdb.GetOneWith(ColTypeDevices, IdxUserID, uid)
	if err != nil {
		return "", err
	}
	return r.(*MongoAuth).DeviceID, nil
}

// InsertOrUpdateSdkIdToUid 修改一条信息，返回表内所有信息
func (mdb *DBByMongoDBV2) InsertOrUpdateSdkIdToUid(infos *SdkIdToUidInfos) ([]*SdkIdToUidInfos, error) {
	err := mdb.UpdateWith(ColTypeSdkIdToUid, KeyNamePlayerNumberID, infos.CurrentPlayerNumber, infos, true)
	if err != nil {
		return nil, err
	}

	return mdb.QuerySdkIdToUid()
}

// DeleteSdkIdToUid 删除一条信息，返回表内所有信息
func (mdb *DBByMongoDBV2) DeleteSdkIdToUid(numberId string, shard uint) ([]*SdkIdToUidInfos, error) {
	err := mdb.DelOneWith(ColTypeSdkIdToUid, KeyNamePlayerNumberID, numberId)
	if err != nil {
		return nil, err
	}

	return mdb.QuerySdkIdToUid()
}

// QuerySdkIdToUid 获取表内所有信息
func (mdb *DBByMongoDBV2) QuerySdkIdToUid() ([]*SdkIdToUidInfos, error) {
	v, err := mdb.GetColAll(ColTypeSdkIdToUid)
	if err != nil {
		return nil, err
	}
	ret := make([]*SdkIdToUidInfos, len(v))
	for i, vv := range v {
		ret[i] = vv.(*SdkIdToUidInfos)
	}

	return ret, nil
}

// QueryUserSpecialAward 获取该账号下已领取过的特殊活动的奖励信息
func (mdb *DBByMongoDBV2) QueryUserSpecialAward(uid db.UserID) (string, error) {
	v, err := mdb.GetOneWith(ColTypeDevices, IdxUserID, uid.String())
	if err != nil {
		return "", err
	}
	return v.(*MongoAuth).SpecialAward, nil
}

// UpdateUserSpecialAward 更新该账号下已领取过的特殊活动的奖励信息
func (mdb *DBByMongoDBV2) UpdateUserSpecialAward(uid db.UserID, awardStr string) error {
	return mdb.UpdateWith(ColTypeDevices, IdxUserID, uid.String(), &MongoAuth{SpecialAward: awardStr}, false)
}

// 充值返利。
func (mdb *DBByMongoDBV2) RewardForUser(rewardType int, userID db.UserID, optType string) (map[int]*config_data.RewardInfo, int32) {
	tilogs.L().Infof("DBByMongoDBV2 RewardForUser rewardType: %v, userID: %v, optType: %v", rewardType, userID, optType)

	switch rewardType {
	case RewardAll:
		// 全部（目前仅支持查询）。
		if optType != RewardOptQuery {
			tilogs.L().Errorf("DBByMongoDBV2 RewardForUser when rewardType is RewardAll, the optType must be RewardOptQuery, userID: %v, optType: %v",
				userID, optType)
			return nil, RewardCodeError
		}

		// 查询全部。
		return mdb.rewardAllHandle(userID.String())
	case RewardPay:
		// 内测充值返利（仅支持领取，需要其余操作随时扩展）。
		if optType != RewardOptClaim {
			tilogs.L().Errorf("DBByMongoDBV2 RewardForUser when rewardType is RewardPay, the optType must be RewardOptClaim, userID: %v, optType: %v",
				userID, optType)
			return nil, RewardCodeError
		}

		// 领取充值返利。
		return mdb.rewardPayClaim(userID.String())
	case RewardLoginAndRank:
		// 登陆和排名返利（仅支持领取，需要其余操作随时扩展）。
		if optType != RewardOptClaim {
			tilogs.L().Errorf("DBByMongoDBV2 RewardForUser when rewardType is RewardLoginAndRank, the optType must be RewardOptClaim, userID: %v, optType: %v",
				userID, optType)
			return nil, RewardCodeError
		}

		// 领取登陆排名返利。
		return mdb.rewardLoginAndRankClaim(userID.String())
	default:
		tilogs.L().Errorf("DBByMongoDBV2 RewardForUser unknown rewardType, rewardType: %v", rewardType)
		return nil, RewardCodeError
	}
}

func (mdb *DBByMongoDBV2) rewardAllHandle(userID string) (map[int]*config_data.RewardInfo, int32) {
	// 玩家充值返利配置。
	payRewardCfg := config_data.GetPayRewardConfig(userID)

	// 玩家登陆和排名配置。
	loginAndRankRewardCfg := config_data.GetLoginAndRankRewardConfig(userID)

	// 玩家没有任何返利配置。
	if payRewardCfg == nil && loginAndRankRewardCfg == nil {
		return nil, RewardCodeOK
	}

	// 获取玩家的MongoAuth信息。
	mongoAuth, errAuth := mdb.queryMongoAuthInfo(userID)
	if errAuth != nil {
		return nil, RewardCodeError
	}

	ret := make(map[int]*config_data.RewardInfo, 2)
	if payRewardCfg != nil {
		// 玩家存在支付返利配置。
		ret[RewardPay] = &config_data.RewardInfo{
			HasClaimed: mongoAuth.PayReward,     // 是否已领取返利。
			MoneyScore: payRewardCfg.MoneyScore, // 玩家累计充值积分。
		}
	}
	if loginAndRankRewardCfg != nil {
		// 玩家存在登陆排名返利配置。
		ret[RewardLoginAndRank] = &config_data.RewardInfo{
			HasClaimed: mongoAuth.LoginRankReward,        // 是否已领取返利。
			LoginDay:   loginAndRankRewardCfg.LoginDay,   // 登陆天数。
			Fair1v1Dan: loginAndRankRewardCfg.Fair1v1Dan, // 公平1v1段位。
			Fair3v3Dan: loginAndRankRewardCfg.Fair3v3Dan, // 公平3v3段位。
			ClaimTime:  mongoAuth.LoginRewardTime,        // 返利奖励领取时间
		}
	}

	return ret, RewardCodeOK
}

func (mdb *DBByMongoDBV2) rewardPayClaim(userID string) (map[int]*config_data.RewardInfo, int32) {
	// 玩家充值返利配置。
	payRewardCfg := config_data.GetPayRewardConfig(userID)
	if payRewardCfg == nil {
		tilogs.L().Infof("DBByMongoDBV2 rewardPayHandle no award, cfg is nil, userID: %v", userID)
		return nil, RewardCodeOK
	}

	oldRaw, errRaw := mdb.UpdateAndReturnOld(ColTypeDevices, IdxUserID, userID, &MongoAuth{PayReward: true}, false)
	if errRaw != nil {
		tilogs.L().Errorf("DBByMongoDBV2 rewardPayHandle UpdateAndReturnOld failed, errRaw: %v, userID: %v", errRaw, userID)
		return nil, RewardCodeError
	}

	oldData := &MongoAuth{}
	if errUnmarshal := bson.Unmarshal(oldRaw, oldData); errUnmarshal != nil {
		tilogs.L().Errorf("DBByMongoDBV2 rewardPayHandle bson.Unmarshal failed, errUnmarshal: %v, userID: %v", errUnmarshal, userID)
		return nil, RewardCodeError
	}

	if oldData.PayReward {
		// 玩家已领取过。
		tilogs.L().Infof("DBByMongoDBV2 rewardPayHandle RewardCodeRepeat, userID: %v", userID)
		return nil, RewardCodeRepeat
	} else {
		// 玩家成功领取。
		tilogs.L().Infof("DBByMongoDBV2 rewardPayHandle success claim, userID: %v", userID)
		ret := make(map[int]*config_data.RewardInfo, 1)
		ret[RewardPay] = &config_data.RewardInfo{
			HasClaimed: true,                    // 是否已领取返利。
			MoneyScore: payRewardCfg.MoneyScore, // 玩家累计充值积分。
		}
		return ret, RewardCodeOK
	}
}

func (mdb *DBByMongoDBV2) rewardLoginAndRankClaim(userID string) (map[int]*config_data.RewardInfo, int32) {
	// 玩家登陆和排名配置。
	loginAndRankRewardCfg := config_data.GetLoginAndRankRewardConfig(userID)
	if loginAndRankRewardCfg == nil {
		tilogs.L().Infof("DBByMongoDBV2 rewardLoginAndRankClaim no award, cfg is nil, userID: %v", userID)
		return nil, RewardCodeOK
	}

	now := timeutil.Now().Unix()
	oldRaw, errRaw := mdb.UpdateAndReturnOld(ColTypeDevices, IdxUserID, userID, &MongoAuth{LoginRankReward: true, LoginRewardTime: now}, false)
	if errRaw != nil {
		tilogs.L().Errorf("DBByMongoDBV2 rewardLoginAndRankClaim UpdateAndReturnOld failed, errRaw: %v, userID: %v", errRaw, userID)
		return nil, RewardCodeError
	}

	oldData := &MongoAuth{}
	if errUnmarshal := bson.Unmarshal(oldRaw, oldData); errUnmarshal != nil {
		tilogs.L().Errorf("DBByMongoDBV2 rewardLoginAndRankClaim bson.Unmarshal failed, errUnmarshal: %v, userID: %v", errUnmarshal, userID)
		return nil, RewardCodeError
	}

	if oldData.LoginRankReward {
		// 玩家已领取过。
		tilogs.L().Infof("DBByMongoDBV2 rewardLoginAndRankClaim RewardCodeRepeat, userID: %v", userID)
		return nil, RewardCodeRepeat
	} else {
		// 玩家成功领取。
		tilogs.L().Infof("DBByMongoDBV2 rewardLoginAndRankClaim success claim, userID: %v", userID)
		ret := make(map[int]*config_data.RewardInfo, 1)
		ret[RewardLoginAndRank] = &config_data.RewardInfo{
			HasClaimed: true,                             // 是否已领取返利。
			LoginDay:   loginAndRankRewardCfg.LoginDay,   // 登陆天数。
			Fair1v1Dan: loginAndRankRewardCfg.Fair1v1Dan, // 公平1v1段位。
			Fair3v3Dan: loginAndRankRewardCfg.Fair3v3Dan, // 公平3v3段位。
			ClaimTime:  now,
		}
		return ret, RewardCodeOK
	}
}

// queryMongoAuthInfo 查询指定玩家账号的MongoAuth信息。
func (mdb *DBByMongoDBV2) queryMongoAuthInfo(userID string) (*MongoAuth, error) {
	info, err := mdb.GetOneWith(ColTypeDevices, IdxUserID, userID)
	if err != nil {
		tilogs.L().Errorf("DBByMongoDBV2 queryMongoAuthInfo failed, err: %v, userID: %v", err, userID)
		return nil, err
	}

	mongoAuth := info.(*MongoAuth)
	if mongoAuth == nil {
		errAuth := fmt.Errorf("DBByMongoDBV2 queryMongoAuthInfo convert failed, info: %v, userID: %v", info, userID)
		tilogs.L().Errorf(errAuth.Error())
		return nil, errAuth
	}

	return mongoAuth, nil
}
