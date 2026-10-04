package allmetrics

import "fmt"

func PrefixModulexMetrics(gid, sid uint) string {
	return fmt.Sprintf("%s.%d.%d", ModulexPrefix, gid, sid)
}

func InitModulexMetrics() {
}
