package user

import (
	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/x/chat/web/db"
	"github.com/nghichtu91/platform/share/x/chat/web/model/errs"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"
)

/**
创建权限管理分组
*/

type DeleteUserRequest struct {
	Email string `form:"email"`
}

type DeleteUserResponse struct {
}

func (req *DeleteUserRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	err := deleteUser(req.Email)
	if err != nil {
		return nil, err, errs.SERVER
	}
	return &DeleteUserResponse{}, nil, ""
}

func deleteUser(email string) error {
	_, err := db.GetDB().Exec("delete from users where email = ?", email)
	if err != nil {
		return err
	}
	return nil
}
