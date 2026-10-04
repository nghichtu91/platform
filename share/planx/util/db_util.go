package util

import (
	"database/sql"
	"strings"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func SafeCloseRows(rows *sql.Rows) {
	if rows == nil {
		return
	}
	err := rows.Close()
	if err == nil {
		return
	}

	tilogs.L().Errorf("close sql.Rows error %v", err)
}

//DBStringSplit 空串不参与分割, 直接返回空即可
func DBStringSplit(str, sep string) []string {
	if str == "" {
		return []string{}
	}
	return strings.Split(str, sep)
}
