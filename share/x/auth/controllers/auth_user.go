package controllers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx/secure"
	"github.com/nghichtu91/platform/share/planx/servers/db"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/auth/errorctl"
	"github.com/nghichtu91/platform/share/x/auth/logic"
	"github.com/nghichtu91/platform/share/x/auth/models"
)

var validID = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]{5}`)

func isUserNameFormatOk(userName string) bool {
	if nl := len(userName); nl < 6 || nl > 128 ||
		!validID.MatchString(userName) {
		return false
	}
	return true
}

// LoginAsUser is used for login existing users
func (uc *AuthController) LoginAsUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		tilogs.L().Debugf("rev LoginAsUser")

		// tracking
		span := opentracing.GlobalTracer().StartSpan("LoginAsUser")
		span.SetTag("role", "root")
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		name := uc.GetString(c, "name")     // 客户端传过来的是经过base64(name)的
		passwd := uc.GetString(c, "passwd") // 客户端应该传过来的是pwdbase64(md5(passwd))

		namedb, errDec1 := base64.URLEncoding.DecodeString(name)
		md5pwd, errDec2 := secure.DefaultEncode.Decode64FromNet(passwd)

		if !isUserNameFormatOk(string(namedb)) {
			errorctl.CtrlErrorReturn(c, "[Auth.Device]",
				fmt.Errorf("err: username(%s) is illegal", namedb),
				errorctl.FailForClient, errorctl.ClientErrorUsernameRegFormatIllegal)
			return
		}

		if errDec1 != nil || errDec2 != nil {
			errorctl.CtrlErrorReturn(c, "[Auth.Device]",
				fmt.Errorf("err1: %v, err2 %v", errDec1, errDec2),
				errorctl.FailForClient, errorctl.ClientErrorFormatVerifyFailed)
			return
		}

		authToken, uid, info_to_client, ban_time, ban_reason, err := models.AuthUserPass(ctx, strings.ToLower(string(namedb)), string(md5pwd))
		if err != nil {
			switch err {
			case models.XErrAuthUsernameNotFound:
				errorctl.CtrlErrorReturn(c, "[Auth.Device]", err, errorctl.FailForClient, errorctl.ClientErrorUsernameNotFound)
			case models.XErrAuthUserPasswordInCorrect:
				errorctl.CtrlErrorReturn(c, "[Auth.Device]", err, errorctl.FailForClient, errorctl.ClientErrorUserPasswordIncorrect)
			case models.XErrByBan:
				errorctl.CtrlBanReturn(c, "[Auth.Device]", err, errorctl.FailForClient, errorctl.ClientErrorBanByGM, ban_time, ban_reason)
			}
			return
		}

		shardHasRole, err := models.GetUserShardHasRole(ctx, uid.String())
		tilogs.L().Debugf("auth user get user shard hasrole : %v", shardHasRole)
		if err != nil {
			tilogs.L().Errorf("[LoginAsUser] login GetUserShard err: %v", err)
			shardHasRole = []models.ShardHasRole{}
		}

		lastShard, err := models.GetUserLastShard(ctx, uid.String())
		if err != nil {
			tilogs.L().Warnf("[quick] login GetUserLastShard err: %v", err)
		}

		err, _, _, gagTime := models.UserBanInfo(uid)
		if err != nil {
			tilogs.L().Warnf("[quick] login UserBanInfo err : %v", err)
		}
		logic.LogLogin(uid.String(), strings.ToLower(string(namedb)), "", "", false)

		c.JSON(200, map[string]interface{}{
			"result":    "ok",
			"authtoken": authToken, // Auth token 返回给客户端
			"info":      *info_to_client,
			"shardrole": shardHasRole,
			"lastshard": lastShard,
			"uid":       uid.String(), // uid 返回给客户端
			"gag_time":  gagTime,      // 禁言时间戳
		})

		sdkDeviceId, _ := models.GetSdkDeviceIdByUid(uid.String())
		// Auth token 发送到登录校验服务器，并设置有效时间
		notifyLoginServer(ctx, authToken, uid, sdkDeviceId)
	}
}

// RegisterAndLogin : register a user with email and password,
// and login at the same time
func (uc *AuthController) RegisterAndLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		tilogs.L().Debugf("rev RegisterAndLogin")
		deviceID := c.Param("id")
		name := uc.GetString(c, "name")     // 客户端传过来的是经过base64(name)的
		passwd := uc.GetString(c, "passwd") // 客户端应该传过来的
		// 是pwdbase64(md5(passwd))过的
		email := uc.GetString(c, "email")

		// 传输解密过程
		byte_name, errDec1 := base64.URLEncoding.DecodeString(name)
		byte_md5pwd, errDec2 := secure.DefaultEncode.Decode64FromNet(passwd)

		if errDec1 != nil || errDec2 != nil {
			errorctl.CtrlErrorReturn(c, "[Auth.Device]",
				fmt.Errorf("err1: %v, err2 %v", errDec1, errDec2),
				errorctl.FailForClient, errorctl.ClientErrorFormatVerifyFailed)
			return
		}

		namedb := strings.ToLower(string(byte_name))
		passwddb := string(byte_md5pwd)

		// 用户名密码必须符合规范
		if !isUserNameFormatOk(namedb) {
			errorctl.CtrlErrorReturn(c, "[Auth.Device]",
				fmt.Errorf("err: username(%s) is illegal", namedb),
				errorctl.FailForClient, errorctl.ClientErrorUsernameRegFormatIllegal)
			return
		}

		// 注册用户名密码
		uid, err := models.RegisterWithPassword(namedb, passwddb, email, deviceID)
		if err != nil {
			switch err {
			case models.XErrRegPwdUserNameUsed:
				errorctl.CtrlErrorReturn(c, "[Auth.Device]", err, errorctl.FailForClient, errorctl.ClientErrorUsernameHasBeenUsed)
			case models.XErrRegPwdUserHasName:
				errorctl.CtrlErrorReturn(c, "[Auth.Device]", err, errorctl.FailForClient, errorctl.ClientErrorDeviceAlreadyBinded)
			default:
				tilogs.L().Errorf("[RegisterAndLogin][RegisterWithPassword] name %s err %s", namedb, err.Error())
			}
			return
		}

		// 使用uuid生成auth token
		authToken, err := models.AuthToken(context.Background(), uid, "")
		if err != nil {
			// AuthToken只返回DB级Error
			errorctl.CtrlErrorReturn(c, "[Auth.Device]", err, errorctl.FailForClient, errorctl.ClientErrorMaybeDBProblem)
			return
		}

		c.JSON(200, map[string]interface{}{
			"result": "ok",
			// "expire":    60 * 5,    //5 min
			"uid":       uid.String(),
			"authtoken": authToken, // Auth token 返回给客户端
		})
		tilogs.L().Debugf("[RegisterAndLogin] ret success, %s", namedb)

		// Auth token 发送到登录校验服务器，并设置有效时间
		notifyLoginServer(context.Background(), authToken, uid, deviceID)
	}
}

func (uc *AuthController) LoginByUid() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")

		authToken, userId, info_to_client, ban_time, ban_reason, _, err := models.AuthUserInfo(uid)
		if err != nil {
			switch err {
			case models.XErrAuthUsernameNotFound:
				errorctl.CtrlErrorReturn(c, "[Auth.Device]", err, errorctl.FailForClient, errorctl.ClientErrorUsernameNotFound)
			case models.XErrAuthUserPasswordInCorrect:
				errorctl.CtrlErrorReturn(c, "[Auth.Device]", err, errorctl.FailForClient, errorctl.ClientErrorUserPasswordIncorrect)
			case models.XErrByBan:
				errorctl.CtrlBanReturn(c, "[Auth.Device]", err, errorctl.FailForClient, errorctl.ClientErrorBanByGM, ban_time, ban_reason)
			}
			return
		}

		shardHasRoleStr := []string{}
		shardHasRole, err := models.GetUserShardHasRole(context.Background(), userId.String())
		if err != nil {
			tilogs.L().Errorf("[LoginAsUser] login GetUserShard err: %v", err)
			shardHasRole = []models.ShardHasRole{}
		} else {
			for _, st := range shardHasRole {
				shardHasRoleStr = append(shardHasRoleStr, st.Shard)
			}
		}

		lastShard, err := models.GetUserLastShard(context.Background(), userId.String())
		if err != nil {
			tilogs.L().Errorf("[quick] login GetUserLastShard err: %v", err)
		}

		c.JSON(200, map[string]interface{}{
			"result": "ok",
			// "expire":    60 * 5,    //5 min
			"authtoken":     authToken, // Auth token 返回给客户端
			"info":          *info_to_client,
			"shardrole":     shardHasRoleStr,
			"shardrolev250": shardHasRole,
			"lastshard":     lastShard,
		})

		// Auth token 发送到登录校验服务器，并设置有效时间
		notifyLoginServer(context.Background(), authToken, userId, "")
	}
}

// http://127.0.0.1:8789/auth/v1/user/isHasReg
func (uc *AuthController) IsHasReg() gin.HandlerFunc {
	return func(c *gin.Context) {
		r := struct {
			Uid string `json:"uid"`
		}{
			"",
		}

		err := c.Bind(&r)

		if err != nil {
			c.String(401, err.Error())
			tilogs.L().Debugf("IsHasReg Err By %s", err.Error())
			return
		}

		acc, err := db.ParseAccount(r.Uid)
		if err != nil {
			c.String(402, err.Error())
			return
		}

		userShard := []string{}
		userShardSt, err := models.GetUserShardHasRole(context.Background(), acc.UserId.String())
		if err != nil {
			tilogs.L().Errorf("[quick] login GetUserShard err: %v", err)
			c.String(403, err.Error())
			return
		} else {
			for _, st := range userShardSt {
				userShard = append(userShard, st.Shard)
			}
		}

		if userShard == nil || len(userShard) == 0 {
			c.String(200, string("no"))
		} else {
			shardName, err := models.GetShardName(acc.ShardId, acc.GameId)
			tilogs.L().Debugf("GetUserShardHasRole %s %v", shardName, userShard)
			if err != nil {
				c.String(404, err.Error())
				return
			}
			for _, s := range userShard {
				if s == shardName {
					c.String(200, string("yes"))
					return
				}
			}
			c.String(200, string("no"))
		}

	}
}

func (uc *AuthController) TestSentry() gin.HandlerFunc {
	return func(c *gin.Context) {
		panic("########### test sentry panic ... ")
	}
}

func (uc *AuthController) EchoIp() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		clientIPonly := strings.Split(clientIP, ":")[0]
		c.JSON(200, clientIPonly)
	}
}

func (uc *AuthController) ResetPasswordByCheat() gin.HandlerFunc {
	return func(c *gin.Context) {
		tilogs.L().Debugf("rev ResetPasswordByCheat")

		// tracking
		span := opentracing.GlobalTracer().StartSpan("ResetPasswordByCheat")
		span.SetTag("role", "root")
		ctx := opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		// 读取csv文件,获取用户名
		filePath := "/opt/supervisor/auth001/conf/" // 指定 csv 文件路径
		err, record := GetCsvInfo(filePath, "anchor_passwd.csv")
		if err != nil {
			tilogs.L().Errorf("ResetPasswordByCheat OpenFile err: %v", err)
			c.String(403, err.Error())
			return
		}
		// 需修改密码的账号名单
		nameList := make([]string, 0, 100)
		for _, row := range record {
			for _, val := range row {
				val = strings.TrimSpace(val)
				if val == "," || val == "Name" || val == "" {
					continue
				}
				nameList = append(nameList, val)
			}
		}
		// 创建一个新的文件
		pwdFile, err := os.Create(filepath.Join(filePath, "new_anchor_passwd.csv"))
		defer pwdFile.Close()
		if err != nil {
			tilogs.L().Errorf("ResetPasswordByCheat create csv fail ,err is %v ", err)
			c.String(403, err.Error())
			return
		}

		byteFile := bytes.NewBuffer([]byte{})
		csvWriter := csv.NewWriter(byteFile)
		_ = csvWriter.Write([]string{"\xEF\xBB\xBF"})
		// 遍历每行数据并生成随机密码，更新密码列
		for _, name := range nameList {
			newPassword := RandLowercase()
			err = models.ResetPwdByCheat(ctx, strings.ToLower(name), newPassword) // 生成长度为6的随机密码
			if err != nil {
				tilogs.L().Errorf("ResetPasswordByCheat ResetPwdByCheat err: %v Name:%v", err, name)
				continue
			}
			_ = csvWriter.Write([]string{"Name", name})
			_ = csvWriter.Write([]string{"Password", newPassword})
			_ = csvWriter.Write([]string{"", ""})
		}
		csvWriter.Flush()
		// 将csv文件写回磁盘
		_, err = pwdFile.Write(byteFile.Bytes())
		if err != nil {
			tilogs.L().Errorf("ResetPasswordByCheat SaveData err: %v", err)
			c.String(403, err.Error())
			return
		}

		c.JSON(200, map[string]interface{}{
			"result": "ok",
		})
	}
}

func RandLowercase() string {
	const BufSize = 6
	var buffer [BufSize]byte
	_, _ = rand.Read(buffer[:])
	// 啊, 又不是不能用
	for i, b := range buffer {
		buffer[i] = 'a' + (b % 26)
	}
	return string(buffer[:])
}

func GetCsvInfo(filePath string, fileName string) (error, [][]string) {
	file, err := os.Open(filepath.Join(filePath, fileName))
	if err != nil {
		tilogs.L().Errorf("GetCsvInfo 打开CSV文件：%v 异常：%v", fileName, err.Error())
		return err, nil
	}

	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	record, err := reader.ReadAll()
	if err != nil || len(record) == 0 {
		tilogs.L().Errorf("GetCsvInfo 读取CSV：%v 内容异常：%v", fileName, err.Error())
		return err, nil
	}

	return nil, record
}
