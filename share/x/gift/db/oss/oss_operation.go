package oss

import (
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/x/gift/config"
	"github.com/nghichtu91/platform/share/x/gift/model/errs"
)

// UploadToOSS 以全量更新的方式向OSS数据库上传指定批次和组号的Json格式码集。
func UploadToOSS(codesJson []byte, batchID, groupID int) (error, string) {
	tilogs.L().Debugf("UploadToOSS Bucket: %v", config.Cfg.GiftCfg.OSSDataBucket)
	if err := ossDB.PutWithBucket(config.Cfg.GiftCfg.OSSCloudRoot, GetBatchGroupOSSPath(batchID, groupID), codesJson, config.Cfg.GiftCfg.OSSDataBucket); err != nil {
		return err, errs.ErrDBOSSUpload
	}
	return nil, errs.Success
}

// DownloadFromOSS 从OSS数据库下载指定批次和组号的Json格式码集。
func DownloadFromOSS(batchID, groupID int) ([]byte, error, string) {
	key := GetBatchGroupOSSPath(batchID, groupID)
	tilogs.L().Debugf("DownloadFromOSS GetBatchGroupOSSPath key: %v", key)
	if content, err := ossDB.Get(key); err != nil {
		return nil, err, errs.ErrDBOSSDownload
	} else {
		return content, nil, errs.Success
	}
}

func GetBatchGroupOSSPath(batchID, groupID int) string {
	return fmt.Sprintf("%v/%v_%v_%v", config.PathCodesName, config.FileCodesNamePrefix, batchID, groupID)
}
