package protocol

import (
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

type HistoryHandler struct {
	params map[string][]string
}

func (his *HistoryHandler) Init() {
	his.params = make(map[string][]string, 1)
	his.params["1"] = []string{"playerA", "0", "0", "10"}
}

func (his *HistoryHandler) FixedRequest(index string) []byte {
	params, ok := his.params[index]
	if !ok {
		fmt.Println("request fixed history msg error")
		return nil
	}

	return his.ControlledRequest(params)
}

func (his *HistoryHandler) ControlledRequest(params []string) []byte {
	if len(params) != 4 {
		fmt.Println("get history msg param count error")
		return nil
	}

	targetId := params[0]
	t, _ := strconv.Atoi(params[1])
	chatType := pb.ChatType(t)
	begin, _ := strconv.Atoi(params[2])
	end, _ := strconv.Atoi(params[3])
	historyMsg := pb.HistoryMsg{
		TargetId:   &targetId,
		Type:       &chatType,
		IndexBegin: proto.Uint32(uint32(begin)),
		IndexEnd:   proto.Uint32(uint32(end)),
	}

	protoBytes := generateMsg(&historyMsg)
	buf := generateCommon(uint32(pb.ChatOperation_C2SHistroyMsg), len(protoBytes))
	buf = append(buf, protoBytes...)
	return buf
}

func (his *HistoryHandler) ResponseString(msg []byte) string {
	p := &pb.HistoryMsgResp{}
	proto.Unmarshal(msg, p)
	return fmt.Sprintf("get history msg: %v\n", p)
}
