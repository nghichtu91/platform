package oss

import (
	"github.com/nghichtu91/platform/share/planx/cloud_db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/chat/web/config"
)

var GMGetCloudDB cloud_db.CloudDb

func InitCloudDB() bool {
	GMGetCloudDB = cloud_db.NewCloudDB(config.Cfg.GetCloudDBConfig())
	if GMGetCloudDB == nil {
		return false
	}
	if err := GMGetCloudDB.Open(); err != nil {
		tilogs.L().Errorf("SaveForCheck.Open err %s", err.Error())
		return false
	}
	return true
}

func StopCloudDB() {
	GMGetCloudDB.Close()
}
