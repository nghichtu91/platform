package job

import (
	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

func (job *Job) PushKey(fatherSpan opentracing.Span, serverId string, p *pb.PushMsg) (err error) {
	var (
		comet *Comet
		ok    bool
	)

	p.GetMsg().TryCompress()

	job.mutex.RLock()
	if comet, ok = job.cometServers[serverId]; !ok {
		job.mutex.RUnlock()
		log.L().Warnf("job push key server id:%s not exist!", serverId)
		return
	}
	job.mutex.RUnlock()

	comet.Push(&pb.PushMsgReqWithSpan{
		Span: fatherSpan,
		PushMsgReq: &pb.PushMsgReq{
			Key:   p.Key,
			Proto: p.Msg,
		},
	})
	log.L().Debugf("push msg to comet, comet id is %s", serverId)
	return
}

func (job *Job) PushRoom(span opentracing.Span, roomId string, buf []byte) (err error) {
	req := &pb.BroadcastRoomReq{
		RoomID: &roomId,
		Proto: &pb.ClientPackage{
			MessageId: proto.Uint32(uint32(pb.ChatOperation_OpRawData)),
			RawData:   make([]byte, len(buf)),
		},
	}

	// race问题
	copy(req.Proto.RawData, buf)

	req.GetProto().TryCompress()

	job.mutex.RLock()
	comets := job.cometServers
	for serverID, c := range comets {
		if err = c.BroadcastRoom(&pb.BroadcastRoomReqWithSpan{
			Span:             span,
			BroadcastRoomReq: req,
		}); err != nil {
			log.L().Errorf("c.BroadcastRoom(%+v) roomID:%s serverID:%s error(%v)", req, roomId, serverID, err)
		}
	}
	log.L().Debugf("broadcastRoom comets:%d, roomId:%s", len(comets), roomId)
	job.mutex.RUnlock()

	return
}
