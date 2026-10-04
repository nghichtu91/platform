package allmetrics

import "fmt"

func PrefixGMServerMetrics(gid uint, serverId string) string {
	return fmt.Sprintf("%s.%d.%s", GMServerPrefix, gid, serverId)
}
