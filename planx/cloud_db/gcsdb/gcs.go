package gcsdb

import (
	"context"
	"fmt"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"io/ioutil"
	"time"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/storage"
)

type gcs struct {
	cfg    gcsCfg
	client *storage.Client
}

type gcsCfg struct {
	bucketName  string
	cloudDbRoot string
}

func NewStoreGCS(bucket, cloudDbRoot string) *gcs {
	return &gcs{
		cfg: gcsCfg{
			bucketName:  bucket,
			cloudDbRoot: cloudDbRoot,
		},
	}
}

func (s *gcs) Open() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	s.client = client
	return nil
}
func (s *gcs) Close() error {
	if s.client != nil {
		if err := s.client.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (s *gcs) Put(key string, val []byte) error {
	return s.PutWithBucket(s.cfg.cloudDbRoot, key, val, s.cfg.bucketName)
}

func (s *gcs) CreateSignedUrl(_, _ string, _ int) (error, string) {
	return fmt.Errorf("gcs not CreateSignedUrl"), ""
}

func (s *gcs) PutWithBucket(cloudDbRoot, key string, val []byte, bucket string) error {
	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	wc := s.client.Bucket(bucket).Object(key).NewWriter(ctx)
	defer func(wc *storage.Writer) { _ = wc.Close() }(wc)

	_, err := wc.Write(val)
	if err != nil {
		return err
	}
	return nil
}

func (s *gcs) PutWithMeta(key string, val []byte, metaKey string, metaVal string) error {
	err := s.Put(key, val) // Put内部会拼接cloudDbRoot, 这里不能再拼接
	if err != nil {
		return err
	}
	if s.cfg.cloudDbRoot != "" {
		key = s.cfg.cloudDbRoot + "/" + key
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	o := s.client.Bucket(s.cfg.bucketName).Object(key)
	objectAttrsToUpdate := storage.ObjectAttrsToUpdate{
		Metadata: map[string]string{
			metaKey: metaVal,
		},
	}
	if _, err = o.Update(ctx, objectAttrsToUpdate); err != nil {
		return err
	}
	return nil
}

func (s *gcs) Get(key string) ([]byte, error) {
	return s.GetWithBucket(s.cfg.bucketName, s.cfg.cloudDbRoot, key)
}

func (s *gcs) GetWithBucket(bucket, cloudDbRoot, key string) ([]byte, error) {
	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	r, err := s.client.Bucket(bucket).Object(key).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("bucket %v, key: %v, err: %v", bucket, key, err)
	}
	defer func(r *storage.Reader) { _ = r.Close() }(r)
	return ioutil.ReadAll(r)
}

func (s *gcs) GetMeta(key, metaKey string) (*string, error) {
	if s.cfg.cloudDbRoot != "" {
		key = s.cfg.cloudDbRoot + "/" + key
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	o := s.client.Bucket(s.cfg.bucketName).Object(key)
	attrs, err := o.Attrs(ctx)
	if err != nil {
		return nil, err
	}
	metaValue, ok := attrs.Metadata[metaKey]
	if !ok {
		return nil, fmt.Errorf("meta %v not exist", metaKey)
	}
	return &metaValue, nil
}

func (s *gcs) Del(key string) error {
	return s.DelWithBucket(s.cfg.bucketName, s.cfg.cloudDbRoot, key)
}

func (s *gcs) DelWithBucket(bucket, cloudDbRoot, key string) error {
	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	o := s.client.Bucket(bucket).Object(key)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	return o.Delete(ctx)
}

// Copy 拷贝文件
// TODO 未测试
func (s *gcs) Copy(oldKey, newKey string) error {
	return s.CopyWithBucket(s.cfg.bucketName, s.cfg.cloudDbRoot, oldKey, newKey)
}

func (s *gcs) CopyWithBucket(bucketName, cloudDbRoot, oldKey, newKey string) error {
	if cloudDbRoot != "" {
		oldKey = cloudDbRoot + "/" + oldKey
		newKey = cloudDbRoot + "/" + newKey
	}
	bucket := s.client.Bucket(bucketName)
	oldObject := bucket.Object(oldKey)
	newObject := bucket.Object(newKey)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err := newObject.CopierFrom(oldObject).Run(ctx)
	return err
}

func (s *gcs) ListObject(lastIdx string, scanLen int64) ([]string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	it := s.client.Bucket(s.cfg.bucketName).Objects(ctx, nil)
	p := iterator.NewPager(it, int(scanLen), lastIdx)
	_ret := make([]*storage.ObjectAttrs, 0, scanLen)
	next, err := p.NextPage(&_ret)
	if err != nil {
		return nil, "", err
	}
	ret := make([]string, 0, 4)
	for _, d := range _ret {
		ret = append(ret, d.Name)
	}
	return ret, next, nil
}

func (s *gcs) ListObjectWithBucketRecent(bucket, lastIdx string, scanLen, recentDay int64, cloudDbRoot, prefix string) (
	[]string,
	string,
	error,
) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if cloudDbRoot != "" {
		prefix = cloudDbRoot + "/" + prefix
	}
	it := s.client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})
	p := iterator.NewPager(it, int(scanLen), lastIdx)
	_ret := make([]*storage.ObjectAttrs, 0, scanLen)
	nextPageToken, err := p.NextPage(&_ret)
	if err != nil {
		return nil, "", err
	}
	ret := make([]string, 0, 4)
	for _, d := range _ret {
		if (timeutil.Now().Unix()-d.Created.Unix())/timeutil.DaySec < recentDay {
			ret = append(ret, d.Name)
		}
	}
	return ret, nextPageToken, nil
}

func (s *gcs) ListObjectWithBucket(bucket, lastIdx string, scanLen int64, cloudDbRoot, prefix string) (
	[]string,
	string,
	error,
) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if cloudDbRoot != "" {
		prefix = cloudDbRoot + "/" + prefix
	}
	it := s.client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})
	p := iterator.NewPager(it, int(scanLen), lastIdx)
	_ret := make([]*storage.ObjectAttrs, 0, scanLen)
	nextPageToken, err := p.NextPage(&_ret)
	if err != nil {
		return nil, "", err
	}
	ret := make([]string, 0, 4)
	for _, d := range _ret {
		ret = append(ret, d.Name)
	}
	return ret, nextPageToken, nil
}
