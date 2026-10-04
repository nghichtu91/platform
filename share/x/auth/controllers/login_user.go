package controllers

import (
	"crypto/rand"
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"

	"github.com/nghichtu91/platform/share/planx/arrayhelper"
	"github.com/nghichtu91/platform/share/x/common/notice"

	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/planx/virtual_gid"

	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/ginhelper"
	"github.com/nghichtu91/platform/share/planx/multilingual"
	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/auth/errorctl"
	"github.com/nghichtu91/platform/share/x/auth/logic"
	"github.com/nghichtu91/platform/share/x/auth/models"
	"github.com/nghichtu91/platform/share/x/common/consts"
	"github.com/nghichtu91/platform/share/x/common/msg"
	"github.com/nghichtu91/platform/share/x/gatex/gate"
)

const channelServerType = "channel"

// LoginController is in charge of login process.
type LoginController struct {
	ginhelper.GinController
}

func (lc *LoginController) FetchShardsInfoV2() gin.HandlerFunc {
	return func(c *gin.Context) {
		tilogs.L().Debugf("rev FetchShardsInfoV2")
		gids := c.Param("gid")
		ver := lc.GetString(c, "ver")
		deviceId := lc.GetString(c, "deviceId")
		authToken := lc.GetString(c, "at")    // AuthToken UUID
		channel, _ := lc.GetInt(c, "channel") // 渠道
		sp := lc.GetString(c, "sp")
		language := lc.GetString(c, "language")
		tilogs.L().Debugf("<fetch shard info> gids:%s channel %d ver:%s deviceId %v, authToken %v sp %s language %s",
			gids, channel, ver, deviceId, authToken, sp, language)

		gid, err := strconv.Atoi(gids)
		if err != nil {
			errorctl.CtrlErrorReturn(c, "[Login.FetchShards]", err, errorctl.FailForClient, errorctl.ClientErrorFormatVerifyFailed)
			return
		}
		if config.GidCfg.ServerType == channelServerType && (gid == 61 || gid == 63) {
			gid = 62
		}
		// 验证authToken有效，不区分game id，不区分shard id
		// 为辅助工具使用
		var sdkId string
		// if authToken != "taihe" {
		// TODO bug 旧mgo接口，Unity登录传空authToken能通过，新接口暂时兼容了
		_, sdk, err := models.LoginVerifyAuthToken(authToken, "fetchShards")
		sdkId = sdk
		if err != nil {
			errorctl.CtrlErrorReturn(c, "[Login.FetchShardsInfo]", err, errorctl.FailForClient, errorctl.ClientErrorLoginAuthtokenNotReady)
			return
		}
		// }

		isSuper := false
		if len(ver) > 0 {
			isSuper = models.IsWhiteListForCurrentUser(deviceId, c.ClientIP(), sdkId)
			whiteListPwd := models.GetWhiteListPwd(gids, ver)
			tilogs.L().Debugf("FetchShardsInfoV2 whiteListPwd %v", whiteListPwd)
			if !isSuper && len(sp) > 0 && len(whiteListPwd) > 0 {
				if sp == whiteListPwd {
					isSuper = true
					tilogs.L().Debugf("FetchShardsInfoV2 got super")
				}
			}
		}

		shards, err := models.FetchShardsInfo(gid)
		if err != nil {
			tilogs.L().Errorf("cannot get shards err :%v", err)
			errorctl.CtrlErrorReturn(c, "[Login.FetchShards]", err, errorctl.FailForServer, errorctl.ClientErrorMaybeDBProblem)
			return
		}

		var vgid int
		if virtual_gid.IsVirtualGroupEnabled() {
			vgid = virtual_gid.GetVirtualGroupIDByChannel(channel)
		}

		now := timeutil.Now().Unix()
		tilogs.L().Debugf("<fetch shard info> %v, now = %v", shards, now)
		_shards := make([]models.ShardInfoForClient, 0, len(shards))
		for _, s := range shards {
			showState, err := strconv.Atoi(s.ShowState)
			if err != nil {
				tilogs.L().Debugf("<fetch shard info> showState is not exist, shardInfo = %v", s)
				continue // 没配showState的话，就跳过
			}

			// 服务器尚未准备就绪，白名单用户和普通用户都看不懂
			if showState == consts.ShowStateNotStart || s.StartTime <= 0 {
				continue
			}

			// 当前未达到对外开服时间，玩家不可见
			// 若不是超级用户，只能看到正常上线了的shard，看不到ShowState_Super的服
			if (!isSuper && showState == consts.ShowStateHide) || (!isSuper && s.StartTime > now) {
				continue
			}

			// 虚拟大区开启时，过滤不属于该虚大区且不是激战虚大区的服务器
			if s.VirtualGID != vgid && s.VirtualGID != virtual_gid.VirtualGidC {
				continue
			}

			if s.GetState() == consts.StateOnline {
				_s := s.Clone()
				_s.DisplayName = multilingual.UnMarshalByLanguage(language, _s.DisplayName)
				_s.NameExtra = multilingual.UnMarshalByLanguage(language, _s.NameExtra)
				_shards = append(_shards, _s)
			}
		}

		r := struct {
			Result string                      `json:"result"`
			Shards []models.ShardInfoForClient `json:"shards"`
		}{
			"ok",
			_shards,
		}
		tilogs.L().Debugf("<fetch shard info> %v", r)
		c.JSON(200, r)
	}
}

// GetGate is used for client to get gate server on specific shards servers
// 如果返回值中有retry，则需要客户端等待ms毫秒后再次重试
func (lc *LoginController) GetGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		tilogs.L().Debugf("[Login.getGate] rev GetGate")
		authToken := lc.GetString(c, "at") // AuthToken UUID
		// game id maybe used later for shard of redis cluster
		// gID := lc.GetString("gid")      // 渠道 game id ex. ios: 1
		shardName := lc.GetString(c, "sn") // 子服的ID -- shard name
		deviceId := lc.GetString(c, "deviceId")
		// Shard name 换算shard id(sid)
		sid, gid, startTime, showState, err := models.GetShardID(shardName)
		if err != nil {
			// 数据库级错误, 键值不存在或者格式不对等，或者连接不上
			errorctl.CtrlErrorReturn(c, "[Login.getGate]", err, errorctl.FailForClient, errorctl.ClientErrorMaybeDBProblem)
			return
		}

		ver := lc.GetString(c, "ver")
		sp := lc.GetString(c, "sp")
		requireHost := lc.GetString(c, "requireHost")
		tilogs.L().Debugf("[Login.getGate] GetGate ver %v sp %v, authToken %v, sn %s, deviceId %s, requireHost %s",
			ver, sp, authToken, shardName, deviceId, requireHost)

		show_state, err := strconv.Atoi(showState)
		if err != nil {
			show_state = consts.ShowStateNotStart // 没配showState的话，不允许登录
		}

		// 验证authToken有效，不区分game id，不区分shard id
		uid, sdkDeviceId, err := models.LoginVerifyAuthToken(authToken, "getGate")
		if err != nil {
			errorctl.CtrlErrorReturn(c, "[Login.getGate]", err, errorctl.FailForClient, errorctl.ClientErrorLoginAuthtokenNotReady)
			return
		}
		// sdkDeviceId, _ := models.GetSdkDeviceIdByUid(uid.String())

		tilogs.L().Infof("[Login.getGate] LoginVerifyAuthToken, authToken %v uid %v", authToken, uid)

		if uid == db.InvalidUserID {
			errorctl.CtrlErrorReturn(c, "[Login.getGate]", fmt.Errorf("timeout"), errorctl.FailForClient, errorctl.ClientErrorLoginAuthtokenNotReady)
			// errorctl.CtrlErrorReturn(c, "[Login.getGate]",
			//	fmt.Errorf("uid should not be 0, this should not happen!"),
			//	errorctl.ClientErrorUnknown)
			return
		}
		account := db.Account{
			UserId:  uid,
			GameId:  gid,
			ShardId: sid,
		}

		// todo 进行判断
		var isSuper bool
		now := timeutil.Now().Unix()
		channelId := lc.GetString(c, "channelId")
		if len(ver) > 0 {
			maintenanceInfo := models.GetGonggao(strconv.Itoa(int(gid)), ver)

			// 检查白名单密码 todo 在这里对isSuper进行修改  sdkId,deviceId,ip三种方式
			ip := c.ClientIP()
			isSuper = models.IsWhiteListForCurrentUser(deviceId, ip, sdkDeviceId)
			tilogs.L().Debugf("[Login.getGate] GetGate pre check WhitelistPwd isSuper:%t", isSuper)

			whiteListPwd := models.GetWhiteListPwd(fmt.Sprintf("%d", gid), ver)
			if !isSuper && len(whiteListPwd) > 0 && sp == whiteListPwd {
				tilogs.L().Debugf("[Login.getGate] GetGate got super")
				isSuper = true
			}

			for _, maintenanceTimeInfo := range maintenanceInfo.TimeInfos {
				// 如果属于当前渠道
				if isChannelOk(maintenanceTimeInfo.Channels, channelId) {
					if maintenanceTimeInfo.StartTime <= 0 || maintenanceTimeInfo.EndTime <= 0 {
						continue
					}

					if now < maintenanceTimeInfo.StartTime || now > maintenanceTimeInfo.EndTime {
						continue
					}

					// 在阻挡公告时间内
					// 只有白名单的能进
					if !isSuper {
						errorctl.CtrlErrorReturn(c, "[Login.getGate]",
							fmt.Errorf("maintance"), errorctl.FailForClient, errorctl.ClientErrorGetGateMaintenance)
						return
					}
				}
			}
		}
		if !isSuper {
			err1, ban_time, ban_reason, _ := models.UserBanInfo(uid)
			if err1 != nil {
				// 该玩家已被禁言
				errorctl.CtrlBanReturn(c, "[Login.getGate]", err1, errorctl.FailForServer, errorctl.ClientErrorBanByGM, ban_time, ban_reason)
				return
			}
			// 检查角色是否被封禁
			playerBanInfo := models.GetPlayerBanInfo(uid.String(), account.String())
			if playerBanInfo != nil {
				if playerBanInfo.BanEndTime > 0 && playerBanInfo.BanEndTime > timeutil.Now().Unix() {
					// 该角色已被封禁
					errorctl.CtrlBanReturn(c, "[Login.getGate]", models.XErrByBanPlayer, errorctl.FailForServer, errorctl.ClientErrorBanPlayerByGm, playerBanInfo.BanEndTime, playerBanInfo.BanResult)
					return
				}
			}
		}

		// 服务器状态为未启动
		if show_state%consts.ShowStateLockedAdd == consts.ShowStateNotStart {
			tilogs.L().Errorf("[Login.getGate] GetGate error, shardState = ShowStateNewSer, shardName = %v", shardName)
			errorctl.CtrlErrorReturn(c, "[Login.getGate]", fmt.Errorf("startTime is not"), errorctl.FailForClient, errorctl.ClientErrorGetGateServerIsAbnormality)
			return
		}

		// 尚未达到对外开放时间
		if (!isSuper && channelId != consts.RobotChannel && now < startTime) || startTime <= 0 {
			tilogs.L().Errorf("[Login.getGate] GetGate error, startTime > now, shardName = %v, state = %v, startTime = %d, now = %d", shardName, show_state, startTime, now)
			errorctl.CtrlErrorReturn(c, "[Login.getGate]", fmt.Errorf("startTime is not"), errorctl.FailForClient, errorctl.ClientErrorGetGateServerIsAbnormality)
			return
		}

		if show_state%consts.ShowStateLockedAdd == consts.ShowStateMaintenanceNew || show_state%consts.ShowStateLockedAdd == consts.ShowStateMaintenanceHot {
			ip := c.ClientIP()
			tilogs.L().Debugf("[Login.getGate] GetGate clientip %s", ip)
			if !isSuper {
				errorctl.CtrlErrorReturn(c, "[Login.getGate]", fmt.Errorf("iplimit %s", ip), errorctl.FailForClient, errorctl.ClientErrorGetGateMaintenance)
				return
			}
		}

		// 服务器为火爆状态时，若该服务器无玩家角色，则不允许登录
		if show_state%consts.ShowStateLockedAdd == consts.ShowStateHot {
			historyShards, err := models.GetUserHistoryShard(uid.String())
			if err != nil {
				tilogs.L().Infof("[Login.getGate] GetUserShardHasRole err = %v", err)
				errorctl.CtrlErrorReturn(c, "[Login.getGate]", fmt.Errorf("GetUserShardHasRole"), errorctl.FailForClient, errorctl.ClientErrorUnknown)
				return
			}

			isFind := false
			for _, history := range historyShards {
				if history == fmt.Sprintf("%d", sid) {
					isFind = true
					break
				}
			}

			if !isFind {
				tilogs.L().Infof("[Login.getGate] server is lock")
				errorctl.CtrlErrorReturn(c, "[Login.getGate]", fmt.Errorf("server is lock"), errorctl.FailForClient, errorctl.ClientErrorGetGateServerIsFull)
				return
			}
		}

		// 生成LoginToken
		loginToken := fmt.Sprintf("%s,%d", uuid.NewV4().String(), sid)
		// 防止同一个帐号多重login, 通知上一个登录的设备退出
		if st := avoidMultipleLogin(gid, uid, consts.ReLogin_Kick, 10, false, consts.ReLoginKickErrCode, "GetGate_"+c.ClientIP()); st == "RETURN" {
			c.JSON(200, map[string]interface{}{
				"result": "retry",
				"ms":     2000, // 2000 ms后重试
			})
			return
		}

		tilogs.L().Debugf("[Login.getGate] after avoidMultipleLogin, authToken %v uid %v", authToken, uid)

		var gateInfo *models.GateInfo
		var recallId string
		// get number ID
		numberID, err := models.GetNumberIDForPlayer(account.String())
		if err != nil {
			tilogs.L().Errorf("get number id for player err by %v", err)
		}
		tilogs.L().Debugf("[Login.getGate] after avoidMultipleLogin, authToken %v account %v", authToken, account)

		recallId, err = models.GetRecallID(fmt.Sprintf("%d:%d:%s", gid, sid, uid.String()))
		if err != nil {
			tilogs.L().Errorf("get recallId for player err by %v", err)
		}
		tilogs.L().Debugf("[Login.getGate] after GetRecallID, authToken %v account %v", authToken, account)

		if recallId == "" {
			recallId = GetSixRecallId(numberID)
			// tilogs.L().Infof("[Login.getGate] Get player recall id: %v", recallId)
			err = models.SetRecallInfo(recallId, fmt.Sprintf("%d:%d:%s", gid, sid, uid.String()), 0, int64(gid))
			if err != nil {
				tilogs.L().Errorf("set RecallID fail: %v", err)
				return
			}
		} else {
			tilogs.L().Debugf("[Login.getGate] AcoountId:%s,recallId:%s", fmt.Sprintf("%d:%d:%s", gid, sid, uid.String()), recallId)
		}

		// 获取玩家的返利信息。
		var moneyScore, loginDay, fair1v1Dan, fair3v3Dan int32
		var claimTime int64
		var payReward, loginAndRankReward bool
		rewardInfos, handleCode := models.PayRewardForUser(models.RewardAll, uid, models.RewardOptQuery)
		if handleCode == models.RewardCodeOK && rewardInfos != nil {
			if info, ok := rewardInfos[models.RewardPay]; ok {
				moneyScore = info.MoneyScore
				payReward = info.HasClaimed
			}
			if info, ok := rewardInfos[models.RewardLoginAndRank]; ok {
				loginDay = info.LoginDay
				fair1v1Dan = info.Fair1v1Dan
				fair3v3Dan = info.Fair3v3Dan
				loginAndRankReward = info.HasClaimed
				claimTime = info.ClaimTime
			}
		}

		var retry uint
		for {
			retry += 1
			if retry > 5 {
				errorctl.CtrlErrorReturn(c, "[Login.getGate]", err, errorctl.FailForServer, errorctl.ClientErrorGetGateNotExist)
				return
			}
			// 确认有效后，根据数据库记录给玩家返回有效的（人数少的服务器）Gate IP
			ob, err := models.GetOneGate(gid, sid)
			if ob == nil || err != nil {
				if err == models.XErrGetGateNotExist {
					errorctl.CtrlErrorReturn(c, "[Login.getGate]", err, errorctl.FailForServer, errorctl.ClientErrorGetGateNotExist)
				} else {
					errorctl.CtrlErrorReturn(c, "[Login.getGate]", err, errorctl.FailForServer, errorctl.ClientErrorUnknown)
				}
				return
			}
			tilogs.L().Debugf("[Login.getGate] after GetOneGate ob: %+v, err:%v, authToken %v account %v", ob, err, authToken, account)
			// loginToken, user id 发送到相关Gate server api， 由Gate Server保存在内存中。
			// FIXME loginToken在后续操作中会变成一个密钥，所以需要使用更高级的加密算法加密传输！
			// 怎么处理失败，要进行指数级别重试么？
			var conn *rpc.Client
			if conn, err = jsonrpc.Dial("tcp", ob.RPCIPAddrPort); err != nil {
				tilogs.L().Errorf("[Login.getGate] jsonrpc.Dial ob %+v err %v, authToken %v account %v", ob, err, authToken, account)
				// CtrlErrorReturn(c, "[Login.getGate.notifygate]", err, ob)
				// beego.Error("[Login.getGate.notifygate] error %s, %v", err.Error(), ob)
				// reason := fmt.Sprintf("[Login.getGate.notifygate] error %s, %v", err.Error(), ob)
				// models.CleanGate(ob, reason)
				if conn != nil {
					conn.Close()
				}
				continue
			}
			var reply bool
			if err = conn.Call("GateServer.RegisterLoginToken",
				&gate.LoginNotify{
					Account:            account,
					LoginToken:         loginToken,
					NumberID:           numberID,
					SdkDeviceId:        sdkDeviceId,
					RechargeMoney:      moneyScore,
					RechargeReturn:     payReward,
					LoginDay:           loginDay,
					Fair1v1Dan:         fair1v1Dan,
					Fair3v3Dan:         fair3v3Dan,
					LoginAndRankReward: loginAndRankReward,
					ClaimTime:          claimTime,
					IsSuper:            isSuper,
				},
				&reply); err != nil {
				tilogs.L().Errorf("[Login.getGate] conn.Call ob %v err %v, authToken %v account %v", ob, err, authToken, account)
				// reason := fmt.Sprintf("[Login.getGate.call] error %s, %v", err.Error(), ob)
				// models.CleanGate(ob, reason)
				if conn != nil {
					conn.Close()
				}
				// CtrlErrorReturn(c, "[Login.getGate RPC call]", err, ob)
				continue
			}
			if conn != nil {
				conn.Close()
			}
			err = nil

			gateInfo = ob
			break
		}

		// 记录玩家历史登录
		models.SetUserHistoryShard(uid.String(), fmt.Sprintf("%d", sid))
		tilogs.L().Infof("[Login.getGate] gate list %s %s, authToken %v account %v, sid = %d",
			gateInfo.GameIPAddrPort, gateInfo.HostPort, authToken, account.String(), sid)

		team, cfgteam := GetServerTeam(uid.Bytes(), gid, sid)
		enc_ip := secure.Encode64ForNet([]byte(gateInfo.GetConnectAddr(requireHost == "1")))
		enc_token := secure.Encode64ForNet([]byte(loginToken))
		logic.LogLoginTeam(uid.String(), team, "", cfgteam, sid)
		c.JSON(200, map[string]interface{}{
			"ip":         enc_ip,
			"result":     "ok",
			"logintoken": enc_token, // Auth token 返回给客户端
			"recallid":   recallId,
			"Team":       team,
		})
	}
}

// DebugKick 调试用API，协助客户端测试踢人操作
func (lc *LoginController) DebugKick() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch config.GidCfg.RunMode {
		case planx.RunMode_Dev:
			loginToken := lc.GetString(c, "lt")
			Reason := lc.GetString(c, "reason")
			var AfterDur, NoLogin int
			var err error
			AfterDur, err = lc.GetInt(c, "after")
			if err != nil {
				AfterDur = 5
			}
			NoLogin, err = lc.GetInt(c, "nologin")
			if err != nil {
				NoLogin = 5
			}

			accountId := lc.GetString(c, "account") //0:0:0001
			account, err := db.ParseAccount(accountId)

			if err != nil {
				c.String(500, "account Err")
				return
			}
			_, _,
				OldRpc, ok := models.GetLastLoginStatus(account.GameId, account.UserId)
			if ok {
				var reply int
				if Conn, err := jsonrpc.Dial("tcp", OldRpc); err == nil {
					err = Conn.Call("GateServer.KickItOffline",
						&msg.KickOfflineParam{
							LoginToken:      loginToken,
							AccountID:       accountId,
							Reason:          Reason,
							KickErrCode:     consts.ReLoginKickErrCode,
							AfterDuration:   AfterDur,
							NoLoginDuration: NoLogin,
						},
						&reply)
					Conn.Close()
					result := map[string]interface{}{
						"rpc": true,
					}
					if err != nil {
						result["err"] = err.Error()
					}
					c.JSON(200, result)
					tilogs.L().Debugf("DebugKick try to kick login status!", loginToken, accountId)
				}
			} else {
				tilogs.L().Debugf("DebugKick did not find rpc login status!", loginToken, accountId)
			}

		default:
		}
		c.String(404, "Runmode Err")
	}
}

func GetSixRecallId(i int64) string {
	num := convert62(i)
	lNum := len(num)
	num2 := GetString(6 - lNum)
	// fmt.Println(num + num2)
	return num + num2
}

var strstr = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// GetString ...
func GetString(size int) string {
	data := make([]byte, size)
	out := make([]byte, size)
	buffer := len(strstr)
	_, err := rand.Read(data)
	if err != nil {
		panic(err)
	}

	for id, key := range data {
		x := byte(int(key) % buffer)
		out[id] = strstr[x]
	}
	return string(out)
}

func convert62(a int64) string {
	ret := make([]byte, 0)
	b := a
	for {
		c := b % 62
		ret = append(ret, getStrBy62(c))
		b /= 62
		if b == 0 {
			break
		}
	}
	return string(ret)
}

func getStrBy62(a int64) byte {
	if a >= 0 && a <= 25 {
		return byte('A' + a)
	} else if a >= 26 && a <= 51 {
		return byte('a' + a - 26)
	} else {
		return byte('0' + a - 52)
	}
}

func isChannelOk(chnannelList []string, ch string) bool {
	return len(chnannelList) == 0 ||
		arrayhelper.ContainsString(chnannelList, ch) ||
		arrayhelper.ContainsString(chnannelList, notice.ChannelId_All)
}
