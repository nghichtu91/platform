package job

import (
	"fmt"
	"sync"

	"github.com/opentracing/opentracing-go"

	"github.com/nghichtu91/platform/share/planx/nats_cli"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	dis "github.com/nghichtu91/platform/share/x/chat/api/discovery"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/jobx/config"
	handler "github.com/nghichtu91/platform/share/x/chat/jobx/job/handlers"
)

// Job is push job.
type Job struct {
	cometServers map[string]*Comet
	rooms        map[string]*Room
	mutex        sync.RWMutex
}

func New() *Job {
	return &Job{
		cometServers: make(map[string]*Comet, 64),
		rooms:        make(map[string]*Room, 64),
	}
}

func (job *Job) InitConsume() {
	gid := fmt.Sprintf("%d", config.Cfg.Gid)
	nats_cli.GetHandlersMgr().RegQueueWithLock(nats_cli.ChatSubj(gid),
		&handler.PushMsgHandle{
			Cb: job,
		})
}

func (job *Job) OnServerAdd(serverId string, serverInfo *dis.CometServerInfo) {
	log.L().Infof("DiscoveryNewComet server id is %s, info is %v", serverId, serverInfo)

	comet, err := NewComet(&config.Cfg, serverId, serverInfo.PrivateAddr)
	if err != nil {
		log.L().Warnf("connect to comet with grpc failed! server id is %s, addr is %s", serverId, serverInfo.PrivateAddr)
	}

	job.mutex.Lock()
	defer job.mutex.Unlock()

	tempComets := make(map[string]*Comet, 16)
	for k, v := range job.cometServers {
		tempComets[k] = v
	}

	tempComets[serverId] = comet
	job.cometServers = tempComets
}

func (job *Job) OnServerDel(serverId string) {
	job.mutex.Lock()
	defer job.mutex.Unlock()

	var delComet *Comet
	tempComets := make(map[string]*Comet, 16)
	for k, v := range job.cometServers {
		if k == serverId {
			delComet = v
			continue
		}
		tempComets[k] = v
	}

	if delComet != nil {
		delComet.cancel()
	}
	job.cometServers = tempComets
	log.L().Infof("DiscoveryDelComet server id is %s", serverId)
}

func (job *Job) Push(fatherSpan opentracing.Span, p *pb.PushMsg) {
	log.L().Debugf("job receive logic push, key is:%s, server is:%s, room is: %s", p.GetKey(), p.GetServer(), p.GetRoom())

	switch p.GetType() {
	case pb.PushMsg_PUSH:
		job.PushKey(fatherSpan, p.GetServer(), p)
	case pb.PushMsg_ROOM:
		job.getRoom(p.GetRoom()).Push(fatherSpan, p.Msg.GetMessageId(), p.Msg.RawData)
	case pb.PushMsg_HISTORY:
		job.PushKey(fatherSpan, p.GetServer(), p)
	}
}
