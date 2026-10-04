package gate

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/nghichtu91/platform/share/planx/net"

	"github.com/astaxie/beego/cache"

	"github.com/nghichtu91/platform/share/planx/client"
	"github.com/nghichtu91/platform/share/planx/safecache"
	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

const defaultTokenAliveDur = 0 // 原则上不过期, 否则可能会出现 session已过期, 但是玩家依旧在线的情况
const cachedToken = "CachedToken"

type sessionInfo struct {
	id         int64 // sessionId
	ip         string
	kickFunc   func(string, int, int, int, string, func())
	notifyFunc func(InfoNotifyToClient)
}

type loginInfo struct {
	sid uint
	// 保存login服务器玩家getGate后通知过来的login token
	// 这里的token都代表，玩家Maybe可能会来服务器
	loginCache cache.Cache // <loginToken, LoginNotify>

	// 玩家使用login token完成Handshake后，这里会有token
	mu         sync.RWMutex
	autoIncId  int64
	sessionMap map[string]*sessionInfo // <accountId, sessionInfo>

	// 存储活跃的sessionId
	activeSessionCache cache.Cache // <sessionId, struct>

	// 单服维护状态
	shardMaintaining bool
}

type LoginNotify struct {
	db.Account
	LoginToken         string
	NumberID           int64
	SdkDeviceId        string
	RechargeMoney      int32 // 充值返利-充值返利的累充积分。
	RechargeReturn     bool  // 充值返利-是否已领取够累充返利的奖励。
	LoginDay           int32 // 登陆排名返利-登陆天数。
	Fair1v1Dan         int32 // 登陆排名返利-1v1段位。
	Fair3v3Dan         int32 // 登陆排名返利-3v3段位。
	LoginAndRankReward bool  // 登陆排名返利-是否已领。
	ClaimTime          int64 // 登陆排名返利-领奖时间。
	IsSuper            bool  // 是否是超级用户（白名单）
}

func newLoginInfo(sid uint) *loginInfo {
	l := &loginInfo{
		sid:        sid,
		sessionMap: make(map[string]*sessionInfo, 2048),
	}
	cc, err := safecache.NewSafeCache(fmt.Sprintf("%s_%d", "gate_cache", sid), `{"interval":60}`)
	if err != nil {
		tilogs.L().Errorf("Gate Server Startup Failed, due to: %s", err.Error())
		panic("Gate Server Startup Failed, by loginCache init failed")
	}
	l.loginCache = cc
	actCache, err := safecache.NewSafeCache(fmt.Sprintf("%s_%d", "gate_act_cache", sid), `{"interval":60}`)
	if err != nil {
		tilogs.L().Errorf("Gate Server Startup Failed 1, due to: %s", err.Error())
		panic("Gate Server Startup Failed, by loginCache init failed 1")
	}
	l.activeSessionCache = actCache
	return l
}

func getRealLoginToken(loginToken string) (realLoginToken string, sid uint) {
	realLoginToken = loginToken
	// 重登的话客户端发过来的是加过re前缀的
	if strings.HasPrefix(loginToken, "re:") {
		realLoginToken = parseReloginToken(loginToken)
		if realLoginToken == "" {
			tilogs.L().Warnf("<Gate> parse relogin token result realLoginToken=%s loginToken=%s", realLoginToken, loginToken)
			return "", 0
		}
	}
	ss := strings.Split(realLoginToken, ",")
	if len(ss) < 2 {
		tilogs.L().Errorf("getRealLoginToken realLoginToken len < 2 %s ", realLoginToken)
		return "", 0
	}
	_sid, err := strconv.Atoi(ss[1])
	if err != nil {
		tilogs.L().Errorf("getRealLoginToken Atoi err %s %s", realLoginToken, err.Error())
		return "", 0
	}
	return realLoginToken, uint(_sid)
}

func (l *loginInfo) clear() {
	l.loginCache.ClearAll() // by zhngzhen 如果注释这行，gamex重启将不会把玩家踢回登录，在gamex重启完成后，玩家可以断线重连回来
	l.mu.Lock()
	l.sessionMap = make(map[string]*sessionInfo, 2048)
	l.mu.Unlock()
	l.activeSessionCache.ClearAll()
}

func (l *loginInfo) kickAll() {
	l.loginCache.ClearAll()

	l.mu.Lock()
	for _, sess := range l.sessionMap {
		sess.kickFunc(consts.ReLogin_Kick, 0, 0, consts.GMKickErrCode, "kickAll", nil)
	}
	l.sessionMap = make(map[string]*sessionInfo, 2048)
	l.mu.Unlock()

	l.activeSessionCache.ClearAll()
}

// queryUserLoginInfo handshake用来调用查询是否这个Logintoken已经推送到当前gate
func (l *loginInfo) queryUserLoginInfo(realLoginToken string, agent *client.PacketConnAgent) (*LoginNotify, int64) {
	// step 1: 检测 login token
	infoNotify := l.getAcidByLoginToken(realLoginToken)
	if infoNotify == nil {
		tilogs.L().Errorf("<Gate> not find acid, realLoginToken = %s ", realLoginToken)
		return nil, 0
	}
	tilogs.L().Debugf("<Gate> get acid by loginToken %v", infoNotify)
	acid := infoNotify.Account.String()

	// step 2: 设置新的连接信息
	newSessionInfo := &sessionInfo{
		ip:       agent.PacketConn.Conn.RemoteAddr().String(),
		kickFunc: getKickFunc(agent, acid),
		notifyFunc: func(info InfoNotifyToClient) {
			// NotifyInfo.SendInfoNotify(agent.SendPacket, info)
		},
	}
	l.mu.RLock()
	oldSessionInfo, ok := l.sessionMap[acid]
	l.mu.RUnlock()
	if ok {
		oldSessionInfo.kickFunc(consts.ReLogin_Kick, 0, 0, consts.Kick_Code_Not_Ready_For_Login, "queryUserLoginInfo", nil)
	}

	// step 3: 等待上次连接结束
	counter := 10 // 5s
	kickSuccess := false
	for {
		if oldSessionInfo == nil {
			kickSuccess = true
			break
		}
		exist := l.activeSessionCache.IsExist(fmt.Sprintf("%d", oldSessionInfo.id))

		counter = counter - 1
		if counter <= 0 {
			tilogs.L().Errorf("queryUserLoginInfo, relogin failed %d, acid %s oldSessionId %d", counter, acid, oldSessionInfo.id)
			return nil, 0
		}
		if !exist {
			// time.Sleep(time.Second * 2)
			kickSuccess = true
			break
		} else {
			time.Sleep(500 * time.Millisecond) // 500ms
			tilogs.L().Debugf("<Gate> queryUserLoginInfo waiting...., acid %s oldSessionId %d", acid, oldSessionInfo.id)
		}
	}

	// step 4: 设置为活跃的sessionId
	if kickSuccess {
		var _id int64
		l.mu.Lock()
		l.autoIncId++
		_id = l.autoIncId
		newSessionInfo.id = l.autoIncId
		l.sessionMap[acid] = newSessionInfo
		agent.SessionId = newSessionInfo.id
		l.mu.Unlock()
		l.activeSessionCache.Put(fmt.Sprintf("%d", _id), 0, defaultTokenAliveDur)
		tilogs.L().Infof("activeSessionCache put sessionId %d, acid %s", newSessionInfo.id, acid)
	} else {
		// 如果没有T掉上一个连接， 这里不让登录
		tilogs.L().Errorf("<Gate> fail to kick old login, acid %s %s %d", acid, realLoginToken, newSessionInfo.id)
	}

	return infoNotify, newSessionInfo.id
}

func getKickFunc(agent *client.PacketConnAgent, acid string) func(string, int, int, int, string, func()) {
	return func(reason string, after, nologin int, kickErrCode int, from string, afterCloseCallBack func()) {
		if after <= 0 {
			after = 1
		}
		NotifyInfo.SendKickNotify(acid, agent.SessionId, agent.AsyncSendPacket, reason, after, nologin, kickErrCode)
		go func() {
			<-time.After(100 * time.Millisecond) // 100ms
			if err := agent.Close(); err != nil {
				if net.NetIsClosed(err) {
					tilogs.L().Infof("<Gate> handleConnection kick agent close err %s, acid %s, from %s", err.Error(), acid, from)
				} else {
					tilogs.L().Errorf("<Gate> handleConnection kick agent close err %s, acid %s, from %s", err.Error(), acid, from)
				}
			}
			tilogs.L().Debugf("<Gate> handleConnection kick agent close %s, from %s", acid, from)
			if afterCloseCallBack != nil {
				afterCloseCallBack()
			}
		}()
	}
}

func parseReloginToken(reloginToken string) string {
	relogin := strings.TrimPrefix(reloginToken, "re:")
	loginTokenb, err := secure.Decode64FromNet(relogin)
	if err != nil {
		tilogs.L().Errorf("<Gate> queryUserLoginInfo, relogin failed 1, reloginToken=%s err %s", reloginToken, err.Error())
		return ""
	}
	return string(loginTokenb)
}

func (l *loginInfo) getAcidByLoginToken(loginToken string) *LoginNotify {
	v := l.loginCache.Get(loginToken)

	if v == nil {
		return nil
	} else {
		loginNotify := v.(*LoginNotify)
		return loginNotify
	}
}

// RegisterLoginNotify 通过login server通知rpc方式注册进来
// 通常玩家**准备**登录，在login server成功获取getGate后，此接口会被调用
func (l *loginInfo) RegisterLoginNotify(loginToken string, lt *LoginNotify) bool {
	tilogs.L().Debugf("<Gate> loginInfo.RegisterLoginInfo coming! token got:%s", loginToken)
	const defaultTokenAliveDur = 36 * time.Hour // 36 HOUR
	if err := l.loginCache.Put(loginToken, lt, defaultTokenAliveDur); err != nil {
		tilogs.L().Errorf("<Gate> loginInfo.RegisterLoginNotify get Error: %s", err.Error())
		return false
	}
	l.loginCache.Incr(cachedToken)
	return true
}

// ///////////////////////

func (l *loginInfo) GetSession(acid string) (*sessionInfo, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	f, ok := l.sessionMap[acid]
	return f, ok
}

func (l *loginInfo) GetInfoNotifyHandler(accountId string) (func(InfoNotifyToClient), bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	f, ok := l.sessionMap[accountId]
	if ok {
		return f.notifyFunc, ok
	} else {
		return nil, ok
	}
}

// /////////////////////////////////

func (g *GateServer) RegisterLoginToken(args *LoginNotify, reply *bool) error {
	defer tilogs.PanicCatcher("RegisterLoginToken %v", args)
	tilogs.L().Debugf("<Gate> Happy!! Gate RPC: user_id %d logged in with loginToken %s", args.UserId, args.LoginToken)
	ss := strings.Split(args.LoginToken, ",")
	if len(ss) < 2 {
		return nil
	}
	sid, err := strconv.Atoi(ss[1])
	if err != nil {
		return nil
	}
	loginInfo := g.getShardLoginInfo(uint(sid))
	if loginInfo == nil {
		return nil
	}
	*reply = loginInfo.RegisterLoginNotify(args.LoginToken, args)
	return nil
}

type KickOfflineParam struct {
	LoginToken      string
	AccountID       string
	Reason          string
	KickErrCode     int
	AfterDuration   int
	NoLoginDuration int
	ByGM            bool
	Source          string
}

// KickItOffline 返回值如果是1, 发现这个玩家，并要求其断线
// 用途1：login服务器收到同一个帐号的登录信息，会给已经登录的人发送Kick RPC
// 用途2：客服系统可能需要对某些玩家进行强制踢出游戏测操作，并禁止登录N分钟
// 用途3：系统运维前，需要提出所有在线玩家，强迫下线
func (g *GateServer) KickItOffline(args *KickOfflineParam, reply *int) error {
	defer tilogs.PanicCatcher("KickItOffline %v", args)
	tilogs.L().Infof("<Gate> GateServer got kick info %v, byGM %v", args, args.ByGM)
	*reply = 0

	if args == nil {
		return nil
	}
	a, _ := db.ParseAccount(args.AccountID)
	loginInfo := g.getShardLoginInfo(a.ShardId)
	if loginInfo == nil {
		return nil
	}

	// 若在线，踢下线
	session, ok := loginInfo.GetSession(args.AccountID)
	if ok {
		exist := loginInfo.activeSessionCache.IsExist(fmt.Sprintf("%d", session.id))
		if exist {
			session.kickFunc(args.Reason, args.AfterDuration, args.NoLoginDuration, args.KickErrCode,
				"KickItOffline", func() {
					// 如果是封禁，通知gamex
					if args.ByGM {
						gGamexMgr.forceQuit(args.AccountID)
					}
				})
			*reply = 1
			tilogs.L().Infof("<Gate> GateServer kick accountid %s sessionid %d success, %v, byGM %v, oldIp %s, newSource %s",
				args.AccountID, session.id, args, args.ByGM, session.ip, args.Source)
		}
	} else {
		// 如果是封禁，离线也通知gamex强制踢人，踢出离线保护状态
		if args.ByGM {
			gGamexMgr.forceQuit(args.AccountID)
		}
	}

	return nil
}

// GameHealthCheck  用于检查gate和gamex之间的连接是否正常
func (g *GateServer) GameHealthCheck(args *GameCheckParam, reply *GameCheckRet) error {
	gameMgr, ok := g.gameSrvs.(*ProtoGameServerManager)
	if !ok || gameMgr == nil {
		*reply = GameCheckFail
		tilogs.L().Errorf("GameHealthCheck ProtoGameServerManager not found")
		return nil
	}
	mgr := gameMgr.gamesMgr
	if mgr == nil {
		*reply = GameCheckFail
		tilogs.L().Errorf("GameHealthCheck gamesMgr not found")
		return nil
	}

	if !mgr.canHealthCheck() {
		tilogs.L().Infof("GameHealthCheck canHealthCheck false")
		*reply = GameCheckSuccess
		return nil
	}

	var param GameCheckParam
	if args != nil {
		param = *args
	}
	g.waitGroup.Wrap(func() { mgr.healthCheck(param) })
	*reply = GameCheckSuccess
	return nil
}

type NotifyInfoParam struct {
	LoginToken string
	AccountID  string
	Info       InfoNotifyToClient
}

// 向玩家Client推送信息
// 用途1：若玩家被禁言,则通过这个通知客户端
func (g *GateServer) NotifyInfo(args *NotifyInfoParam, reply *int) error {
	defer tilogs.PanicCatcher("NotifyInfo %v", args)
	tilogs.L().Infof("GateServer got kick info %v", args)
	*reply = 0

	if args == nil {
		return nil
	}
	a, _ := db.ParseAccount(args.AccountID)
	loginInfo := g.getShardLoginInfo(a.ShardId)
	if loginInfo == nil {
		return nil
	}
	h, ok := loginInfo.GetInfoNotifyHandler(args.AccountID)
	if ok {
		if h != nil {
			tilogs.L().Infof("GateServer NotifyInfo %s success, %v", args.AccountID, args)
			h(args.Info)
		}

		*reply = 1
	}

	return nil
}
