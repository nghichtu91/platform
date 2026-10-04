package obsdb

import (
	"bytes"
	"fmt"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"io"
	"io/ioutil"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type OBS struct {
	client *obs.ObsClient
	cfg    OBSCfg
}

type OBSCfg struct {
	endPoint    string
	bucketName  string
	cloudDbRoot string
	format      string
	seq         string
	accessKey   string
	secretKey   string
}

func NewStoreOBS(endPoint, accessKey, secretKey, bucket, cloudDbRoot, format, seq string) *OBS {
	return &OBS{
		cfg: OBSCfg{
			endPoint:    endPoint,
			bucketName:  bucket,
			cloudDbRoot: cloudDbRoot,
			format:      format,
			seq:         seq,
			accessKey:   accessKey,
			secretKey:   secretKey,
		},
	}
}

func (s *OBS) Open() error {
	var err error
	s.client, err = obs.New(s.cfg.accessKey, s.cfg.secretKey, s.cfg.endPoint)
	if err != nil {
		tilogs.L().Errorf("<StoreObs> new OBS err %v", err)
		return nil
	}
	return nil
}

func (s *OBS) Close() error {
	return nil
}

func (s *OBS) Put(key string, val []byte) error {
	return s.PutWithBucket(s.cfg.cloudDbRoot, key, val, s.cfg.bucketName)
}

func (s *OBS) CreateSignedUrl(bucket, key string, expires int) (error, string) {
	input := &obs.CreateSignedUrlInput{}
	input.Method = obs.HttpMethodGet
	input.Bucket = bucket
	input.Key = key
	input.Expires = expires

	outPut, err := s.client.CreateSignedUrl(input)
	if err != nil {
		return err, ""
	}

	return nil, outPut.SignedUrl
}

func (s *OBS) PutWithBucket(cloudDbRoot, key string, val []byte, bucket string) error {
	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	input := &obs.PutObjectInput{}
	input.Bucket = bucket
	input.Key = key
	input.Body = bytes.NewReader(val)

	_, err := s.client.PutObject(input)
	return err
}
func (s *OBS) PutWithMeta(key string, val []byte, metaKey string, metaVal string) error {
	if s.cfg.cloudDbRoot != "" {
		key = s.cfg.cloudDbRoot + "/" + key
	}
	input := &obs.PutObjectInput{}
	input.Bucket = s.cfg.bucketName
	input.Key = key
	input.Body = bytes.NewReader(val)
	input.Metadata = map[string]string{metaKey: metaVal}

	_, err := s.client.PutObject(input)
	return err

}
func (s *OBS) Get(key string) ([]byte, error) {
	return s.GetWithBucket(s.cfg.bucketName, s.cfg.cloudDbRoot, key)
}

func (s *OBS) GetWithBucket(bucket, cloudDbRoot, key string) ([]byte, error) {
	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	input := &obs.GetObjectInput{}
	input.Bucket = bucket
	input.Key = key

	output, err := s.client.GetObject(input)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) { _ = body.Close() }(output.Body)
	return ioutil.ReadAll(output.Body)
}

func (s *OBS) GetMeta(key, metaKey string) (*string, error) {
	if s.cfg.cloudDbRoot != "" {
		key = s.cfg.cloudDbRoot + "/" + key
	}
	input := &obs.GetObjectInput{}
	input.Bucket = s.cfg.bucketName
	input.Key = key

	output, err := s.client.GetObject(input)
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) { _ = body.Close() }(output.Body)
	metaValue, ok := output.Metadata[metaKey]
	if !ok {
		return nil, fmt.Errorf("meta %v not exist", metaKey)
	}
	return &metaValue, nil
}

func (s *OBS) Del(key string) error {
	return s.DelWithBucket(s.cfg.bucketName, s.cfg.cloudDbRoot, key)
}

func (s *OBS) DelWithBucket(bucket, cloudDbRoot, key string) error {
	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	input := &obs.DeleteObjectInput{}
	input.Bucket = bucket
	input.Key = key

	_, err := s.client.DeleteObject(input)
	return err
}

// Copy 复制对象
func (s *OBS) Copy(oldKey string, newKey string) error {
	return s.CopyWithBucket(s.cfg.bucketName, s.cfg.cloudDbRoot, oldKey, newKey)
}

// CopyWithBucket 复制对象
func (s *OBS) CopyWithBucket(bucketName, cloudDBRoot, oldKey string, newKey string) error {
	if cloudDBRoot != "" {
		oldKey = cloudDBRoot + "/" + oldKey
		newKey = cloudDBRoot + "/" + newKey
	}
	input := &obs.CopyObjectInput{}
	input.Bucket = bucketName
	input.Key = newKey
	input.CopySourceBucket = bucketName
	input.CopySourceKey = oldKey

	_, err := s.client.CopyObject(input)
	return err
}

func (s *OBS) ListObject(lastIdx string, scanLen int64) ([]string, string, error) {
	return s.ListObjectWithBucket(s.cfg.bucketName, lastIdx, scanLen, s.cfg.cloudDbRoot, "")
}

func (s *OBS) ListObjectWithBucketRecent(bucket, lastIdx string, scanLen, recentDay int64, cloudDbRoot, prefix string) ([]string, string, error) {
	if cloudDbRoot != "" {
		prefix = cloudDbRoot + "/" + prefix
	}
	input := &obs.ListObjectsInput{}
	input.Bucket = bucket
	input.MaxKeys = int(scanLen)
	input.Marker = lastIdx
	input.Prefix = prefix

	output, err := s.client.ListObjects(input)
	if err != nil {
		return nil, "", err
	}

	ret := make([]string, 0, len(output.Contents))
	for _, v := range output.Contents {
		if (timeutil.Now().Unix()-v.LastModified.Unix())/timeutil.DaySec < recentDay {
			ret = append(ret, v.Key)
		}
	}
	// https://support.huaweicloud.com/api-obs/obs_04_0022.html
	next := ""
	if output.IsTruncated {
		next = output.NextMarker
	}
	return ret, next, err
}

func (s *OBS) ListObjectWithBucket(bucket, lastIdx string, scanLen int64, cloudDbRoot, prefix string) (
	[]string,
	string,
	error,
) {
	if cloudDbRoot != "" {
		prefix = cloudDbRoot + "/" + prefix
	}
	input := &obs.ListObjectsInput{}
	input.Bucket = bucket
	input.MaxKeys = int(scanLen)
	input.Marker = lastIdx
	input.Prefix = prefix

	output, err := s.client.ListObjects(input)
	if err != nil {
		return nil, "", err
	}

	ret := make([]string, 0, len(output.Contents))
	for _, v := range output.Contents {
		ret = append(ret, v.Key)
	}
	// https://support.huaweicloud.com/api-obs/obs_04_0022.html
	next := ""
	if output.IsTruncated {
		next = output.NextMarker
	}
	return ret, next, err
}
