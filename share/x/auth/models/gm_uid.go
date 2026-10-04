package models

func InsertOrUpdateSdkIdToUid(sdkIdToUidInfos *SdkIdToUidInfos) ([]*SdkIdToUidInfos, error) {

	return db_interface.InsertOrUpdateSdkIdToUid(sdkIdToUidInfos)

}

func DeleteSdkIdToUid(numberId string, shard uint) ([]*SdkIdToUidInfos, error) {

	return db_interface.DeleteSdkIdToUid(numberId, shard)
}

func QuerySdkIdToUid() ([]*SdkIdToUidInfos, error) {

	return db_interface.QuerySdkIdToUid()
}
