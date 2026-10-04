package models

import "github.com/nghichtu91/platform/share/planx/servers/db"

func QueryUserSpecialAward(uid db.UserID) (string, error) {
	return db_interface.QueryUserSpecialAward(uid)
}
func UpdateUserSpecialAward(uid db.UserID, awardStr string) error {
	return db_interface.UpdateUserSpecialAward(uid, awardStr)
}
