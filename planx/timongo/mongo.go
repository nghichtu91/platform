package timongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

/*
	大部分非Auth服务只需要使用单表单Collection
	这里实现了一个简单的抽象结构方便使用
*/

const (
	MongoConnectTimeOut = 3 * time.Second

	KeyUIDType = "UIDType"
	KeyNextUID = "NextUID"
)

var (
	ErrNilConfig      = errors.New("nil mongo config")
	ErrNilCollection  = errors.New("nil mongo collection")
	ErrNilPrimalKey   = errors.New("nil primal key")
	ErrInvalidIncrKey = errors.New("invalid incr key")
)

const (
	MongoCreateIndexFail = "Create index for %v with option %v failed" // param1: IndexKeyName, param2: Options
	MongoDropIndexFail   = "Drop index for %s failed with option %v"   // param1: IndexKeyName, param2: Options
)

// MongoDB 只包含单个Collection的轻量级封装
// 需要设置主键，单索引和多索引都依赖于
type MongoDB struct {
	Db         *mongo.Database
	Coll       *mongo.Collection
	PriKeyName string        // 主键
	Client     *mongo.Client // 关闭用
}

// MongoCfg 初始化Mongo需要的配置
type MongoCfg struct {
	URL        string
	DBName     string
	ColName    string
	PriKeyName string
}

// NewMongoConn 建立一个单表单链接单Key的Mongo连接
// 除了Auth服务外，基本满足其他服务的需求
func NewMongoConn(cfg *MongoCfg) (*MongoDB, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	ctx, cancel := context.WithTimeout(context.Background(), MongoConnectTimeOut)
	defer cancel()

	opt := options.Client().ApplyURI(cfg.URL)
	// opt.SetMaxPoolSize(0) // 如果没有特殊需求，不需要调整这个选项

	client, err := mongo.Connect(ctx, opt)
	if err != nil {
		return nil, err
	}

	db := new(MongoDB)
	db.Client = client

	// TODO 后续如果上mongo集群，可能需要对这里的参数做调整
	db.Db = client.Database(cfg.DBName,
		options.Database().SetReadConcern(readconcern.Local()),
		options.Database().SetReadPreference(readpref.Primary()),
		options.Database().SetWriteConcern(writeconcern.New(writeconcern.WMajority())))

	db.Coll = db.Db.Collection(cfg.ColName)
	db.PriKeyName = cfg.PriKeyName

	return db, nil
}

func (m *MongoDB) checkErr() error {
	switch {
	case m.Coll == nil:
		return ErrNilCollection
	case m.PriKeyName == "":
		return ErrNilPrimalKey
	}
	return nil
}

// GenSingleIndex 建立主键的索引
func (m *MongoDB) GenSingleIndex(isUnique bool) error {
	return CreateSingleIndex(m.Coll, m.PriKeyName, isUnique)
}

// GenDoubleIndex 将多个键值和主键建立连接索引
func (m *MongoDB) GenDoubleIndex(isUnique bool, keys ...string) error {
	keys = append([]string{m.PriKeyName}, keys...)
	return CreateCompoundIndexes(m.Coll, keys, isUnique)
}

// Close 一般情况下，推荐使用完mongo后关闭client链接
func (m *MongoDB) Close() error {
	return m.Client.Disconnect(context.Background())
}

// CreateSingleIndex 为collection建立单字段索引
func CreateSingleIndex(coll *mongo.Collection, indexKey string, isUnique bool) error {
	if coll == nil {
		return ErrNilCollection
	}

	indexView := coll.Indexes()

	opt := options.Index().
		SetUnique(isUnique).
		SetSparse(true) // 不存在时不索引

	idxName, err := indexView.CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys:    bsonx.Doc{{Key: indexKey, Value: bsonx.Int32(1)}},
			Options: opt,
		})

	if err != nil {
		tilogs.L().Errorf(MongoCreateIndexFail, indexKey, opt)
		return err
	}

	tilogs.L().Infof("Index for %v created, idx %v", indexKey, idxName)
	return nil
}

// CreateSingleIndexWithOpts 为collection建立自定义单字段索引
// TODO 检查已经存在的索引，如果字段一样但参数不一样，删除旧索引，建立新索引
func CreateSingleIndexWithOpts(coll *mongo.Collection, indexKey string, opt *options.IndexOptions) error {
	if coll == nil {
		return ErrNilCollection
	}

	indexView := coll.Indexes()

	idxName, err := indexView.CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys:    bsonx.Doc{{Key: indexKey, Value: bsonx.Int32(1)}},
			Options: opt,
		})

	if err != nil {
		tilogs.L().Errorf(MongoCreateIndexFail, indexKey, opt)
		return err
	}

	tilogs.L().Infof("Index for %v created, idx %v", indexKey, idxName)
	return nil
}

// DropSingleIndexWithOpts 删除指定单索引，为了表升级用
func DropSingleIndexWithOpts(coll *mongo.Collection, indexKey string, opt ...*options.DropIndexesOptions) error {
	if coll == nil {
		return ErrNilCollection
	}

	indexView := coll.Indexes()

	_, err := indexView.DropOne(
		context.Background(),
		indexKey,
		opt...,
	)

	if err != nil {
		// tilogs.L().Errorf(MongoDropIndexFail, indexKey, opt)
		return err
	}

	tilogs.L().Infof("Index for %v dropped", indexKey)
	return nil
}

// CreateCompoundIndexes 为collection建立复合索引
func CreateCompoundIndexes(coll *mongo.Collection, indexKeys []string, isUnique bool) error {
	if coll == nil {
		return ErrNilCollection
	}

	indexView := coll.Indexes()

	doc := bsonx.Doc{}
	for i := 0; i < len(indexKeys); i++ {
		doc = doc.Append(indexKeys[i], bsonx.Int32(1))
	}

	opt := options.Index().
		SetUnique(isUnique).
		SetSparse(true) // 不存在时不索引

	idxName, err := indexView.CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys:    doc,
			Options: opt,
		})

	if err != nil {
		tilogs.L().Errorf(MongoCreateIndexFail, doc, opt)
		return err
	}

	tilogs.L().Infof("Index for %v created, idx %v", indexKeys, idxName)
	return nil
}

// IncrRet Incr返回数据
type IncrRet struct {
	UIDType string
	NextUID int64
}

// Incr 原子性的返回自增的int值
// 如果对应key之前不存在，返回1
//
// 实现IAtomicDB接口
func (m *MongoDB) Incr(key string) (int64, error) {
	if err := m.checkErr(); err != nil {
		return 0, err
	}

	if key == "" {
		return 0, ErrInvalidIncrKey
	}

	r := m.Coll.FindOneAndUpdate(context.Background(),
		bson.M{
			m.PriKeyName: key,
		},
		bson.M{
			"$inc": bson.M{KeyNextUID: 1},
		},
		options.FindOneAndUpdate().SetUpsert(true))

	if r.Err() != nil {
		if errors.Is(r.Err(), mongo.ErrNoDocuments) {
			return 1, nil
		}
		return 0, r.Err()
	}

	ret := new(IncrRet)
	if err := r.Decode(ret); err != nil {
		return 0, err
	}

	tilogs.L().Infof("Incr success, uid type %s last uid %d", ret.UIDType, ret.NextUID)

	return ret.NextUID + 1, nil
}
