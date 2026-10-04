package controllers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"strings"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/x/common/consts"

	uuid "github.com/satori/go.uuid"

	"github.com/nghichtu91/platform/share/planx/ginhelper"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	//"github.com/nghichtu91/platform/share/x/auth/config"

	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/x/auth/errorctl"
	"github.com/nghichtu91/platform/share/x/auth/models"
	"github.com/nghichtu91/platform/share/x/gatex/gate"
)

// APIController is in charge of internal communication with gate servers.
type UserMgrController struct {
	ginhelper.GinController
}

// NotifyUserLogin is called by gate/game server
// to notify login server that user logged in
// Gate发现玩家成功登录后调用参数 logintoken, userid
// @router /notifylogin [post]
func (uc *UserMgrController) NotifyUserLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		data := struct {
			LoginToken string `form:"logintoken"`
			AccountId  string `form:"accountid"`
			Rpc        string `form:"rpcaddrport"`
		}{}
		err := c.ShouldBindWith(&data, binding.Form)

		if err != nil {
			tilogs.L().Errorf("NotifyUserLogin Err %s", err.Error())
			c.JSON(500, err.Error())
			return
		}

		tilogs.L().Debugf("NotifyUserLogin get user: %s, %s, %s",
			data.AccountId, data.LoginToken, data.Rpc)
		_account, err := db.ParseAccount(data.AccountId)
		if err != nil {
			c.String(500, "err : %s", err.Error())
			return
		}
		models.UpdateLoginStatus(_account, data.Rpc, data.LoginToken, models.LSC_LOGIN)

		c.JSON(200, "ok")

	}
}

// Auth服务器通知Login踢人 ?id={uid}&gid={gid}&banTime={banTime}&reason{reason}
// http://127.0.0.1:8081/login/v1/api/kick [get]
func (uc *UserMgrController) NotifyKickUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var rst struct {
			Result string `json:"result"`
		}

		id := c.DefaultQuery("id", "")
		gid, err := uc.GetInt(c, "gid")
		if id == "" || err != nil {
			tilogs.L().Errorf("NotifyKickUser Err by ID nil")
			rst.Result = "NOID"
			c.JSON(500, rst)
			return
		}
		banTime, _ := uc.GetInt64(c, "banTime")
		reason := uc.GetString(c, "reason")
		byGM := uc.GetString(c, "byGM")
		avoidMultipleLogin(uint(gid), db.UserIDFromStringOrNil(id), reason, banTime, byGM == "yes", consts.GMKickErrCode, "NotifyKickUser")

		tilogs.L().Debugf("NotifyKickUser user: %s %d %s", id, banTime, reason)
		rst.Result = "ok"
		c.JSON(200, rst)
	}
}

// Auth服务器通知Login通知禁言信息给客户端 ?id={uid}&gid={gid}&gagt={gag_time}
// http://127.0.0.1:8081/login/v1/api/kick [get]
func (uc *UserMgrController) NotifyGagUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var rst struct {
			Result string `json:"result"`
		}

		id := c.DefaultQuery("id", "")
		gid, err := uc.GetInt(c, "gid")
		gag, err_gag := uc.GetInt64(c, "gagt")
		if id == "" || err != nil || err_gag != nil {
			tilogs.L().Errorf("NotifyKickUser Err by ID nil")
			rst.Result = "NOID"
			c.JSON(500, rst)
			return
		}

		info := gate.InfoNotifyToClient{
			GagTime: gag,
		}

		notifyInfo(uint(gid), db.UserIDFromStringOrNil(id), info)

		tilogs.L().Debugf("NotifyGagUser user: %s", id)
		rst.Result = "ok"
		c.JSON(200, rst)

	}
}

// NotifyUserLogout is called by gate/game server
// to notify login server that user logged out
// Gate发现玩家掉线后调用 或者 玩家主动注销
// @router /notifylogout [post]
func (uc *UserMgrController) NotifyUserLogout() gin.HandlerFunc {
	return func(c *gin.Context) {
		data := struct {
			LoginToken string `form:"logintoken"`
			AccountId  string `form:"accountid"`
			Rpc        string `form:"rpcaddrport"`
		}{}
		err := c.ShouldBindWith(&data, binding.Form)

		if err != nil {
			tilogs.L().Errorf("NotifyUserLogin Err %s", err.Error())
			c.JSON(500, err.Error())
			return
		}

		tilogs.L().Debugf("NotifyUserLogout get user: %s, %s, %s",
			data.AccountId, data.LoginToken, data.Rpc)
		_account, err := db.ParseAccount(data.AccountId)
		if err != nil {
			c.String(500, "err : %s", err.Error())
			return
		}

		models.UpdateLoginStatus(_account, data.Rpc, data.LoginToken, models.LSC_LOGOFF)

		c.JSON(200, "ok")
	}
}

func (uc *UserMgrController) NotifyUserInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		data := struct {
			Gid           string `form:"gid"`
			ShardId       string `form:"shardid"`
			Uid           string `form:"uid"`
			RoleName      string `form:"rolename"`
			RoleLevel     string `form:"rolelevel"`
			MainHero      string `form:"mainhero"`
			LastLoginTime string `form:"lastLoginTime"`
			MainlineLevel string `form:"mainlineLevel"`
			PayFeedBack   string `form:"PayFeedBack"`
			Newrole       string `form:"newrole"`
			HeadIcon      string `form:"headicon"`
			PlayerId      string `form:"playerid"`
		}{}

		ret := struct {
			Res            string
			HadGotFeedBack bool
		}{}
		err := c.ShouldBindWith(&data, binding.Form)
		if err != nil {
			tilogs.L().Errorf("NotifyUserInfo Err %s", err.Error())
			ret.Res = err.Error()
			c.JSON(200, ret)
			return
		}

		gid, err := strconv.Atoi(data.Gid)
		if err != nil {
			tilogs.L().Errorf("NotifyUserInfo Err1 %s", err.Error())
			ret.Res = err.Error()
			c.JSON(200, ret)
			return
		}
		sid, err := strconv.Atoi(data.ShardId)
		if err != nil {
			tilogs.L().Errorf("onligin Err2 %s", err.Error())
			ret.Res = err.Error()
			c.JSON(200, ret)
			return
		}

		tilogs.L().Debugf("NotifyUserInfo rec %v", data)
		if data.RoleLevel != "" {
			err := models.SetUserShardHasRole(gid, sid, data.Uid, data.RoleName, data.RoleLevel, data.MainHero, data.LastLoginTime, data.MainlineLevel, data.HeadIcon, data.PlayerId)
			if err != nil {
				tilogs.L().Errorf("NotifyUserInfo Err3 %s", err.Error())
				ret.Res = err.Error()
				c.JSON(200, ret)
				return
			}
			tilogs.L().Debugf("NotifyUserInfo %d:%d:%s", gid, sid, data.Uid)
		}

		if err := models.SetUserLastShard(data.Uid, gid, sid); err != nil {
			tilogs.L().Errorf("NotifyUserInfo SetUserLastShard Err4 %s", err.Error())
			ret.Res = err.Error()
			c.JSON(200, ret)
			return
		}

		if data.PayFeedBack == "wantGet" {
			err, canGot := models.SetUserPayFeedBackIfNot(data.Uid)
			if err != nil {
				tilogs.L().Errorf("NotifyUserInfo SetUserPayFeedBackIfNot Err5 %s", err.Error())
				ret.Res = err.Error()
				c.JSON(200, ret)
				return
			}
			ret.HadGotFeedBack = canGot
		}

		if data.Newrole == "1" {
			err := models.AddShardNewUser(data.ShardId)
			if err != nil {
				tilogs.L().Errorf("NotifyUserInfo AddShardNewUser Err6 %s", err.Error())
				ret.Res = err.Error()
				c.JSON(200, ret)
				return
			}
		}
		ret.Res = "ok"
		c.JSON(200, ret)
	}
}

// OnlineStatusQuery is used for find user login status
// 在线登录状态查询
// @router /onlinestatusquery [get]
func (uc *UserMgrController) OnlineStatusQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, "ok")
	}
}

// AuthTokenNotify called by Auth server, with base64 encoded content of data
// data content is base64(json{ at:authToken, uid:user_id })
func (uc *UserMgrController) AuthTokenNotify() gin.HandlerFunc {
	return func(c *gin.Context) {
		var authToken struct {
			AuthToken string    `json:"at"`
			UserID    db.UserID `json:"uid"`
		}
		info := uc.GetString(c, "info")
		if info == "" {
			errorctl.CtrlErrorReturn(c, "[AuthToken0]", fmt.Errorf("info param is empty"), errorctl.FailForServer, -1)
			return
		}

		tilogs.L().Debugf("[AuthToken0] raw info param:%v", info)
		js, err := base64.URLEncoding.DecodeString(info)
		if err != nil {
			errorctl.CtrlErrorReturn(c, "[AuthToken1]", err, errorctl.FailForServer, -1)
			return
		}
		tilogs.L().Debugf("[AuthToken1] info param decoded:%v", string(js))
		err = json.Unmarshal(js, &authToken)
		if err != nil {
			errorctl.CtrlErrorReturn(c, "[AuthToken2]", err, errorctl.FailForServer, -2)
			return
		}
		err = models.LoginRegAuthToken(authToken.AuthToken, authToken.UserID, "")
		if err != nil {
			errorctl.CtrlErrorReturn(c, "[AuthToken3]", err, errorctl.FailForServer, -3)
			return
		}

		c.JSON(200, struct {
			Result string `json:"result"`
		}{
			Result: "ok",
		})
	}
}

func (uc *UserMgrController) BanUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("id")
		ban_time, err := uc.GetInt64(c, "time") // 秒，封禁持续时间
		gid := uc.GetString(c, "gid")
		reason := uc.GetString(c, "reason")
		if err != nil || gid == "" {
			c.JSON(401, map[string]interface{}{
				"ok":     "false",
				"reason": err.Error(),
			})
			return
		}

		tilogs.L().Debugf("BanUser uid=%s gid=%s ban_time=%d reason=%s", uid, gid, ban_time, reason)
		err = BanUser(uid, gid, ban_time, reason)
		if err != nil {
			tilogs.L().Errorf("UserMgrController BanUser err %v uid ", err, uid)
			c.JSON(401, map[string]interface{}{
				"ok":     "false",
				"reason": err.Error(),
			})
			return
		}

		c.JSON(200, map[string]interface{}{
			"result": "ok",
		})
		return
	}
}

func (uc *UserMgrController) IsBan() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		uid := uuid.FromStringOrNil(id)

		err, _, _, _ := models.UserBanInfo(db.UserID{UUID: uid})
		isBan := false
		if err != nil {
			isBan = true
		}
		c.JSON(200, map[string]interface{}{
			"result": "ok",
			"isBan":  isBan,
		})
		return
	}
}

// http://127.0.0.1:8789/auth/v1/user/gag/{userid}?time={timetoban}&gid={gid}
func (uc *UserMgrController) GagUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("id")
		gag_time, err := uc.GetInt64(c, "time")
		gid := uc.GetString(c, "gid")

		if err != nil || gid == "" {
			c.JSON(401, map[string]interface{}{
				"ok":     "false",
				"reason": err.Error(),
			})
			return
		}

		tilogs.L().Debugf("Gag %s %s %d", uid, gid, gag_time)
		err = GagUser(uid, gid, gag_time)
		if err != nil {
			c.JSON(401, map[string]interface{}{
				"ok":     "false",
				"reason": err.Error(),
			})
			return
		}

		c.JSON(200, map[string]interface{}{
			"result": "ok",
		})
		return
	}
}

func (uc *UserMgrController) GetShardNewUserCount() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := c.Param("id")
		n, err := models.GetShardNewUserCount(sid)
		if err != nil {
			if strings.Contains(err.Error(), "nil return") {
				c.JSON(200, map[string]interface{}{
					"result": "ok",
					"num":    "0",
				})
				return
			}
			c.JSON(401, map[string]interface{}{
				"ok":     "false",
				"reason": err.Error(),
			})
			return
		}

		c.JSON(200, map[string]interface{}{
			"result": "ok",
			"num":    fmt.Sprintf("%d", n),
		})
		return
	}
}

func notifyInfo(gid uint, uid db.UserID, info gate.InfoNotifyToClient) {
	OldAccountID, OldLoginToken,
		OldRpc, ok := models.GetLastLoginStatus(gid, uid)

	if ok { //如果这个账户已有登录信息，尝试通知退出
		var oldConn *rpc.Client
		var reply int
		var err error
		if oldConn, err = jsonrpc.Dial("tcp", OldRpc); err == nil {
			defer oldConn.Close()
			err = oldConn.Call("GateServer.NotifyInfo",
				&gate.NotifyInfoParam{
					LoginToken: OldLoginToken,
					AccountID:  OldAccountID,
					Info:       info,
				},
				&reply)
		}

		if err != nil {
			//相关服务器上RPC通知失败，则允许继续登录
			tilogs.L().Warnf("[Login.getGate.call] notifyInfo last failed. rpc conn", err.Error())
		}

	}
}

func (uc *UserMgrController) InsertOrUpdateGmUid() gin.HandlerFunc {
	return func(c *gin.Context) {

		sdkIdToUidInfo := struct {
			SdkId                   string `form:"sdk_id"`
			Uid                     string `form:"uid"`
			CurrentPlayerNumber     string `form:"current_player_number"`
			CurrentPlayerNumberDesc string `form:"current_player_number_desc"`
			TargetPlayerNumber      string `form:"target_player_number"`
			TargetPlayerNumberDesc  string `form:"target_player_number_desc"`
			ShardId                 string `form:"shard_id"`
		}{}

		if err := c.ShouldBindWith(&sdkIdToUidInfo, binding.Form); err != nil {
			tilogs.L().Errorf("bind sdkIdToUidInfo fail")
			return
		}

		shardId, err := strconv.Atoi(sdkIdToUidInfo.ShardId)
		if err != nil {
			tilogs.L().Errorf("convert shardId fail from string to int ,err is  %v", err)
		}
		sdkIdToUidInfos := models.SdkIdToUidInfos{
			CurrentPlayerNumber:     sdkIdToUidInfo.CurrentPlayerNumber,
			CurrentPlayerNumberDesc: sdkIdToUidInfo.CurrentPlayerNumberDesc,
			TargetPlayerNumber:      sdkIdToUidInfo.TargetPlayerNumber,
			TargetPlayerNumberDesc:  sdkIdToUidInfo.TargetPlayerNumberDesc,
			ShardId:                 uint(shardId),
			Uid:                     sdkIdToUidInfo.Uid,
			SdkId:                   sdkIdToUidInfo.SdkId,
		}
		sdkIdToUids, err := models.InsertOrUpdateSdkIdToUid(&sdkIdToUidInfos)

		if err != nil {
			tilogs.L().Errorf("InsertOrUpdateSdkIdToUid fail  ,err is  %v", err)
			c.JSON(401, map[string]interface{}{
				"ok":    "false",
				"error": err.Error(),
			})
		}

		c.JSON(200, sdkIdToUids)
		return
	}
}

func (uc *UserMgrController) QueryGmUid() gin.HandlerFunc {
	return func(c *gin.Context) {

		//查询操作，返回
		sdkIdToUids, err := models.QuerySdkIdToUid()
		if err != nil {
			tilogs.L().Errorf("QuerySdkIdToUid fail  ,err is  %v", err)
			c.JSON(401, map[string]interface{}{
				"ok":    "false",
				"error": err.Error(),
			})
		}

		c.JSON(200, sdkIdToUids)
		return
	}
}

func (uc *UserMgrController) DeleteGmUid() gin.HandlerFunc {
	return func(c *gin.Context) {
		numberId := uc.GetString(c, "current_player_number")
		sdkIdToUids, err := models.DeleteSdkIdToUid(numberId, 0)
		if err != nil {
			tilogs.L().Errorf("delete data from SdkIdToUid fail ,err is %v", err)
		}

		c.JSON(200, sdkIdToUids)
		return
	}
}

//白名单操作
func (uc *UserMgrController) QueryWhiteList() gin.HandlerFunc {
	return func(c *gin.Context) {
		whiteListInfos, err := models.QueryAllWhiteList()
		if err != nil {
			tilogs.L().Errorf("query data from WhiteList fail ,err is %v", err)
		}

		c.JSON(200, whiteListInfos)
		return
	}
}

func (uc *UserMgrController) DeleteWhiteList() gin.HandlerFunc {
	return func(c *gin.Context) {
		content := uc.GetString(c, "content")
		typ, err := uc.GetInt(c, "typ")
		if err != nil {
			tilogs.L().Errorf("WhiteList : get typ fail ,err is %v", err)
		}
		whiteListInfos, err := models.DeleteWhiteList(content, uint(typ))
		if err != nil {
			tilogs.L().Errorf("delete data from WhiteList fail ,err is %v", err)
		}

		c.JSON(200, whiteListInfos)
		return
	}
}

func (uc *UserMgrController) InsertOrUpdateWhiteList() gin.HandlerFunc {
	return func(c *gin.Context) {

		whiteList := struct {
			Typ        string `form:"typ"`
			Content    string `form:"content"`
			ModifyTime string `form:"modify_time"`
			SdkId      string `form:"sdk_id"`
		}{}

		if err := c.ShouldBindWith(&whiteList, binding.Form); err != nil {
			tilogs.L().Errorf("bind WhiteList fail")
			return
		}

		typ, err := strconv.Atoi(whiteList.Typ)
		if err != nil {
			tilogs.L().Errorf("convert shardId fail from string to int ,err is  %v", err)
		}
		modifyTime, err := strconv.Atoi(whiteList.ModifyTime)
		if err != nil {
			tilogs.L().Errorf("convert modifyTime fail from string to int ,err is  %v", err)
		}
		whiteListInfo := models.WhiteListInfo{
			Typ:        uint(typ),
			Content:    whiteList.Content,
			SdkId:      whiteList.SdkId,
			ModifyTime: int64(modifyTime),
		}
		whiteListInfos, err := models.InsertOrUpdateWhiteList(&whiteListInfo)

		if err != nil {
			tilogs.L().Errorf("InsertOrUpdateWhiteList fail  ,err is  %v", err)
			c.JSON(401, map[string]interface{}{
				"ok":    "false",
				"error": err.Error(),
			})
		}

		c.JSON(200, whiteListInfos)
		return
	}
}

func (uc *UserMgrController) QueryUserSpecialAward() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.DefaultQuery("id", "")
		uid := uuid.FromStringOrNil(id)
		tilogs.L().Debugf("QueryUserSpecialAward,id:%s", id)

		awardStr, err := models.QueryUserSpecialAward(db.UserID{UUID: uid})
		if err != nil {
			tilogs.L().Errorf("QueryUserSpecialAward fail  ,err is  %v", err)
			c.JSON(401, map[string]interface{}{
				"result": "false",
				"error":  err.Error(),
			})
		}
		c.JSON(200, map[string]interface{}{
			"result":   "ok",
			"awardStr": awardStr,
		})
		return
	}
}

func (uc *UserMgrController) UpdateUserSpecialAward() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := c.GetPostForm("id")
		awardStr, _ := c.GetPostForm("awardStr")
		uid := uuid.FromStringOrNil(id)
		tilogs.L().Debugf("UpdateUserSpecialAward id:%s,awardStr:%s", id, awardStr)

		err := models.UpdateUserSpecialAward(db.UserID{UUID: uid}, awardStr)
		if err != nil {
			tilogs.L().Errorf("UpdateUserSpecialAward fail  ,err is  %v", err)
			c.JSON(401, map[string]interface{}{
				"result": "false",
				"error":  err.Error(),
			})
		}
		c.JSON(200, map[string]interface{}{
			"result": "ok",
		})
		return
	}
}

func (uc *UserMgrController) BanPlayer() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("id")
		acid := uc.GetString(c, "acid")
		banTime, err := uc.GetInt64(c, "time") // 秒，封禁结束时间
		gid := uc.GetString(c, "gid")
		reason := uc.GetString(c, "reason")
		if err != nil || gid == "" {
			c.JSON(401, map[string]interface{}{
				"ok":     "false",
				"reason": err.Error(),
			})
			return
		}

		tilogs.L().Debugf("BanPlayer uid=%s acid=%s gid=%s ban_time=%d reason=%s", uid, acid, gid, banTime, reason)
		err = BanPlayer(uid, acid, gid, banTime, reason)
		if err != nil {
			c.JSON(401, map[string]interface{}{
				"ok":     "false",
				"reason": err.Error(),
			})
			return
		}

		c.JSON(200, map[string]interface{}{
			"result": "ok",
		})
		return
	}
}

func (uc *UserMgrController) IsBanPlayer() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		uid := uuid.FromStringOrNil(id)
		acid := uc.GetString(c, "acid")

		info := models.GetPlayerBanInfo(uid.String(), acid)
		isBan := false
		if info != nil {
			if info.BanEndTime > 0 && info.BanEndTime > timeutil.Now().Unix() {
				isBan = true
			}
		}
		c.JSON(200, map[string]interface{}{
			"result": "ok",
			"isBan":  isBan,
		})
		return
	}
}

// PayRewardForUser 玩家领取返利。
func (uc *UserMgrController) PayRewardForUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := c.GetPostForm("id")
		rewardInfo, _ := c.GetPostForm("rewardType")
		userID := uuid.FromStringOrNil(id)
		optType, _ := c.GetPostForm("optType")
		rewardType, _ := strconv.Atoi(rewardInfo)

		tilogs.L().Infof("UserMgrController PayRewardForUser userID: %v, rewardType: %v, optType: %v", userID, rewardInfo, optType)

		ret := &PayRewardRet{}
		info, handleCode := models.PayRewardForUser(rewardType, db.UserID{UUID: userID}, optType)
		if handleCode == models.RewardCodeOK && info != nil {
			switch rewardType {
			case models.RewardPay:
				// 充值返利。
				if payInfo, ok := info[models.RewardPay]; ok {
					ret.MoneyScore = payInfo.MoneyScore
				}
			case models.RewardLoginAndRank:
				// 登陆排名返利。
				if loginAndRankInfo, ok := info[models.RewardLoginAndRank]; ok {
					ret.LoginDay = loginAndRankInfo.LoginDay
					ret.Fair1v1Dan = loginAndRankInfo.Fair1v1Dan
					ret.Fair3v3Dan = loginAndRankInfo.Fair3v3Dan
					ret.ClaimTime = loginAndRankInfo.ClaimTime
				}
			}
		}
		ret.Result = handleCode

		c.JSON(200, ret)
		return
	}
}

//充值返利的结果
type PayRewardRet struct {
	MoneyScore int32 `json:"money_score" `                     // 充值返利-累充积分。
	LoginDay   int32 `json:"login_day" bson:"login_day"`       // 登陆排名返利-登陆天数。
	Fair1v1Dan int32 `json:"fair_1v1_dan" bson:"fair_1v1_dan"` // 登陆排名返利-1v1段位。
	Fair3v3Dan int32 `json:"fair_3v3_dan" bson:"fair_3v3_dan"` // 登陆排名返利-3v3段位。
	Result     int32 `json:"result"`                           // 返回的结果。
	ClaimTime  int64 `json:"claim_time" bson:"claim_time"`     // 领取时间。
}
