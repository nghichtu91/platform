package oss

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/planx/cloud_db"
)

var ossDB cloud_db.CloudDb

// 初始化OSS数据库链接。
func InitDB(endPoint string, dataBucket, cloudDbRoot string, accessKey string, secretKey string) error {
	// 创建OSSDB链接对象。

	ossDB = cloud_db.NewCloudDB(cloud_db.CloudDbConfig{
		DbDriver:    cloud_db.CloudDb_Aliyun,
		Region:      endPoint,
		Bucket:      dataBucket,
		CloudDbRoot: cloudDbRoot,
		AccessKey:   accessKey,
		SecretKey:   secretKey,
	})

	if ossDB == nil {
		return fmt.Errorf("create OSSDB Failed, object is nil")
	}

	if err := ossDB.Open(); err != nil {
		return fmt.Errorf("open OSSDB Failed, err: %v", err)
	}

	return nil
}

// 关闭OSS链接对象。
func CloseDB() {
	if ossDB != nil {
		if err := ossDB.Close(); err != nil {
			tilogs.L().Errorf("ossDB.Close err: %v", err)
		}
	}
}

// 获取OSS链接对象。
func GetDB() cloud_db.CloudDb {
	return ossDB
}
