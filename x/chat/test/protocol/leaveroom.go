package protocol

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

type LeaveRoomHandler struct {
	params map[string][]string
}

func (lroom *LeaveRoomHandler) Init() {
	lroom.params = make(map[string][]string, 1)
	lroom.params["1"] = []string{"room1"}
}

func (lroom *LeaveRoomHandler) FixedRequest(index string) []byte {
	params, ok := lroom.params[index]
	if !ok {
		fmt.Println("request fixed chat msg error")
		return nil
	}

	return lroom.ControlledRequest(params)
}

func (lroom *LeaveRoomHandler) ControlledRequest(params []string) []byte {
	if len(params) <= 0 {
		fmt.Println("leave room msg param count error")
		return nil
	}

	loginMsg := pb.JoinRoom{
		RoomId: params,
	}

	protoBytes := generateMsg(&loginMsg)
	buf := generateCommon(uint32(pb.ChatOperation_C2SLeaveRoom), len(protoBytes))
	buf = append(buf, protoBytes...)
	return buf
}

func (lroom *LeaveRoomHandler) ResponseString(msg []byte) string {
	p := &pb.JoinRoomResp{}
	proto.Unmarshal(msg, p)
	return fmt.Sprintf("leave room msg: %v\n", p)
}
