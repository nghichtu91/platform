package user

import (
	"time"

	"github.com/nghichtu91/platform/share/x/chat/web/model/errs"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"

	"github.com/nghichtu91/platform/share/x/chat/web/db"

	_ "database/sql"

	"github.com/gin-gonic/gin"
)

/**
查询用户列表
*/

type QueryUserRequest struct {
}

type QueryUserResponse struct {
	Data []*UserInfo `json:"data"`
}

type UserInfo struct {
	Email      string `json:"email"`
	CreateTime string `json:"createTime"` // 2003-03-18 11:38:31
	RoleId     int    `json:"roleId"`
	RoleName   string `json:"roleName"`
	Status     int    `json:"status"` // 0= ,可用， 1 禁用
}

func (req *QueryUserRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	if data, err := getInfoFromDB(); err != nil {
		return nil, err, errs.SERVER
	} else {
		return data, nil, ""
	}
}

func getInfoFromDB() (*QueryUserResponse, error) {
	users, err := getUserInfo()
	if err != nil {
		return nil, err
	}
	return &QueryUserResponse{
		Data: users,
	}, nil
}

func getUserInfo() ([]*UserInfo, error) {
	rows, err := db.GetDB().Query(`select users.email as email,
		users.user_group as groupId,
		users.create_time as createTime,
		users.status as status,
		user_group.name as roleName
		from users, user_group
		where users.user_group = user_group.id`)
	if err != nil {
		return nil, err
	}
	users := make([]*UserInfo, 0)
	for rows.Next() {
		var (
			email                       string
			groupId, createTime, status int
			roleName                    string
		)
		if err = rows.Scan(&email, &groupId, &createTime, &status, &roleName); err != nil {
			return nil, err
		}
		users = append(users, &UserInfo{
			Email:      email,
			RoleId:     groupId,
			CreateTime: time.Unix(int64(createTime), 0).Format("2006-01-02 15:04:05"),
			Status:     status,
			RoleName:   roleName,
		})
	}

	row := db.GetDB().QueryRow(`select email, create_time, status from users where user_group=?`, super_user.AccountGroupAdmin)

	var (
		email              string
		createTime, status int
	)
	if err = row.Scan(&email, &createTime, &status); err != nil {
		return nil, err
	}

	users = append(users, &UserInfo{
		Email:      email,
		RoleId:     super_user.AccountGroupAdmin,
		CreateTime: time.Unix(int64(createTime), 0).Format("2006-01-02 15:04:05"),
		Status:     status,
		RoleName:   "super admin",
	})

	return users, nil
}
