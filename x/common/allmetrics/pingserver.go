package allmetrics

import "fmt"

func PrefixPingServerMetrics(gid uint, serverId string) string {
	battlexGid = gid
	dataprefix := fmt.Sprintf("%s.%d.%s", PingServerPrefix, gid, serverId)
	return dataprefix
}
