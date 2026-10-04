package noticeimp

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nghichtu91/platform/share/x/common/notice"
)

type MysqlDriver struct {
	db *sql.DB
}

func (m *MysqlDriver) Get(gid uint, version string) []*NoticeInfo {
	if version != "" {
		row := m.db.QueryRow("select content from system_notice_public where version = ?", version)
		var content []byte
		err := row.Scan(&content)
		if err != nil {
			tilogs.L().Errorf("QueryRow system_notice_public %d %s, %v ", gid, version, err)
			return nil
		}
		noticeInfo := parseNotice(fmt.Sprintf("%d", gid), version, content)
		return []*NoticeInfo{noticeInfo}
	} else {
		memoryList := make([]*NoticeInfo, 0)
		rows, err := m.db.Query("select version, content from system_notice_public")
		if err != nil {
			tilogs.L().Errorf("query system_notice_public %d, %v ", gid, err)
			return nil
		}
		defer util.SafeCloseRows(rows)

		for rows.Next() {
			var version string
			var content []byte
			err := rows.Scan(&version, &content)
			if err != nil {
				tilogs.L().Errorf("scan rows err %v", err)
				break
			}
			memory := parseNotice(fmt.Sprintf("%d", gid), version, content)
			memoryList = append(memoryList, memory)
		}
		return memoryList
	}
}

func parseNotice(gid, version string, content []byte) *NoticeInfo {
	publicNotice := &notice.PublicRelease{}
	err := json.Unmarshal(content, publicNotice)
	if err != nil {
		tilogs.L().Errorf("json Unmarshal notice error %v, %s", err, string(content))
		return nil
	}
	noticeMemory := &NoticeInfo{
		Gid:        gid,
		Version:    version,
		NoticeInfo: *publicNotice,
	}
	tilogs.L().Infof("[Notice] parse notice ok %v", noticeMemory)
	return noticeMemory
}
