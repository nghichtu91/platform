package merge_db

import (
	"fmt"
)

const (
	rollbackPrefix = "merge_rollback" // 回滚数据的默认键值前缀
)

// GenRollbackKeyName 拼接用于回滚的redis key
// e.g.  gamex:43002:7:merge_rollback:guild
func GenRollbackKeyName(sid int32, mergedUID uint, typ string) string {
	return fmt.Sprintf("gamex:%d:%d:%s:%s:", sid, mergedUID, rollbackPrefix, typ)
}
