package protocol

import (
	"fmt"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

type HeartbeatHandler struct {
}

func (heartbeat *HeartbeatHandler) Init() {
}

func (heartbeat *HeartbeatHandler) FixedRequest(index string) []byte {
	buf := generateCommon(uint32(pb.ChatOperation_C2SHeartbeat), 0)
	return buf
}

func (heartbeat *HeartbeatHandler) ControlledRequest(params []string) []byte {
	buf := generateCommon(uint32(pb.ChatOperation_C2SHeartbeat), 0)
	return buf
}

func (heartbeat *HeartbeatHandler) ResponseString(msg []byte) string {
	return fmt.Sprintf("chat pong")
}
