package login

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"database/sql"

	"fmt"

	"crypto/md5"

	"time"

	"github.com/nghichtu91/platform/share/x/chat/web/db"

	"github.com/nghichtu91/platform/share/x/chat/web/model/errs"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"
)

type LoginRequest struct {
	Name string `form:"username"`
	Pwd  string `form:"password"`
}

type LoginResponse struct {
	LoginToken string        `json:"login_token"`
	Permission map[int][]int `json:"role_power"`
}

func (req *LoginRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	data := &LoginResponse{}
	checkPwd := buildPwd(req.Name, req.Pwd)
	row := db.GetDB().QueryRow("select user_group, permission, status from users where email = ? and pwd = ?", req.Name, checkPwd)
	var permissionBytes []byte
	var user_group, status int
	err := row.Scan(&user_group, &permissionBytes, &status)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errs.LogicError, errs.PwdError
		} else {
			return nil, err, errs.PwdError
		}
	} else if status == 1 {
		return nil, errs.LogicError, errs.AccountForbidden
	} else {
		data.LoginToken = buildLoginToken(req.Name, req.Pwd, loginKey)
		data.Permission = mergePermission(user_group)
		tilogs.L().Debugf("LoginToken %v", data.LoginToken)
		session := sessions.Default(c)
		session.Set(data.LoginToken, req.Name)
		session.Set(req.Name, data.LoginToken)
		session.Save()
		tilogs.L().Debugf("after login %s %s", session.Get(data.LoginToken), session.Get(req.Name))
	}
	return data, nil, ""
}

func buildPwd(userName string, pwd string) string {
	pwdWithSalt := []byte(userName + pwd)
	return fmt.Sprintf("%x", md5.Sum(pwdWithSalt))
}

const loginKey = "D4cSYuwNjbysYVVj"

func buildLoginToken(userName string, pwd string, loginKey string) string {
	info := md5.Sum([]byte(fmt.Sprintf("%s%s%s%d", userName, pwd, loginKey, time.Now().Unix())))
	return fmt.Sprintf("%x", info)
}

func mergePermission(group int) map[int][]int {
	if group == super_user.AccountGroupAdmin {
		return super_user.AdminRootPermission
	}
	row := db.GetDB().QueryRow("SELECT permission from user_group where id = ?", group)
	var permission []byte
	err := row.Scan(&permission)
	if err != nil {
		return nil
	}
	groupPermission := super_user.ParsePermission(permission)
	return groupPermission
}
