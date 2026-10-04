package etcd

import (
	"strconv"
	"strings"
)

// GetKeyLastName
// 获取etcd key用/分割后，最后一段的字符串
// 如果字符串最后一位是'/'会直接删除
// 譬如 server_root/gid/switch/devdebug/crossx 返回 "crossx"
// server_root/gid/switch/devdebug/ 返回 "devdebug"
// server_root 返回 "server_root"
func GetKeyLastName(key string) string {
	if key == "" {
		return ""
	}

	if key[len(key)-1] == '/' {
		key = key[:len(key)-1]
	}

	idx := strings.LastIndex(key, "/")
	if idx == -1 {
		return key
	}

	return key[idx+1:]
}

// GetKeyLastNameAtoI
// 获取etcd key用/分割后，最后一段的字符串的int形势
// 譬如 server_root/gid/switch/cross/100 返回 100
func GetKeyLastNameAtoI(key string) (int, error) {
	return strconv.Atoi(GetKeyLastName(key))
}
