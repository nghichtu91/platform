package scene_pool

import (
	"sync"

	"github.com/nghichtu91/platform/share/x/common/msg/scene/pb"
)

var (
	msgPool = sync.Pool{New: func() interface{} {
		return new(pb.SceneMsg)
	}}
)

func GetMsg() *pb.SceneMsg {
	return msgPool.Get().(*pb.SceneMsg)
}

func PutMsg(msg *pb.SceneMsg) {
	msg.Reset()
	msgPool.Put(msg)
}
