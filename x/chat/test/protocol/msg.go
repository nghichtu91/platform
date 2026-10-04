package protocol

import (
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/planx/nats_cli"
	"github.com/nghichtu91/platform/share/x/chat/api/protogen"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/test/user_info"
)

type ChatMsgHandler struct {
	params map[string][]string
}

func (chat *ChatMsgHandler) Init() {
	chat.params = make(map[string][]string, 20)
	chat.params["1"] = []string{"WorldChannel_120004", "1", "hello world"}
	chat.params["2"] = []string{"WorldChannel_120004", "1", "hi"}
	chat.params["3"] = []string{"WorldChannel_120004", "1", "how old are you"}
	chat.params["4"] = []string{"WorldChannel_120004", "1", "where are you from"}
	chat.params["5"] = []string{"WorldChannel_120004", "1", "fuck you"}
	chat.params["6"] = []string{"WorldChannel_120004", "1", "welcome"}
	chat.params["7"] = []string{"WorldChannel_120004", "1", "i love you"}
	chat.params["8"] = []string{"WorldChannel_120004", "1", "nothing"}
	chat.params["9"] = []string{"WorldChannel_120004", "1", "todo"}
	chat.params["10"] = []string{"WorldChannel_120004", "1", "end"}

	chat.params["11"] = []string{"WorldChannel_1", "1", "hello world"}
	chat.params["12"] = []string{"WorldChannel_1", "1", "hi"}
	chat.params["13"] = []string{"WorldChannel_1", "1", "how old are you"}
	chat.params["14"] = []string{"WorldChannel_1", "1", "where are you from"}
	chat.params["15"] = []string{"WorldChannel_1", "1", "fuck you"}
	chat.params["16"] = []string{"WorldChannel_1", "1", "welcome"}
	chat.params["17"] = []string{"WorldChannel_1", "1", "i love you"}
	chat.params["18"] = []string{"WorldChannel_1", "1", "nothing"}
	chat.params["19"] = []string{"WorldChannel_1", "1", "todo"}
	chat.params["20"] = []string{"WorldChannel_1", "1", "end"}
}

func (chat *ChatMsgHandler) FixedRequest(index string) []byte {
	params, ok := chat.params[index]
	if !ok {
		fmt.Println("request fixed chat msg error")
		return nil
	}

	return chat.ControlledRequest(params)
}

func (chat *ChatMsgHandler) ControlledRequest(params []string) []byte {
	if len(params) != 3 {
		fmt.Println("chat msg param count error")
		return nil
	}
	acid := params[0]
	typ, ok := strconv.Atoi(params[1])
	if ok != nil {
		fmt.Println("chat msg param atoi failed")
		return nil
	}

	content := params[2]
	loginMsg := pb.GamexChatMsg{
		FromId:   proto.String(""),
		TargetId: &acid,
		Type:     pb.ChatType(typ).Enum(),
		Content:  &content,
		ExtraParam: &protogen.ExtraParam{
			PlayerShowInfo: &protogen.CommonShowRoleHeadInfo{
				Level: proto.Uint32(1),
			},
		},
	}

	nats_cli.Request(nats_cli.ChatGamexSubj(fmt.Sprint(user_info.GetGid())), &loginMsg)
	return nil
}

func (chat *ChatMsgHandler) ResponseString(msg []byte) string {
	p := &pb.ChatMsgResp{}
	proto.Unmarshal(msg, p)
	return fmt.Sprintf("chat msg: %v\n", p)
}
