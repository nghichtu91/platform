package scene

import (
	"strconv"

	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/x/common/msg/scene/pb"
)

func graphiteName(prefix, code string) string {
	return prefix + code
}

func (s *scene_server) recordRequestMetrics(msg *pb.SceneMsg) string {
	name := graphiteName(s.sceMgr.metricsPrefix, strconv.Itoa(int(msg.MsgId))+"req")
	recordCountAndSize(name, len(msg.MsgData))
	return name
}

func (s *scene_server) recordLineRequestMetrics(code string) string {
	name := graphiteName(s.sceMgr.metricsPrefix, code)
	metrics.CountStatistics(name)
	return name
}

func (s *scene_server) recordSendMetrics(msg *pb.SceneMsg) {
	name := graphiteName(s.sceMgr.metricsPrefix, strconv.Itoa(int(msg.MsgId))+"send")
	recordCountAndSize(name, len(msg.MsgData))
}

func recordCountAndSize(name string, size int) {
	metrics.CountStatistics(name)
	metrics.SizeStatistics(name, size)
}

func recordTime(name, k4stat string, beforeTimeNano int64) {
	metrics.TimeStatistics(name, k4stat, beforeTimeNano)
}
