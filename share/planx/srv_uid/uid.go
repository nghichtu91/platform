package srv_uid

import (
	"github.com/nghichtu91/platform/share/planx/etcd"
	"github.com/nghichtu91/platform/share/planx/util"
)

// 服务uint16 uid定义
// sonyflake 使用uint16作为MachineID，确保生成的uint64 UID唯一
// 因此需要在0-65535之间，给每类服务划定区间

// gamex：0-9999 // 预留，gamex modules里已经使用了
// scene: 10000-19999
// battlex: 数量较多，预留5000应该够了
// xscene: 可能也不少，1000吧
// gmserver: 只能有1个
// crossx: 预计最多100个
// stressx: 预计最多100个
// gamex cache_client使用：30000-39999
// 后面的服务用到再说吧

const (
	_               = iota
	UIDSceneBegin   = 10000
	UIDBattlexBegin = 20000
	UIDXSceneBegin  = 25000
	UIDGMServer     = 26000
	UIDCrossxBegin  = 26001
	UIDStressxBegin = 26101
	UIDGamexBegin   = 30000
)

const (
	cutNum = 10000
)

// GetSrvUID 从ServerID生成对应的uint16 UID
// 如果服务器末尾带数字，会自动转成对应的服务器序号，譬如"cross0", "xscene1600001"
// 超过10000的部分会被截掉，因为gamex最大数量限制每个大区为10000，其他服务不能超过gamex的数量
func GetSrvUID(srvTyp, srvID string) uint16 {
	if len(srvID) == 0 {
		return 0
	}

	num := uint16(util.SimpleGetUintFromString(srvID) % cutNum)

	switch srvTyp {
	case etcd.Server_Gamex:
		return UIDGamexBegin + num
	case etcd.Server_Scene:
		return UIDSceneBegin + num
	case etcd.Server_Battle:
		return UIDBattlexBegin + num
	case etcd.ServerXScene:
		return UIDXSceneBegin + num
	case etcd.ServerSer_GMServer:
		return UIDGMServer
	case etcd.Server_Crossx:
		return UIDCrossxBegin + num
	case etcd.Server_Stressx:
		return UIDStressxBegin + num
	default:
		return 0
	}
}
