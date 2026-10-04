package models

import (
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/x/auth/config_data"
)

func PayRewardForUser(rewardType int, uid db.UserID, opt string) (map[int]*config_data.RewardInfo, int32) {
	return db_interface.RewardForUser(rewardType, uid, opt)
}
