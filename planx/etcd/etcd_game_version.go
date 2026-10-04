package etcd

import (
	"encoding/json"
	"errors"
	"fmt"
)

type BundleInfo struct {
	BundleVer        string `json:"version"`
	ForceUpdate      bool   `json:"force_update"`
	ClientExpireTime int64  `json:"clientExpireTime"`
}

// 获取ETCD中旧的Ver版本信息
func GetBundleVerInfo(bundleVerkey string) (map[string]BundleInfo, error) {
	info, _, err := GetBundleVerInfoWithRev(bundleVerkey)
	return info, err
}

// GetBundleVerInfoWithRev 获取ETCD中旧的Ver版本信息和rev
func GetBundleVerInfoWithRev(bundleVerkey string) (map[string]BundleInfo, int64, error) {
	infoJson, rev, getErr := GetWithRev(bundleVerkey)
	if getErr != nil && !errors.Is(getErr, EmptyKvErr) {
		return nil, rev, fmt.Errorf("<AddGameVersionRequest> etcd get error: %v", getErr)
	}

	bundleVerInfo, unmarshalErr := UnmarshalBundleVerInfo(infoJson)
	if unmarshalErr != nil {
		return nil, rev, fmt.Errorf("<AddGameVersionRequest> json unmarshal error: %v", unmarshalErr)
	}

	return bundleVerInfo, rev, nil
}

// 解析获取的Ver版本信息
func UnmarshalBundleVerInfo(infoJson string) (map[string]BundleInfo, error) {
	bundleVerInfo := make(map[string]BundleInfo) // map[Version][BundleVer]
	if infoJson != "" {
		err := json.Unmarshal([]byte(infoJson), &bundleVerInfo)
		if err != nil {
			return nil, err
		}
	}
	return bundleVerInfo, nil
}

// 写入ETCD中新的Ver版本信息
func PutBundleVerInfo(bundleVerkey string, bundleVerInfo map[string]BundleInfo) error {
	bundleVerJson, _ := json.Marshal(bundleVerInfo)
	err := Put(bundleVerkey, string(bundleVerJson))
	if err != nil {
		return fmt.Errorf("<AddGameVersionRequest> etcd put error: %v", err)
	}

	return nil
}
