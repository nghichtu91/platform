package ossdb

import (
	"bytes"
	"fmt"
	"io"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type OSS struct {
	client *oss.Client
	cfg    OSSCfg
	bucket *oss.Bucket
}

func (s *OSS) filePath(path string) string {
	if s.cfg.cloudDbRoot == "" {
		return path
	}
	return s.cfg.cloudDbRoot + "/" + path
}

type OSSCfg struct {
	endPoint    string
	bucketName  string
	cloudDbRoot string
	format      string
	seq         string
	accessKey   string
	secretKey   string
}

func NewStoreOSS(endPoint, accessKey, secretKey, bucket, cloudDbRoot, format, seq string) *OSS {
	s := &OSS{
		cfg: OSSCfg{
			bucketName:  bucket,
			cloudDbRoot: cloudDbRoot,
			format:      format,
			seq:         seq,
		},
	}
	var err error
	s.client, err = oss.New(endPoint, accessKey, secretKey)
	if err != nil {
		tilogs.L().Errorf("<StoreOss> new OOS err %v", err)
		return nil
	}
	return s
}

func (s *OSS) Open() error {
	var err error
	// 桶大概率是已存在的, 这里不做判断
	_ = s.client.CreateBucket(s.cfg.bucketName)
	// 获取桶
	s.bucket, err = s.client.Bucket(s.cfg.bucketName)
	if err != nil {
		tilogs.L().Errorf("<StoreOss> new OOS Bucket err %v", err)
	}
	return err
}

func (s *OSS) CreateSignedUrl(_, _ string, _ int) (error, string) {
	return fmt.Errorf("OSS not CreateSignedUrl"), ""
}

func (s *OSS) PutWithBucket(cloudDbRoot, key string, val []byte, bucketName string) error {
	bucket, err := s.client.Bucket(bucketName)
	if err != nil {
		return err
	}

	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	err = bucket.PutObject(key, bytes.NewReader(val))
	if err != nil {
		return err
	}
	return nil
}

func (s *OSS) Close() error {
	return nil
}

func (s *OSS) Put(key string, val []byte) error {
	if s.bucket == nil {
		return fmt.Errorf("oss bucket is nil")
	}
	key = s.filePath(key)
	err := s.bucket.PutObject(key, bytes.NewReader(val))
	if err != nil {
		return err
	}
	// TODO 失败重试
	return nil
}

func (s *OSS) Get(key string) ([]byte, error) {
	if s.bucket == nil {
		return nil, fmt.Errorf("oss bucket is nil")
	}
	key = s.filePath(key)
	body, err := s.bucket.GetObject(key)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, body)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *OSS) Del(key string) error {
	if s.bucket == nil {
		return nil
	}
	key = s.filePath(key)
	err := s.bucket.DeleteObject(key)
	if err != nil {
		return err
	}
	return nil
}

func (s *OSS) DelWithBucket(bucketName, cloudDbRoot, key string) error {
	var err error
	bucket, err := s.client.Bucket(bucketName)
	if err != nil {
		tilogs.L().Errorf("<OSS> GetWithBucket bucket err", err)
		return err
	}
	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	return bucket.DeleteObject(key)
}

// Copy 复制文件
func (s *OSS) Copy(oldKey, newKey string) error {
	var err error

	cloudDbRoot := s.cfg.cloudDbRoot
	if cloudDbRoot != "" {
		oldKey = cloudDbRoot + "/" + oldKey
		newKey = cloudDbRoot + "/" + newKey
	}
	_, err = s.bucket.CopyObject(oldKey, newKey)
	return err
}

// CopyWithBucket 复制文件
func (s *OSS) CopyWithBucket(bucketName, cloudDbRoot, oldKey, newKey string) error {
	var err error
	var bucket *oss.Bucket
	bucket, err = s.client.Bucket(bucketName)
	if err != nil {
		tilogs.L().Errorf("<OSS> GetWithBucket bucket err", err)
		return err
	}
	if cloudDbRoot != "" {
		oldKey = cloudDbRoot + "/" + oldKey
		newKey = cloudDbRoot + "/" + newKey
	}
	_, err = bucket.CopyObject(oldKey, newKey)
	return err
}

func (s *OSS) Clone() (*OSS, error) {
	clone := NewStoreOSS(s.cfg.endPoint, s.cfg.accessKey, s.cfg.secretKey,
		s.cfg.bucketName, s.cfg.cloudDbRoot, s.cfg.format, s.cfg.seq)
	if err := clone.Open(); err != nil {
		return nil, err
	}
	return clone, nil
}

func (s *OSS) GetWithBucket(bucketName, cloudDbRoot, key string) ([]byte, error) {
	var err error
	bucket, err := s.client.Bucket(bucketName)
	if err != nil {
		tilogs.L().Errorf("<OSS> GetWithBucket bucket err", err)
		return nil, err
	}
	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	body, err := bucket.GetObject(key)
	if err != nil {
		tilogs.L().Errorf("<OSS> GetWithBucket object err", err)
		return nil, err
	}
	defer body.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, body)
	if err != nil {
		tilogs.L().Errorf("<OSS> GetWithBucket copy err", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *OSS) ListObject(lastIdx string, scanLen int64) ([]string, string, error) {
	return s.ListObjectWithBucket(s.cfg.bucketName, lastIdx, scanLen, s.cfg.cloudDbRoot, "")
}

func (s *OSS) ListObjectWithBucketRecent(bucketName, lastIdx string, scanLen, recentDay int64, cloudDbRoot, prefix string) (
	[]string,
	string,
	error,
) {
	bucket, err := s.client.Bucket(bucketName)
	if err != nil {
		// A non-service error occurred.
		return []string{}, "", err
	}
	// 此处参数的使用是参照S3使用的
	if cloudDbRoot != "" {
		prefix = cloudDbRoot + "/" + prefix
	}
	resp, err := bucket.ListObjects(oss.MaxKeys(int(scanLen)), oss.Prefix(prefix),
		oss.Marker(lastIdx))
	if err != nil {
		// A non-service error occurred.
		return []string{}, "", err
	}

	keys := make([]string, 0, len(resp.Objects))
	now := timeutil.Now().Unix()
	for _, v := range resp.Objects {
		if (now-v.LastModified.Unix())/timeutil.DaySec < recentDay {
			keys = append(keys, v.Key)
		}
	}

	next := ""
	// https://help.aliyun.com/document_detail/31965.html
	if resp.IsTruncated {
		next = resp.NextMarker
	}
	return keys, next, err
}

func (s *OSS) ListObjectWithBucket(bucketName, lastIdx string, scanLen int64, cloudDbRoot, prefix string) (
	[]string,
	string,
	error,
) {
	bucket, err := s.client.Bucket(bucketName)
	if err != nil {
		// A non-service error occurred.
		return []string{}, "", err
	}
	// 此处参数的使用是参照S3使用的
	if cloudDbRoot != "" {
		prefix = cloudDbRoot + "/" + prefix
	}
	resp, err := bucket.ListObjects(oss.MaxKeys(int(scanLen)), oss.Prefix(prefix),
		oss.Marker(lastIdx))
	if err != nil {
		// A non-service error occurred.
		return []string{}, "", err
	}

	keys := make([]string, 0, len(resp.Objects))
	for _, v := range resp.Objects {
		keys = append(keys, v.Key)
	}

	next := ""
	// https://help.aliyun.com/document_detail/31965.html
	if resp.IsTruncated {
		next = resp.NextMarker
	}
	return keys, next, err
}

func (s *OSS) GetMeta(key, metaKey string) (*string, error) {
	key = s.filePath(key)
	props, err := s.bucket.GetObjectDetailedMeta(key)

	if err != nil {
		return nil, err
	}

	res := props.Get(metaKey)

	return &res, nil
}

func (s *OSS) PutWithMeta(key string, val []byte, metaKey string, metaVal string) error {
	key = s.filePath(key)
	err := s.bucket.PutObject(key, bytes.NewReader(val))
	if err != nil {
		return err
	}

	// 场景：设置Bucket Meta，可以设置一个或多个属性。
	// 注意：Meta不区分大小写
	options := []oss.Option{
		oss.Meta(metaKey, metaVal)}
	return s.bucket.SetObjectMeta(key, options...)
}
