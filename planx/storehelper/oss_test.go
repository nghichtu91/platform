package storehelper

import (
	"testing"
)

var (
	OSS_Bucket    = "jws2-battle-data"
	OSS_EndPoint  = "http://oss-cn-hangzhou.aliyuncs.com"
	OSS_AccessKey = ""
	OSS_SecretKey = ""
)

func TestStoreOSS(t *testing.T) {
	ossStore := NewStore(OSS, OSSInitCfg{
		EndPoint:  OSS_EndPoint,
		AccessId:  OSS_AccessKey,
		AccessKey: OSS_SecretKey,
		Bucket:    OSS_Bucket,
	})
	ossStore.Open()
	key := "compare_test1_test2"
	err := ossStore.Put(key, []byte("北国风光，千里冰封，万里雪飘"), nil)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

}
