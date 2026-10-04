package feishu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"hash"
	"strconv"
	"time"
)

var (
	h hash.Hash //  hmac.New(sha256.New, nil)

	data []byte // 缓存 timestamp + \n + key

	lastTS   int64
	lastSign string
)

// timestamp + key 做sha256, 再进行base64 encode
// 每秒的sign都是一样的，这里采用缓存的方式
// 同时sender是串行发送，不需要考虑多线程问题
func getSign() string {
	if time.Now().Unix() == lastTS {
		return lastSign
	}

	// 更新时间戳
	lastTS = time.Now().Unix()
	copy(data, strconv.Itoa(int(lastTS)))

	// 生成新签名
	h = hmac.New(sha256.New, data)

	lastSign = base64.StdEncoding.EncodeToString(h.Sum(nil))

	return lastSign
}
