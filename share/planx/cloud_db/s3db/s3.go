package s3db

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cenk/backoff"

	"github.com/nghichtu91/platform/share/planx/awshelper"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type BucketCannedACL string

const (
	Private           BucketCannedACL = "private"
	PublicRead        BucketCannedACL = "public-read"
	PublicReadWrite   BucketCannedACL = "public-read-write"
	AuthenticatedRead BucketCannedACL = "authenticated-read"
)

type S3 struct {
	s           *s3.S3
	bucket      string
	cloudDbRoot string
	region      string
	accessKey   string
	secretKey   string

	count_all int
	count_err int

	time_log time.Time

	format string
	seq    string

	needEndPoint bool
}

func NewStoreS3(region, accessKey, secretKey, bucket, cloudDbRoot, format, seq string, needEndPoint bool) *S3 {
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}

	myS3 := &S3{
		bucket:       bucket,
		cloudDbRoot:  cloudDbRoot,
		region:       region,
		accessKey:    accessKey,
		secretKey:    secretKey,
		time_log:     time.Now(),
		format:       format,
		seq:          seq,
		needEndPoint: needEndPoint,
	}

	return myS3
}

func (s *S3) PutWithBucket(cloudDbRoot, key string, val []byte, bucket string) error {
	if s.count_all >= 100 {
		s.LogInfo()
	}

	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	_, err := s.s.PutObject(&s3.PutObjectInput{
		ACL:         aws.String(string(Private)),
		Bucket:      aws.String(bucket),
		Body:        bytes.NewReader(val),
		ContentType: aws.String("application/octet-stream"),
		Key:         aws.String(key),
	})

	s.count_all++
	if err == nil {
		return nil
	}
	s.count_err++

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 100 * time.Millisecond
	b.MaxElapsedTime = 1 * time.Minute
	// 以下取默认值
	// DefaultMaxInterval         = 60 * time.Second
	// DefaultMaxElapsedTime      = 15 * time.Minute
	ticker := backoff.NewTicker(b)
	defer ticker.Stop()

	for range ticker.C {
		_, err := s.s.PutObject(&s3.PutObjectInput{
			ACL:         aws.String(string(Private)),
			Bucket:      aws.String(s.bucket),
			Body:        bytes.NewReader(val),
			ContentType: aws.String("text/plain"),
			Key:         aws.String(key),
		})

		if err == nil {
			break
		}
		//tilogs.L().Warnf("re put %s %v", key, k)
	}
	return nil
}

func (s *S3) Open() error {
	var mySession *session.Session
	if !s.needEndPoint {
		mySession = awshelper.CreateAWSSession(s.region, s.accessKey, s.secretKey, 3)
	} else {
		mySession = awshelper.CreateAWSSessionWithEndPoint(s.region, s.accessKey, s.secretKey, 3)
	}
	s.s = s3.New(mySession)
	return nil
}

func (s *S3) Close() error {
	return nil
}

func (s *S3) Clone() (*S3, error) {
	n := NewStoreS3(
		s.region,
		s.accessKey,
		s.secretKey,
		s.bucket,
		s.cloudDbRoot,
		s.format,
		s.seq, s.needEndPoint)
	if err := n.Open(); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *S3) LogInfo() {
	n := time.Now()
	in := n.Sub(s.time_log)

	tilogs.L().Infof("put info %d / %d by %d",
		s.count_err, s.count_all, in.Seconds())
	s.count_all = 0
	s.count_err = 0
	s.time_log = n
}

func (s *S3) Put(key string, val []byte) error {
	return s.PutWithBucket(s.cloudDbRoot, key, val, s.bucket)
}

func (s *S3) CreateSignedUrl(_, _ string, _ int) (error, string) {
	return fmt.Errorf("s3 not CreateSignedUrl"), ""
}

func (s *S3) PutWithMeta(key string, val []byte, metaKey string, metaVal string) error {
	key = s.cloudDbRoot + "/" + key
	_imp := func() error {
		_, err := s.s.PutObject(&s3.PutObjectInput{
			ACL:         aws.String(string(Private)),
			Bucket:      aws.String(s.bucket),
			Body:        bytes.NewReader(val),
			Key:         aws.String(key),
			ContentType: aws.String("text/plain"),
			Metadata: map[string]*string{
				metaKey: aws.String(metaVal),
			},
		})
		return err
	}

	err := _imp()
	s.count_all++
	if err == nil {
		return nil
	}

	s.count_err++

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 100 * time.Millisecond
	b.MaxElapsedTime = 1 * time.Minute
	// 以下取默认值
	// DefaultMaxInterval         = 60 * time.Second
	// DefaultMaxElapsedTime      = 15 * time.Minute
	ticker := backoff.NewTicker(b)
	defer ticker.Stop()
	for range ticker.C {
		err = _imp()

		if err == nil {
			break
		}
		//tilogs.L().Warnf("re put %s %v", key, k)
	}

	return err
}

func (s *S3) Get(key string) ([]byte, error) {
	return s.GetWithBucket(s.bucket, s.cloudDbRoot, key)
}

func (s *S3) GetWithBucket(bucket, cloudDbRoot, key string) ([]byte, error) {
	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	out, err := s.s.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer func(body io.ReadCloser) { _ = body.Close() }(out.Body)

	b, _ := ioutil.ReadAll(out.Body)

	return b, err
}

func (s *S3) GetMeta(key, metaKey string) (*string, error) {
	key = s.cloudDbRoot + "/" + key
	out, err := s.s.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	res, ok := out.Metadata[metaKey]
	if !ok {
		return nil, fmt.Errorf("meta %v not exist", metaKey)
	}
	return res, nil
}

func (s *S3) Del(key string) error {
	return s.DelWithBucket(s.bucket, s.cloudDbRoot, key)
}

func (s *S3) DelWithBucket(bucketName, cloudDbRoot, key string) error {
	if cloudDbRoot != "" {
		key = cloudDbRoot + "/" + key
	}
	_, err := s.s.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})

	return err
}

// Copy 从oldKey拷贝到newKey
// TODO 未测试接口
func (s *S3) Copy(oldKey string, newKey string) error {
	return s.CopyWithBucket(s.bucket, s.cloudDbRoot, oldKey, newKey)
}

// CopyWithBucket 指定bucket和cloudDbRoot, 从oldKey拷贝到newKey,
func (s *S3) CopyWithBucket(bucketName, cloudDBRoot, oldKey string, newKey string) error {
	if cloudDBRoot != "" {
		oldKey = cloudDBRoot + "/" + oldKey
		newKey = cloudDBRoot + "/" + newKey
	}
	_, err := s.s.CopyObject(&s3.CopyObjectInput{
		Bucket:     aws.String(bucketName),
		CopySource: aws.String(oldKey),
		Key:        aws.String(newKey),
	})
	return err
}

func (s *S3) ListObject(lastIdx string, scanLen int64) ([]string, string, error) {
	return s.ListObjectWithBucket(s.bucket, lastIdx, scanLen, s.cloudDbRoot, "")
}

func (s *S3) ListObjectWithBucketRecent(bucket, lastIdx string, scanLen, recentDay int64, cloudDbRoot, prefix string) ([]string, string, error) {
	if cloudDbRoot != "" {
		prefix = cloudDbRoot + "/" + prefix
	}
	params := &s3.ListObjectsInput{
		Bucket: aws.String(bucket), // Required
		//Delimiter:    aws.String("Delimiter"),
		//EncodingType: aws.String("EncodingType"),
		Marker:  aws.String(lastIdx),
		MaxKeys: aws.Int64(scanLen),
		Prefix:  aws.String(prefix),
	}

	resp, err := s.s.ListObjects(params)
	if err != nil {
		// A non-service error occurred.
		return []string{}, "", err
	}

	keys := make([]string, 0, len(resp.Contents))

	for _, v := range resp.Contents {
		if (timeutil.Now().Unix()-v.LastModified.Unix())/timeutil.DaySec < recentDay {
			keys = append(keys, aws.StringValue(v.Key))
		}
	}
	next := ""
	// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjects.html
	if aws.BoolValue(resp.IsTruncated) {
		next = aws.StringValue(resp.NextMarker)
		if next == "" && len(keys) > 0 {
			next = keys[len(keys)-1]
		}
	}
	return keys, next, err
}

func (s *S3) ListObjectWithBucket(bucket, lastIdx string, scanLen int64, cloudDbRoot, prefix string) (
	[]string,
	string,
	error,
) {
	if cloudDbRoot != "" {
		prefix = cloudDbRoot + "/" + prefix
	}
	params := &s3.ListObjectsInput{
		Bucket: aws.String(bucket), // Required
		//Delimiter:    aws.String("Delimiter"),
		//EncodingType: aws.String("EncodingType"),
		Marker:  aws.String(lastIdx),
		MaxKeys: aws.Int64(scanLen),
		Prefix:  aws.String(prefix),
	}

	resp, err := s.s.ListObjects(params)
	if err != nil {
		// A non-service error occurred.
		return []string{}, "", err
	}

	keys := make([]string, 0, len(resp.Contents))

	for _, v := range resp.Contents {
		keys = append(keys, aws.StringValue(v.Key))
	}
	next := ""
	// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjects.html
	if aws.BoolValue(resp.IsTruncated) {
		next = aws.StringValue(resp.NextMarker)
		if next == "" && len(keys) > 0 {
			next = keys[len(keys)-1]
		}
	}
	return keys, next, err
}
