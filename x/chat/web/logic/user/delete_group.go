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

type DeletePowerGroupRequest struct {
	Id int `form:"id"`
}

type DeletePowerGroupResponse struct {
}

func (req *DeletePowerGroupRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	err := deletePowerGroup(req.Id)
	if err != nil {
		return nil, err, errs.DbError
	}

	return &DeletePowerGroupResponse{}, nil, ""
}

func deletePowerGroup(id int) error {
	_, err := db.GetDB().Exec("delete from user_group where id = ?", id)
	if err != nil {
		return err
	}
	return nil
}
