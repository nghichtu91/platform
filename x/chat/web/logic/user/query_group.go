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

type QueryPowerGroupRequest struct {
	Page int `form:"page"`
}

type QueryPowerGroupResponse struct {
	List []*GroupInfo `json:"list"`
}

type GroupInfo struct {
	Id         int           `json:"id"`
	Name       string        `json:"name"`
	Permission map[int][]int `json:"power"`
}

func (req *QueryPowerGroupRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	_, err := countPowerGroup()
	if err != nil {
		return nil, err, errs.SERVER
	}

	groupInfos, err := queryPowerGroup(req.Page)
	if err != nil {
		return nil, err, errs.SERVER
	}

	return &QueryPowerGroupResponse{
		List: groupInfos,
	}, nil, ""

}

const (
	Group_Limit = 10
)

func queryPowerGroup(page int) ([]*GroupInfo, error) {
	//pageStart := page * Group_Limit
	//pageEnd := pageStart + 10
	rows, err := db.GetDB().Query("select id, name, permission from user_group " +
		"order by id desc limit 0, 10")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	infos := make([]*GroupInfo, 0)
	for rows.Next() {
		var id int
		var name string
		var permission []byte
		if err := rows.Scan(&id, &name, &permission); err != nil {
			return nil, err
		}
		infos = append(infos, &GroupInfo{
			Id:         id,
			Name:       name,
			Permission: super_user.ParsePermission(permission),
		})
	}
	return infos, nil
}

func countPowerGroup() (int, error) {
	row := db.GetDB().QueryRow("select count(1) from user_group")
	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	} else {
		return count, nil
	}
}
