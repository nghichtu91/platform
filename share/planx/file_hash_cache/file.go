package file_hash_cache

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// GenHash 生成对应文件的hash string
// 文件越大，生成hash的时间越长
func GenHash(fn string) (string, error) {
	f, err := os.Open(fn)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := md5.New()
	if _, err = io.Copy(hash, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
