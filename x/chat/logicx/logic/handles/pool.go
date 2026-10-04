package handles

import (
	"sync"

	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

var (
	chatMsgPool = sync.Pool{
		New: func() interface{} {
			return new(pb.GamexChatMsg)
		}}
	validMsgPool = sync.Pool{
		New: func() interface{} {
			return new(pb.TextValidateReq)
		},
	}
)
