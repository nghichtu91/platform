package cloud_db

import (
	"github.com/nghichtu91/platform/share/planx/cloud_db/gcsdb"
	"github.com/nghichtu91/platform/share/planx/cloud_db/obsdb"
	"github.com/nghichtu91/platform/share/planx/cloud_db/ossdb"
	"github.com/nghichtu91/platform/share/planx/cloud_db/s3db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

const (
	CloudDb_S3     = "S3"
	CloudDb_Aliyun = "aliyun"
	CloudDb_Huawei = "huawei"
	CloudDb_Google = "gcp"
	CloudDb_R2     = "R2"
)

// 云存储 S3， OSS
// 以下不带 WithBucket 的接口，路径前都会自动拼上cloudDbRoot
type CloudDb interface {
	Open() error
	Close() error
	Put(key string, val []byte) error
	PutWithBucket(cloudDbRoot, key string, val []byte, bucket string) error
	PutWithMeta(key string, val []byte, metaKey string, metaVal string) error
	Get(key string) ([]byte, error)
	GetWithBucket(bucket, cloudDbRoot, key string) ([]byte, error)
	GetMeta(key, metaKey string) (*string, error)
	Del(key string) error
	DelWithBucket(bucket, cloudDbRoot, key string) error
	Copy(oldKey string, newKey string) error
	CopyWithBucket(bucketName, cloudDBRoot, oldKey string, newKey string) error
	// 下面两个list接口返回的是带cloudDbRoot的全路径，若要用返回的路径再做get或put等操作，就需要用WithBucket的接口，并且cloudDbRoot填空
	ListObject(last_idx string, scan_len int64) ([]string, string, error)
	// 根据传入参数对最近修改时间进行过滤
	ListObjectWithBucketRecent(bucket, last_idx string, scan_len, recentDay int64, cloudDbRoot, prefix string) ([]string, string, error)
	ListObjectWithBucket(bucket, last_idx string, scan_len int64, cloudDbRoot, prefix string) ([]string, string, error)

	CreateSignedUrl(bucket, key string, expires int) (error, string)
}

type CloudDbConfig struct {
	DbDriver    string
	Region      string
	Bucket      string
	CloudDbRoot string // 为了运维好隔离不同大区下的数据，增加CloudDbRoot，作为bucket下的根路径
	AccessKey   string
	SecretKey   string
	Format      string
	Seq         string
}

func NewCloudDB(cfg CloudDbConfig) CloudDb {
	switch cfg.DbDriver {
	case CloudDb_S3:
		return s3db.NewStoreS3(cfg.Region,
			cfg.AccessKey, cfg.SecretKey, cfg.Bucket, cfg.CloudDbRoot, "", "", false)
	case CloudDb_Aliyun:
		return ossdb.NewStoreOSS(cfg.Region,
			cfg.AccessKey, cfg.SecretKey, cfg.Bucket, cfg.CloudDbRoot, "", "")
	case CloudDb_Huawei:
		return obsdb.NewStoreOBS(cfg.Region,
			cfg.AccessKey, cfg.SecretKey, cfg.Bucket, cfg.CloudDbRoot, "", "")
	case CloudDb_Google:
		return gcsdb.NewStoreGCS(cfg.Bucket, cfg.CloudDbRoot)
	case CloudDb_R2:
		return s3db.NewStoreS3(cfg.Region,
			cfg.AccessKey, cfg.SecretKey, cfg.Bucket, cfg.CloudDbRoot, "", "", true)
	}
	tilogs.L().Errorf("no available clouddb %v", cfg)
	return nil
}
