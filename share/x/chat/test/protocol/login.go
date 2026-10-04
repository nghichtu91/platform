package protocol

import (
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/test/user_info"
)

type LoginHandler struct {
	params map[string][]string
}

func (login *LoginHandler) Init() {
	login.params = make(map[string][]string, 16)
}

func (login *LoginHandler) FixedRequest(index string) []byte {
	var params = make([]string, 0, 16)
	if user_info.IsMultiple() {
		robotId, err := strconv.Atoi(index)
		if err != nil {
			fmt.Println("request fixed login error index strconv to int failed")
			return nil
		}
		robot := user_info.GetRobot(robotId)
		if robot == nil {
			fmt.Printf("request fixed login error, robot not exist id:%d", robotId)
			return nil
		}

		params = append(params, robot.GetUserId())
		params = append(params, robot.GetToken())
	} else {
		params = append(params, user_info.GetUserAcid())
		params = append(params, user_info.GetUserToken())
	}

	return login.ControlledRequest(params)
}

func (login *LoginHandler) ControlledRequest(params []string) []byte {
	if len(params) != 2 {
		fmt.Println("login param count error")
		return nil
	}

	loginMsg := pb.ChatLogin{
		PlayerId: proto.String(params[0]),
		Token:    proto.String(params[1]),
	}

	protoBytes := generateMsg(&loginMsg)
	buf := generateCommon(uint32(pb.ChatOperation_C2SLogin), len(protoBytes))
	buf = append(buf, protoBytes...)
	return buf
}

func (login *LoginHandler) ResponseString(msg []byte) string {
	p := &pb.ChatLoginResp{}
	proto.Unmarshal(msg, p)
	return fmt.Sprintf("login msg: %v\n", p)
}
