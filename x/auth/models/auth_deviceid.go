package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"

	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func makeAuthUidKey(uid db.UserID) string {
	return fmt.Sprintf("uid:%s", uid)
}

// TODO 增加deviceUserInfo field(ban,gag, ...)
type DeviceUserInfo struct {
	UserId     db.UserID `json:"user_id"`
	Display    string    `json:"dn,omitempty"`
	Name       string    `json:"name,omitempty"`
	LastTime   int64     `json:"lasttime"`
	CreateTime int64     `json:"createtime"`
	ChannelId  string    `json:"channel_id"`
}

// checkDevice 数据查询，检查是否存在当前deviceID的信息
// 返回值：
// - (deviceInfo， nil) 查询到，正常结束
// - (xxxx, error) 查询过程失败，返回错误信息
func checkDevice(deviceID string) (*DeviceUserInfo, error, bool) {
	deviceinfo, err, isGM := db_interface.GetDeviceInfo(deviceID, true)
	switch err {
	case XErrDBNotFound:
		return nil, XErrChkDeviceNotFound, isGM
	case nil:
		tilogs.L().Debugf("Check Device:%v", deviceinfo)
		db_interface.UpdateDeviceInfo(deviceID, deviceinfo)
		return deviceinfo, nil, isGM
	default:
		return nil, err, isGM
	}
}

type UserInfoToClient struct {
	GagTime int64 `json:"g"`
}

// AuthUserPass ...
// 返回逻辑错误：XErrAuthUsernameNotFound，XErrAuthUserPasswordInCorrect
func AuthUserPass(ctx context.Context, name, passwd string) (string, db.UserID, *UserInfoToClient, int64, string, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "AuthUserPass")
	defer span.Finish()

	uid, err, isGM := db_interface.GetUnKeyV2(ctx, name, true)
	res_info := &UserInfoToClient{}
	if err != nil {
		return "", db.InvalidUserID, res_info, 0, "", err
	}

	var authToken string
	if !isGM {
		dbpasswd := fmt.Sprintf("%x", secure.DefaultEncode.PasswordForDB(passwd))

		info, err := db_interface.GetUnInfoV2(ctx, uid)
		if err != nil {
			return "", db.InvalidUserID, res_info, 0, "", XErrAuthUsernameNotFound
		}
		name_in_db := info.Name
		pswd_in_db := info.PassWord
		ban_time := info.BanTime
		gag_time := info.GagTime
		ban_reason := info.Reason
		if name_in_db != name || pswd_in_db != dbpasswd {
			return "", db.InvalidUserID, res_info, 0, "", XErrAuthUserPasswordInCorrect
		}
		if ban_time > 0 && time.Now().Unix() <= ban_time {
			return "", db.InvalidUserID, res_info, ban_time, ban_reason, XErrByBan
		}
		authToken, err = AuthToken(ctx, uid, "")
		res_info.GagTime = gag_time
	} else {
		tilogs.L().Infof("GM login by %s %s", name, uid.String())
		authToken, err = AuthToken(ctx, uid, "")
	}

	return authToken, uid, res_info, 0, "", err
}

func UserBanInfo(uid db.UserID) (error, int64, string, int64) {
	info, err := db_interface.GetUnInfo(uid)
	if err != nil {
		return err, 0, "", 0
	}
	ban_time := info.BanTime
	ban_reason := info.Reason
	gag_time := info.GagTime
	tilogs.L().Debugf(" UserBanInfo user %v ban_time %v,gag_time %v", uid, ban_time, gag_time)
	if ban_time > 0 && time.Now().Unix() <= ban_time {
		return XErrByBan, ban_time, ban_reason, gag_time
	}
	if gag_time > 0 && time.Now().Unix() <= gag_time {
		return nil, ban_time, ban_reason, gag_time
	}
	return nil, 0, "", 0
}

func GetPlayerBanInfo(uid string, acid string) *BanPlayerInfo {
	return db_interface.GetPlayerBanInfo(uid, acid)
}

// AuthToken ...
// 返回DB级error
func AuthToken(ctx context.Context, uid db.UserID, deviceID string) (string, error) {
	authToken := uuid.NewV4().String()
	err := db_interface.UpdateUnInfoV2(ctx, uid, deviceID, authToken)
	if err != nil {
		tilogs.L().Errorf("AuthToken UpdateUnInfo err %s", err.Error())
		return "", err
	}
	return authToken, nil
}

func SetBanUnInfo(uid string, time_to_ban int64, reason string) error {
	return db_interface.UpdateBanUn(uid, time_to_ban, reason)
}

func SetGagUnInfo(uid string, time_to_ban int64) error {
	return db_interface.UpdateGagUn(uid, time_to_ban)
}

func SetBanPlayer(uid string, acid string, banTime int64, reason string) error {
	return db_interface.UpdateBanPlayer(uid, acid, banTime, reason)
}

var (
	XErrUserExistButNeedPassword = errors.New("User account exists but need login with user/pass only")
	XErrUserNotExist             = errors.New("User account does not exist.")
)

func TryToCheckDeviceInfo(DeviceID string) (*DeviceUserInfo, error, bool) {

	di, err, isGM := checkDevice(DeviceID)
	if err != nil {
		switch err {
		case XErrChkDeviceNotFound:
			//可以创建新帐号
			return nil, XErrUserNotExist, isGM
		default:
			return nil, err, isGM
		}
	}
	//err == nil && di != nil
	if di != nil {
		//找到匿名帐号
		uid := di.UserId
		if uid.IsValid() {
			if di.Name != "" {
				//要求强制用用户名密码登录
				return di, XErrUserExistButNeedPassword, isGM
			}
			//可以匿名登录
			return di, nil, isGM
		}
	}

	return nil, fmt.Errorf("TryToCheckDeviceInfo got unknown errors."), isGM
}

// AddDevice 创建新UUID的然后返回uid
// error 返回数据库级和语言类错误
func AddDevice(DeviceID, Name, channelId, device string) (*DeviceUserInfo, error) {
	//num, err := db_interface.IncrDeviceTotal()
	//if err != nil {
	//return nil, err
	//}

	//uid := db.UserID(num + 1000)
	uid := db.NewUserID()
	value, err := db_interface.SetDeviceInfo(DeviceID, Name, uid, channelId, device)
	if err != nil {
		return nil, err
	}
	allmetrics.AddAuthRegisterCount()
	return value, nil
}

var (
	XErrRegPwdUserNameUsed = errors.New("User name has been used.")
	XErrRegPwdUserHasName  = errors.New("You've already have user name registered.")
)

// RegisterWithPassword 注册用户email，当前user_id同用户名密码绑定
// 返回逻辑错误：XErrRegPwdUserNameUsed, XErrRegPwdUserHasName， 数据库级错误
func RegisterWithPassword(name, passwd, email, deviceID string) (db.UserID, error) {
	// 检测注册用户名, 已经有人使用
	isexist, err := db_interface.IsNameExist(name)
	if err != nil {
		return db.InvalidUserID, err //db error
	}
	if isexist == 1 {
		return db.InvalidUserID, XErrRegPwdUserNameUsed
	}
	var creatNewAccount bool
	var bindAccount bool
	di, err, _ := checkDevice(deviceID)
	if err != nil {
		switch err {
		case XErrChkDeviceNotFound:
			creatNewAccount = true
			bindAccount = true
			err = nil
		default:
			//数据库级错误
			tilogs.L().Errorf("Check Device Error:", err.Error())
		}
	} else if di != nil {
		//设备ID存在，检查是否已有用户名
		if di.UserId.IsValid() && di.Name != "" {
			err = XErrRegPwdUserHasName
		} else {
			//bind user name
			bindAccount = true
		}
	}

	if err != nil {
		return db.InvalidUserID, err
	}
	if creatNewAccount && bindAccount {
		//create new
		//检查是否已经存在
		var errn error
		di, errn = AddDevice(deviceID, name, "", "")
		if errn != nil {
			//返回数据库级和语言类错误
			tilogs.L().Errorf("Check Device Error:", errn.Error())
			return db.InvalidUserID, errn
		}
	} else if bindAccount {
		//bind
		tilogs.L().Warnf("[RegisterWithPassword] bindAccount %s %s", deviceID, name)
	} else {
		return db.InvalidUserID, err
	}

	uid := di.UserId

	//更新deviceid -> user_id的信息，方便处理强制有密码用户用密码登录
	/*
		info.Name = name

		deviceInfo, err := json.Marshal(info)
		if err != nil {
			return -1, err
		}
		_, err = db.Do("SET", deviceID, deviceInfo)
		if err != nil {
			return -2, err
		}
	*/

	//构建用户名索引，用户用户密码登录检索用
	err = db_interface.SetUnKey(name, uid)
	if err != nil {
		return db.InvalidUserID, err
	}

	//构建注册用户数据
	err = db_interface.SetUnInfo(uid, name, deviceID, passwd, email, "")
	if err != nil {
		return db.InvalidUserID, err
	}

	return uid, nil
}

func AuthUserInfo(uid string) (string, db.UserID, *UserInfoToClient, int64, string, string, error) {
	userId := db.UserIDFromStringOrNil(uid)
	userInfo, err := db_interface.GetUnInfo(userId)
	infoToClient := &UserInfoToClient{}

	if err != nil {
		return "", db.InvalidUserID, infoToClient, 0, "", "", err
	}

	ban_time := userInfo.BanTime
	gag_time := userInfo.GagTime
	ban_reason := userInfo.Reason

	if ban_time > 0 && time.Now().Unix() <= ban_time {
		return "", db.InvalidUserID, infoToClient, ban_time, ban_reason, "", XErrByBan
	}
	authToken, err := AuthToken(context.Background(), userId, "")
	if err != nil {
		return "", db.InvalidUserID, infoToClient, 0, "", "", err
	}
	infoToClient.GagTime = gag_time

	return authToken, userId, infoToClient, 0, "", userInfo.DeviceID, err
}

type UserInfo struct {
	Name     string
	PassWord string
	BanTime  int64
	GagTime  int64
	Reason   string
	DeviceID string
	RecallID string
}

// cheat修改账号密码
func ResetPwdByCheat(ctx context.Context, name, passwd string) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "ResetPwdByCheat")
	defer span.Finish()

	// 根据name获取uid
	uid, err, _ := db_interface.GetUnKeyV2(ctx, name, true)
	if err != nil {
		return err
	}

	//构建注册用户数据
	info, err := db_interface.GetUnInfoV2(ctx, uid)
	if err != nil {
		return XErrAuthUsernameNotFound
	}

	// 模拟客户端加密解密过程
	dbpasswd := secure.DefaultEncode.PwdEncode(passwd)
	md5pwd, err := secure.DefaultEncode.Decode64FromNet(dbpasswd)
	if err != nil {
		return err
	}
	// 重置密码
	pwd := fmt.Sprintf("%x", secure.DefaultEncode.PasswordForDB(string(md5pwd)))
	tilogs.L().Debugf("ResetPwdByCheat uid:%s name:%s deviceID:%s passwd:%s pwd:%v", uid, name, info.DeviceID, passwd, pwd)
	err = db_interface.SetUnInfoPass(uid, name, info.DeviceID, pwd, "", "")
	if err != nil {
		return err
	}
	return nil
}
