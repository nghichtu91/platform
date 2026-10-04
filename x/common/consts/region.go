package consts

// Gid枚举
const (
	LiveCN         = 51 // 51大区，国服线上			北京时间			东八区
	LiveCNStreamer = 52 // 52大区，主播服				北京时间			东八区
	LiveHMT        = 61 // 61大区，港澳台线上 			北京时间			东八区
	LiveAME        = 62 // 62大区，美洲 				北京时间-13h		西五区
	LiveEMEA       = 63 // 63大区，欧洲+中东+非洲 		北京时间-5h		东三区
	LiveJP         = 65 // 65大区，日本 				北京时间+1h		东九区
	LiveVN         = 66 // 66大区，越南 				北京时间-1h 		东七区
	LiveKR         = 67 // 67大区，韩国 				北京时间+1h		东九区
)

var (
	LiveRegions = map[int]string{
		LiveCN:   "CN",
		LiveHMT:  "HMT",
		LiveAME:  "AME",
		LiveEMEA: "EMEA",
		LiveJP:   "JP",
		LiveVN:   "VN",
		LiveKR:   "KR",
	}
)

// IsLiveRegion 是否是线上大区
func IsLiveRegion(gid int) bool {
	_, ok := LiveRegions[gid]
	return ok
}
