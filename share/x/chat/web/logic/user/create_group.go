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

type CreatePowerGroupRequest struct {
	Name  string `form:"name"`
	Power string `form:"power"`
	Id    int    `form:"id"`
}

type CreatePermissionGroupResponse struct {
}

func (req *CreatePowerGroupRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	if req.Id == 0 {
		if err := createPowerGroup(req); err != nil {
			return nil, err, errs.SERVER
		}
	} else {
		if err := updatePowerGroup(req); err != nil {
			return nil, err, errs.SERVER
		}
	}
	return &CreatePermissionGroupResponse{}, nil, ""
}

func createPowerGroup(req *CreatePowerGroupRequest) error {
	_, err := db.GetDB().Exec("insert into user_group(name, permission) values(?,?)", req.Name, req.Power)
	if err != nil {
		return err
	}
	return nil
}

func updatePowerGroup(req *CreatePowerGroupRequest) error {
	_, err := db.GetDB().Exec("update user_group set name = ?, permission=? where id=?",
		req.Name, req.Power, req.Id)
	if err != nil {
		return err
	}
	return nil
}
