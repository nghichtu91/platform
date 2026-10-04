package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

// GateInfo infomation structure...
type GateInfo struct {
	GID    uint
	GateID string
	CCU    int64
	// GameIPAddrPort 通常是公网ip:port
	GameIPAddrPort string
	// HostPort 公网域名:port
	HostPort string
	// RPCIPAddrPort 通常是内网ip:port !千万不要开放成公网IP!
	RPCIPAddrPort string
}

// GetConnectAddr 获取连接地址, 如果requireHost为true, 则返回host, 否则返回ip
func (g *GateInfo) GetConnectAddr(requireHost bool) string {
	if requireHost && g.HostPort != "" {
		return g.HostPort
	}
	return g.GameIPAddrPort
}

const gateZSetKey string = "gates:%s"    // gateid shard id
const gateIPExpKey string = "gate:%d:%s" // gateid, ip
const gateInfoKey string = "gateinfo:%s" // ip
func makeGateInfoKey(addrPort string) string {
	return fmt.Sprintf(gateInfoKey, addrPort)
}

func CleanGate(gate *GateInfo, reason string) error {
	GateMgr.delGate(fmt.Sprintf("%d", gate.GID), gate.GateID)
	tilogs.L().Infof("clean gate reason %s", reason)
	return nil
}

// GetOneGate 通常情况下，找出方式应该是找出CCU最少的那个Gate给玩家使用。
// 在下线某服务器的情况下，可以使得指定Gate上的用户越来越少，实现Gate服务器下线的平滑运维。
func GetOneGate(gid, sid uint) (gate *GateInfo, err error) {
	g, ok := GateMgr.GetOneGate(strconv.Itoa(int(gid)))
	if !ok {
		return nil, XErrGetGateNotExist
	}
	// JWS2-42263 改为使用服务发现判断gamex是否可用
	// keyInternalIp := consts.GetGameInternalIp4Auth(config.Cfg.CommonCfg.EtcdServer, gid, sid)
	// _, err = etcd.Get(keyInternalIp)
	// if err != nil {
	// 	return nil, fmt.Errorf("models.login.GetOneGate get internalip from etcd err %v", err)
	// }
	tilogs.L().Infof("IsGamexAliveGetKey: %s", etcd_ser_discovery.IsGamexAliveGetKey(config.Cfg.CommonCfg.EtcdServer, gid, sid) )
	
	if !etcd_ser_discovery.IsGamexAlive(config.Cfg.CommonCfg.EtcdServer, gid, sid) {
		return nil, fmt.Errorf("models.login.GetOneGate gamex not alive")
	}

	return &GateInfo{
		GID:            gid,
		GateID:         g.gateId,
		CCU:            g.ccu,
		GameIPAddrPort: g.addr,
		RPCIPAddrPort:  g.rpcAddr,
		HostPort:       g.host,
	}, nil
}

type LoginStatueCode int

const (
	LSC_GETGATE LoginStatueCode = 1 + iota
	LSC_LOGIN
	LSC_LOGOFF
)

const (
	LogInExpireTime  = 7776000 // 90天
	LogOffExpireTime = 86400   // 1天
)

type loginStatus struct {
	Sid           uint
	RPCGateIPAddr string
	LoginToken    string
	AccountID     string
	Timestamp     int64
	Status        int
}

func makeLoginStatusKey(gid uint, uid db.UserID) string {
	return fmt.Sprintf("ls:%d:%s", gid, uid)
}

func GetLastLoginStatus(gid uint, uid db.UserID) (
	OldAccountID, OldToken, OldRPC string, HasOldValue bool) {

	db := loginRedisPool.Get()
	defer db.Close()

	rk_game_login_status := makeLoginStatusKey(gid, uid)
	oldvalue, err := redis.Bytes(db.Do(allmetrics.GetAuthDBStatPrefix("LoginStatus", "GET"), "GET", rk_game_login_status))
	// 刚注册的玩家肯定没有上次登录信息，如果没找到key，也忽略
	if err != nil && err != redis.ErrNil {
		tilogs.L().Warnf("models.GetLastLoginStatus err: %v %s", err.Error(), rk_game_login_status)
		return
	}

	if oldvalue == nil {
		HasOldValue = false
		return
	}
	var oldls loginStatus
	err = json.Unmarshal(oldvalue, &oldls)
	if err != nil {
		tilogs.L().Warnf("models.GetLastLoginStatus unmarshal oldvalue failed: ", err)
		return
	} else {
		// TODO 根据旧值，需要踢出前面登录的玩家
		OldToken = oldls.LoginToken
		OldRPC = oldls.RPCGateIPAddr
		OldAccountID = oldls.AccountID
		HasOldValue = true
		tilogs.L().Debugf("models.GetLastLoginStatus old status should logout, gid: %v, uid:%s", gid, uid.String())
	}
	return
}

// DeleteLoginStatus 通常因为某种原因，需要主动删除错误信息
func DeleteLoginStatus(gid uint, uid db.UserID) {
	db := loginRedisPool.Get()
	defer db.Close()

	rk_game_login_status := makeLoginStatusKey(gid, uid)

	tilogs.L().Debugf("LoginStatus del %s", rk_game_login_status)
	if _, err := db.Do(allmetrics.GetAuthDBStatPrefix("LoginStatus", "DEL"), "DEL", rk_game_login_status); err != nil {
		tilogs.L().Warnf("models.DeleteLoginStatus DEL key error: ", rk_game_login_status, err.Error())
	}
}

// UpdateLoginStatus 用于控制玩家在一个gameid下，同时只能玩一个帐号
// 使用gid:uid为唯一标识，存储上次玩家的登录信息
func UpdateLoginStatus(account db.Account, gateRPCAddr, loginToken string, status LoginStatueCode) {
	db := loginRedisPool.Get()
	defer db.Close()

	rk_game_login_status := makeLoginStatusKey(account.GameId, account.UserId)

	var exTime int
	switch status {
	case LSC_LOGIN:
		exTime = LogInExpireTime
	case LSC_LOGOFF:
		exTime = LogOffExpireTime
	}

	ls := loginStatus{
		Sid:           account.ShardId,
		RPCGateIPAddr: gateRPCAddr,
		LoginToken:    loginToken,
		AccountID:     account.String(),
		Status:        int(status),
		Timestamp:     time.Now().Unix(),
	}
	if value, err := json.Marshal(&ls); err != nil {
		tilogs.L().Errorf("models.updateloginstatus err json Marshal with:", err, ls)
	} else {
		// 如果玩家使用模拟器，很有可能两次维护期间都不会下线，原来的24h过期时间会导致踢人功能失效
		// 为了防止这种情况的发生，将在线状态的过期时间改为90天，基本上应该能杜绝无法踢人的情况
		tilogs.L().Debugf("UpdateLoginStatus SETEX %s", rk_game_login_status)
		_, err := db.Do(allmetrics.GetAuthDBStatPrefix("LoginStatus", "SETEX"), "SETEX", rk_game_login_status, exTime, value)
		if err != nil {
			tilogs.L().Errorf("models.updateloginstatus err with db with :", err)
		}
	}
	return
}
