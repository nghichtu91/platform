package allmetrics

import "fmt"

func PrefixCrossMetrics(gid uint, serverId string) string {
	battlexGid = gid
	dataprefix := fmt.Sprintf("%s.%d.%s", CrossxPrefix, gid, serverId)
	return dataprefix
}

func InitCrossMetrics() {

}
