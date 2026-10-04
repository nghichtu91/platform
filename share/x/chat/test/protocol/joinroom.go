package protocol

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

type JoinRoomHandler struct {
	params map[string][]string
}

func (jroom *JoinRoomHandler) Init() {
	jroom.params = make(map[string][]string, 4)
	jroom.params["1"] = []string{"room1"}
	jroom.params["2"] = []string{"room1", "room2"}
	jroom.params["3"] = []string{"WorldChannel_120004"}
	jroom.params["4"] = []string{"WorldChannel_1"}
}

func (jroom *JoinRoomHandler) FixedRequest(index string) []byte {
	params, ok := jroom.params[index]
	if !ok {
		fmt.Println("request fixed join room msg error")
		return nil
	}

	return jroom.ControlledRequest(params)
}

func (jroom *JoinRoomHandler) ControlledRequest(params []string) []byte {
	if len(params) <= 0 {
		fmt.Println("join room msg param count error")
		return nil
	}

	loginMsg := pb.JoinRoom{
		RoomId: params,
	}

	protoBytes := generateMsg(&loginMsg)
	buf := generateCommon(uint32(pb.ChatOperation_C2SJoinRoom), len(protoBytes))
	buf = append(buf, protoBytes...)
	return buf
}

func (jroom *JoinRoomHandler) ResponseString(msg []byte) string {
	p := &pb.JoinRoomResp{}
	proto.Unmarshal(msg, p)
	return fmt.Sprintf("join room msg: %v\n", p)
}
