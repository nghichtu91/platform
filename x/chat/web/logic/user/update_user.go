package user

import (
	"time"

	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"

	"github.com/gin-gonic/gin"

	_ "database/sql"

	"crypto/md5"

	"fmt"

	"github.com/nghichtu91/platform/share/x/chat/web/db"
	"github.com/nghichtu91/platform/share/x/chat/web/model/errs"
)

/**
更新或者创建用户
*/

type UpdateUserRequest struct {
	UpdateType int    `form:"update_type"` // 0 添加， 1 更新内容， 2 禁用， 3， 改密码
	Email      string `form:"email"`
	RoleId     int    `form:"roleId"`
	Status     int    `form:"status"` // 0 可用 1 禁用
	Password   string `form:"pwd"`
}

type UpdateUserResponse struct {
}

func (req *UpdateUserRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	switch req.UpdateType {
	case 0:
		if err := createUser(req); err != nil {
			return nil, err, errs.SERVER
		}
	case 1:
		if err := updateUser(req); err != nil {
			return nil, err, errs.SERVER
		}
	case 2:
		if err := updateStatus(req); err != nil {
			return nil, err, errs.SERVER
		}
	case 3:
		if err := updatePwd(req); err != nil {
			return nil, err, errs.SERVER
		}
	}

	return &UpdateUserResponse{}, nil, ""
}

func createUser(req *UpdateUserRequest) error {
	pwdWithSalt := []byte(req.Email + super_user.DefaultPwd)
	pwd := fmt.Sprintf("%x", md5.Sum(pwdWithSalt))
	_, err := db.GetDB().Exec("insert into users(email, pwd, user_group, create_time)values(?, ?, ?, ?)",
		req.Email, pwd, req.RoleId, time.Now().Unix())
	if err != nil {
		return err
	}
	return nil
}

func updateUser(req *UpdateUserRequest) error {
	if req.Email == super_user.AdminRoot {
		return fmt.Errorf("不能修改超级管理员信息")
	}
	_, err := db.GetDB().Exec("update users set user_group = ? where email = ?",
		req.RoleId, req.Email)
	if err != nil {
		return err
	}
	return nil
}

func updateStatus(req *UpdateUserRequest) error {
	if req.Email == super_user.AdminRoot {
		return fmt.Errorf("不能修改超级管理员信息")
	}
	_, err := db.GetDB().Exec("update users set status = ? where email = ?",
		req.Status, req.Email)
	if err != nil {
		return err
	}
	return nil
}

func updatePwd(req *UpdateUserRequest) error {
	pwdWithSalt := []byte(req.Email + req.Password)
	pwd := fmt.Sprintf("%x", md5.Sum(pwdWithSalt))
	_, err := db.GetDB().Exec("update users set pwd = ? where email = ?",
		pwd, req.Email)
	if err != nil {
		return err
	}
	return nil
}
