package allmetrics

import "fmt"

func PrefixGMChatMetrics(gid uint) string {
	return fmt.Sprintf("%s.%d.gmchat", GMChatPrefix, gid)
}
