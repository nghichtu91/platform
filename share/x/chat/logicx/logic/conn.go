package logic

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego/httplib"
	"github.com/nghichtu91/platform/share/planx/secure"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/chat/api/const_value"
	"github.com/nghichtu91/platform/share/x/chat/api/json_define"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/logicx/config"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

// Connect connected a conn.
func (l *Logic) Connect(c context.Context, server, key, token, sessionId string) (err error) {
	var (
		origins string
		buf     []byte
		dbTime  int
	)

	// 用于server_check
	// 会进行一次数据库操作以验证可用性
	if key == consts.RobotCometxID && token == consts.RobotCometxToken {
		if err = l.dao.SetPlayerMapping(key, token); err != nil {
			log.L().Errorf("l.dao.SetPlayerCometId server check failed, err %s", err.Error())
		}
		return
	}

	// 如果是机器人的话 可以不需要正常的登陆流程
	if !config.IsRobotNeedLogin && token == const_value.RobotToken {
		if err = l.dao.SetPlayerMapping(key, fmt.Sprintf("%s,%s", server, sessionId)); err != nil {
			log.L().Errorf("l.dao.SetPlayerCometId(%s,%s) error(%v)", key, server, err)
		}
		return
	}

	// 登录验证
	if origins, err = l.dao.GetPlayerToken(key); err != nil {
		log.L().Warnf("logic server: Connect key not exist, key:%s", key)
		return
	}

	// 检测token是否过期
	originSlice := strings.Split(origins, ",")
	if len(originSlice) != 2 {
		log.L().Warnf("logic server: Connect dbToken split failed, db data:%s", originSlice)
		return errors.New(pb.ErrorCode_SomethingError.String())
	}
	if buf, err = secure.Decode64FromNet(token); err != nil {
		log.L().Warnf("logic server: Connect dbToken split failed, key:%s, token:%s", key, token)
		return errors.New(pb.ErrorCode_SomethingError.String())
	}

	// 验证token
	dbToken := originSlice[0]
	if dbToken != string(buf) {
		log.L().Warnf("logic server: Connect token error, token:%s, dbToken:%s", token, dbToken)
		return errors.New(pb.ErrorCode_SomethingError.String())
	}

	// 验证时间
	dbTimeString := originSlice[1]
	dbTime, err = strconv.Atoi(dbTimeString)
	if int64(dbTime+config.Cfg.Auth.MaxAuthTimeOut) < time.Now().Unix() {
		return errors.New(pb.ErrorCode_SomethingError.String())
	}

	if err = l.dao.SetPlayerMapping(key, fmt.Sprintf("%s,%s", server, sessionId)); err != nil {
		log.L().Errorf("l.dao.SetPlayerCometId(%s,%s,%s) error(%v)", key, server, sessionId, err)
		return
	}

	log.L().Infof("conn connected key:%s server:%s session:%s", key, server, sessionId)
	return nil
}

// Disconnect disconnect a conn.
func (l *Logic) Disconnect(c context.Context, key, server, sessionId string) (has bool, err error) {
	log.L().Infof("conn disconnected key:%s server:%s session:%s", key, server, sessionId)
	if err = l.dao.DelPlayerMapping(key, sessionId); err != nil {
		log.L().Warnf("l.dao.DelMapping(%s) error(%v), maybe already del, because gate or test client will del", key, server)
		return false, err
	}
	return true, err
}

func (l *Logic) GetForbiddenInfo(c context.Context, key string) (reason string, endTime int64, err error) {
	sendMsgRet := &json_define.SendMsgRet{}
	sendMsgReq := httplib.Post(fmt.Sprintf("%s/router/v1/sendMsg", config.Cfg.GmServerUrl))
	sendMsgReq.SetTimeout(
		time.Second*time.Duration(config.Cfg.GmServer.ConnectTimeout),
		time.Second*time.Duration(config.Cfg.GmServer.ReadWriteTimeout))

	serverZone := config.Cfg.Gid
	sendMsgReq.Param("serverZone", strconv.Itoa(int(serverZone)))
	sendMsgReq.Param("uid", key)

	err = sendMsgReq.ToJSON(sendMsgRet)
	if err != nil {
		return
	}

	if sendMsgRet.Code != 0 {
		log.L().Warnf("request to gm server failed. code=%d", sendMsgRet.Code)
		return "", 0, fmt.Errorf("request to gm server failed%d", sendMsgRet.Code)
	}

	reason = sendMsgRet.Reason
	endTime = sendMsgRet.EndTime

	/*
		// 可能就没有
		if err != nil {
			log.L().Infof("l.dao.GetForbiddenInfo(%s) error(%v)", key, err)
			reason = ""
			endTime = 0
			return
		}
	*/

	// 禁言结束
	if endTime < time.Now().Unix() && endTime != int64(const_value.ForbiddenForever) {
		//l.dao.DelPlayerForbidden(key)
		reason = ""
		endTime = 0
	}

	return
}
