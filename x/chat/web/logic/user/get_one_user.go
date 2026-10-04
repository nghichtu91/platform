package user

import (
	"time"

	"github.com/nghichtu91/platform/share/x/chat/web/model/errs"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"

	"github.com/gin-gonic/gin"

	_ "database/sql"

	"github.com/nghichtu91/platform/share/x/chat/web/db"
)

type GetOneUserRequest struct {
	Email string `form:"email"`
}

type GetOneUserResponse struct {
	Data *UserInfo `json:"data"`
}

func (req *GetOneUserRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	if userInfo, err := getOneUserInfo(req.Email); err != nil {
		return nil, err, errs.SERVER
	} else {
		return &GetOneUserResponse{
			Data: userInfo,
		}, nil, ""
	}
}

func getOneUserInfo(reqEmail string) (*UserInfo, error) {
	if reqEmail == super_user.AdminRoot {
		return &UserInfo{
			Email:      super_user.AdminRoot,
			RoleId:     super_user.AccountGroupAdmin,
			CreateTime: time.Unix(int64(0), 0).Format("2006-01-02 15:04:05"),
			Status:     0,
			RoleName:   super_user.AdminRoleName,
		}, nil
	}

	rows := db.GetDB().QueryRow(`select users.email as email,
		users.user_group as groupId,
		users.create_time as createTime,
		users.status as status,
		user_group.name as roleName
		from users, user_group
		where users.user_group = user_group.id and users.email =? `, reqEmail)
	var (
		email                       string
		groupId, createTime, status int
		roleName                    string
	)
	if err := rows.Scan(&email, &groupId, &createTime, &status, &roleName); err != nil {
		return nil, err
	} else {
		return &UserInfo{
			Email:      email,
			RoleId:     groupId,
			CreateTime: time.Unix(int64(createTime), 0).Format("2006-01-02 15:04:05"),
			Status:     status,
			RoleName:   roleName,
		}, nil
	}
}
