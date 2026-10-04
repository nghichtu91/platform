package gate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/nghichtu91/platform/share/x/common/msg"

	"github.com/nghichtu91/platform/share/planx/client"
	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	ver "github.com/nghichtu91/platform/share/planx/version"
)

type HandShake struct {
	LoginToken  string
	ShardID     int
	GateIP      string
	SubID       uint
	HandShakeID uint
	RandomKey   uint
}

const (
	MAX_LoginToken_Length = 512 * 1024
)

// handShake
// TODO 所有异常都应该有监控并搜集玩家IP，防止恶意攻击服务器IP快速被发现 1095
func (g *GateServer) handShake(con net.Conn, agent *client.PacketConnAgent) (
	success bool, ln LoginNotify, gzipLimit uint64, sessionId int64, clientInfo string) {
	success = false
	clientID := con.RemoteAddr().String()
	IpAddr, _, _ := net.SplitHostPort(clientID)

	con.SetReadDeadline(time.Now().Add(time.Second * 15))
	cnc := &util.ConnNopCloser{C: con}

	// 防止恶意客户端发送超过512字符串的命令
	limit_reader := io.LimitReader(cnc, MAX_LoginToken_Length)
	reader := bufio.NewReader(limit_reader)
	tr := textproto.NewReader(reader)
	writer := bufio.NewWriter(cnc)

	// 建立链接后的第一个消息是客户端发过来的loginToken，否则断开链接
	tilogs.L().Debugf("<Gate> client [%s], handshake before ReadLine \n", clientID)
	line, err := tr.ReadLine()
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		tilogs.L().Warnf("<Gate> Alarm: Gate Server try to make a handshake failed with client [%s] ,error [%s]", IpAddr, err.Error())
		send_fail(writer)
		return
	}
	tilogs.L().Debugf("<Gate> client [%s], handshake got loginToken:%s\n", clientID, line)
	loginToken := line
	tilogs.L().Debugf("<Gate> client %s, queryUserLoginInfo loginToken:%s line:%s\n", clientID, loginToken, line)

	realLoginToken, sid := getRealLoginToken(loginToken)
	if realLoginToken == "" || sid <= 0 {
		tilogs.L().Warnf("<Gate> Alarm: Gate Server try to make a handshake with client [%s], loginToken illegal [%s]", IpAddr, loginToken)
		send_fail(writer)
		return
	}
	tilogs.L().Debugf("<Gate> client %s, queryUserLoginInfo realLoginToken:%s ", clientID, realLoginToken)
	loginInfo := g.getShardLoginInfo(sid)
	if loginInfo == nil {
		// 避免servercheck产生冗余数据
		if strconv.Itoa(int(sid)) != consts.FakeShardID {
			tilogs.L().Warnf("<Gate> Alarm: Gate Server try to make a handshake with client [%s], shard not found %d", IpAddr, sid)
		}
		send_fail(writer)
		return
	}
	uInfo, sessionId := loginInfo.queryUserLoginInfo(realLoginToken, agent)
	if uInfo == nil {
		tilogs.L().Warnf("<Gate> Alarm: Gate Server try to make a handshake with client [%s], failed with [%s]", IpAddr, "queryUserLoginInfo")
		send_fail(writer)
		return
	}

	ln = *uInfo
	uid := uInfo.UserId
	// gid := uInfo.GameId
	// sid := uInfo.ShardId
	accountID := ln.String() // makeAccountID(gid, sid, uid)
	accountNumberID := ln.NumberID
	_log := tilogs.L().WithUser(accountID).With("sessionid", sessionId).With("clientip", con.RemoteAddr().String())
	_log.Infof("handShake after queryUserLoginInfo")

	gziplimit, err := tr.ReadLine()
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake gzipLimit failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	gzipLimit, err2 := strconv.ParseUint(gziplimit, 10, 64)
	if err2 != nil {
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake gzipLimit strconv.ParseUint failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err2.Error())
		send_fail(writer)
		return
	}
	_channelId, err := tr.ReadLine()
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake channelId failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_bdcId, err := tr.ReadLine() // bdcid，来自客户端
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake bdcId failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_version, err := tr.ReadLine() // 客户端版本，来自客户端
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake version failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_deviceType, err := tr.ReadLine() // 客户端手机机型
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake deviceType failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_deviceSystem, err := tr.ReadLine() // 客户端手机系统
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake deviceSystem failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_updateVer, err := tr.ReadLine() // 客户端热更的版本
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake updateVer failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_IP, err := tr.ReadLine() // 客户端热更的版本
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake ip failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_IDFA, err := tr.ReadLine() // 客户端广告id
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake IDFA failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_memory, err := tr.ReadLine() // 客户端设备内存
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake memory failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_addressFlag, err := tr.ReadLine() // 客户端设备内存
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake addressFlag failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_deviceId, err := tr.ReadLine() // deviceId
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake deviceId failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_appChannelId, err := tr.ReadLine()
	if err != nil {
		// appChannelId 子渠道channelId
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake AppChannelId failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_sdkChannelUid, err := tr.ReadLine()
	if err != nil {
		// sdkUid (目前适用于英雄sdk)
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake SdkChannelUid failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}

	var _bdcCommonInfo string
	var index int
	for {
		newBdcCommonInfo, err := tr.ReadLine()
		if err != nil {
			// sdkUid (目前适用于英雄sdk)
			_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake newBdcCommonInfo failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
			send_fail(writer)
			return
		}

		_log.Debugf("handShake newBdcCommonInfo %s", newBdcCommonInfo)
		if (newBdcCommonInfo == "" || newBdcCommonInfo == consts.RobotChannel) && index == 0 {
			_bdcCommonInfo = ""
			break
		}

		index++
		_bdcCommonInfo = _bdcCommonInfo + newBdcCommonInfo
		if strings.Contains(newBdcCommonInfo, "}") {
			break
		}

		if index > 100 {
			_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake newBdcCommonInfo failed loop > 100 with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
			send_fail(writer)
			return
		}
	}

	_log.Debugf("handShake _bdcCommonInfo %s", _bdcCommonInfo)
	_language, err := tr.ReadLine() // 语言
	if err != nil {
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake language failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}
	_clientBuildTimeAndHotVer, err := tr.ReadLine() // clientBuildTimeAndHotVer
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake clientBuildTimeAndHotVer failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}

	_BDCDeviceId, err := tr.ReadLine() // BDCDeviceId
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake BDCDeviceId failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}

	_SubChannelID, err := tr.ReadLine() // BDCDeviceId
	if err != nil {
		// 需要实现恶意检测ip计数报告, 判断超出长度情况
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake SubChannelID failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}

	_clientPlatform, err := tr.ReadLine()
	if err != nil {
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake ClientPlatform failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}

	_distinctID, err := tr.ReadLine()
	if err != nil {
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake DistinctID failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}

	// 这行必须是DEBUG或RELEASE
	_clientType, err := tr.ReadLine()
	if err != nil || (_clientType != consts.ClientTypeDebug && _clientType != consts.ClientTypeRelease) {
		_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake clientType failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
		send_fail(writer)
		return
	}

	// 客户端把ip带过来
	info := msg.ClientHandShakeInfo{
		ChannelId:                _channelId,
		BDCId:                    _bdcId,
		Version:                  _version,
		DeviceType:               _deviceType,
		DeviceSystem:             _deviceSystem,
		UpdateVer:                _updateVer,
		IP:                       _IP,
		IDFA:                     _IDFA,
		DeviceMemory:             _memory,
		AddressFlag:              _addressFlag,
		DeviceId:                 _deviceId,
		AppChannelId:             _appChannelId,
		SdkChannelUid:            _sdkChannelUid,
		BdcCommonInfo:            _bdcCommonInfo,
		Language:                 _language,
		ClientBuildTimeAndHotVer: _clientBuildTimeAndHotVer,
		BDCDeviceId:              _BDCDeviceId,
		SubChannelID:             _SubChannelID,
		ClientPlatform:           _clientPlatform,
		DistinctID:               _distinctID,

		SdkId:              ln.SdkDeviceId,
		AccountNumberId:    accountNumberID,
		RechargeMoney:      ln.RechargeMoney,
		RechargeReturn:     ln.RechargeReturn,
		LoginDay:           ln.LoginDay,
		Fair1v1Dan:         ln.Fair1v1Dan,
		Fair3v3Dan:         ln.Fair3v3Dan,
		LoginAndRankReward: ln.LoginAndRankReward,
		ClaimTime:          ln.ClaimTime,
		IsSuper:            ln.IsSuper,
	}

	// 标记是否为机器人
	if _channelId == consts.RobotChannel {
		agent.IsRobot = true
	}

	_log.Infof("AddressFlag : %s", _addressFlag)
	_clientInfo, err := json.Marshal(info)
	if err != nil {
		_log.Errorf("<Gate> clienthandshakeinfo json.Marshal err %s", err.Error())
		send_fail(writer)
		return
	}
	clientInfo = string(_clientInfo)

	reserved := 3
	for reserved > 0 {
		reserved--
		reline, err := tr.ReadLine() // reserved line 1
		if err != nil {
			// 需要实现恶意检测ip计数报告, 判断超出长度情况
			_log.Warnf("<Gate> Alarm: Gate Server try to make a handshake reserved failed with accountID [%s] client [%s] ,error [%s]", accountID, IpAddr, err.Error())
			send_fail(writer)
			return
		}
		if !strings.HasPrefix(reline, "reserved line") {
			_log.Errorf("<Gate> Alarm: Gate Server try to make a handshake reserved failed with accountID [%s] client [%s] ,reline [%s]", accountID, IpAddr, reline)
			send_fail(writer)
			return
		}
	}

	// XXX: 这里的修改需要配合UnitySDK修改和botx的handshake模拟的修改 by YZH
	if uid == db.InvalidUserID {
		_log.Warnf("<Gate> Alarm: Gate Server can't find loginToken for current user! Connection(%s) will be closed. login token: %s, accountID [%s]", IpAddr, loginToken, accountID)
		send_fail(writer)
		return
	} else {
		fmt.Fprintf(writer, "ok\r\n")
		// should pass 0:0:1001
		fmt.Fprintf(writer, "%s\r\n", secure.Encode64ForNet([]byte(accountID)))
		fmt.Fprintf(writer, "%d\r\n", timeutil.NowByShardId(sid).Unix()) // 服务器时间
		fmt.Fprintf(writer, "%s\r\n", ver.GetVersion())
		fmt.Fprintf(writer, "%s\r\n", secure.Encode64ForNet([]byte(strconv.FormatInt(accountNumberID, 10))))
		writer.Flush()
		success = true
		_log.Debugf("<Gate> Gate Server make a good handshake with client. client:%s user_id:%s, login token:%s, clieninfo %+v",
			IpAddr, uid.String(), loginToken, info)
		return
	}
}

func send_fail(w *bufio.Writer) {
	fmt.Fprintf(w, "fail\r\n")
	w.Flush()
}
