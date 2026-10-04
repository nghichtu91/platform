package user

import (
	"database/sql"

	"github.com/nghichtu91/platform/share/x/chat/web/model/command"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"

	"github.com/nghichtu91/platform/share/x/chat/web/model/errs"

	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/x/chat/web/db"
)

type QueryRecordRequest struct {
}

type QueryRecordResponse struct {
	List []RecordInfo `json:"list"`
}

type RecordInfo struct {
	Id              int    `json:"id"`
	User            string `json:"user"`
	Ip              string `json:"ip"`
	Time            int    `json:"time"`
	OperationType   string `json:"operation_type"`
	OperationName   string `json:"operation_name"`
	OperationDetail string `json:"operation_detail"`
}

func (req *QueryRecordRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	rows, err := db.GetDB().Query("select * from operation_log order by id asc")
	if err != nil {
		return nil, err, errs.SERVER
	}
	defer rows.Close()
	list := make([]RecordInfo, 0)
	for rows.Next() {
		var (
			id, time                           int
			ip, operationType, operationDetail string
			user                               sql.NullString
		)
		err := rows.Scan(&id, &user, &ip, &time, &operationType, &operationDetail)
		if err != nil {
			return nil, err, errs.DbError
		}
		list = append(list, RecordInfo{
			Id:              id,
			User:            user.String,
			Ip:              ip,
			Time:            time,
			OperationType:   operationType,
			OperationName:   command.CommandName[operationType],
			OperationDetail: operationDetail,
		})
	}
	return &QueryRecordResponse{
		List: list,
	}, nil, ""
}
