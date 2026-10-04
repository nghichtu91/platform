package job

import (
	"errors"
	"time"

	"github.com/opentracing/opentracing-go"

	log "github.com/nghichtu91/platform/share/planx/tilogs"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/jobx/config"
	"github.com/nghichtu91/platform/share/x/chat/pkg/bytes"
)

var (
	// ErrComet commet error.
	ErrComet = errors.New("comet rpc is not available")
	// ErrCometFull comet chan full.
	ErrCometFull = errors.New("comet proto chan full")
	// ErrRoomFull room chan full.
	ErrRoomFull    = errors.New("room proto chan full")
	roomReadyProto = new(pb.ClientPackageWithSpan)
	InitBodySize   = 1024
)

// Room room.
type Room struct {
	c     *config.ChatConfig
	job   *Job
	id    string
	proto chan *pb.ClientPackageWithSpan
}

// NewRoom new a room struct, store channel room info.
func NewRoom(job *Job, id string, c *config.ChatConfig) (r *Room) {
	r = &Room{
		c:     c,
		id:    id,
		job:   job,
		proto: make(chan *pb.ClientPackageWithSpan, c.Room.Batch*2),
	}
	go r.pushproc(c.Room.Batch, time.Millisecond*time.Duration(c.Room.Signal))
	return
}

// Push push msg to the room, if chan full discard it.
func (r *Room) Push(fatherSpan opentracing.Span, messageId uint32, msg []byte) (err error) {
	select {
	case r.proto <- &pb.ClientPackageWithSpan{
		Span: fatherSpan,
		ClientPackage: &pb.ClientPackage{
			MessageId: &messageId,
			RawData:   msg,
		},
	}:
	default:
		log.L().Errorf("Room.Push Msg Miss messageId:%d", messageId)
		err = ErrRoomFull
	}
	return
}

// pushproc merge proto and push msgs in batch.
func (r *Room) pushproc(batch int, sigTime time.Duration) {
	var (
		n         int
		last      time.Time
		p         *pb.ClientPackageWithSpan
		buf       = bytes.NewWriterSize(InitBodySize)
		firstSpan opentracing.Span
	)
	log.L().Infof("start room:%s goroutine", r.id)
	td := time.AfterFunc(sigTime, func() {
		select {
		case r.proto <- roomReadyProto:
		default:
			log.L().Errorf("Room.Push Proc roomReadyProto Miss")
		}
	})

	defer func() {
		// 关闭房间时,将chan中的协议处理完
		buf.Reset()
		var hasChange bool
		for {
			select {
			case p = <-r.proto:
				if p == nil {
					if hasChange {
						_ = r.job.PushRoom(firstSpan, r.id, buf.Buffer())
						if firstSpan != nil {
							firstSpan.Finish()
							firstSpan = nil
						}
					}
					log.L().Infof("room:%s goroutine exit", r.id)
					return
				}
				if p == roomReadyProto {
					continue
				}

				if p.Span != nil && firstSpan == nil {
					firstSpan = opentracing.GlobalTracer().StartSpan("GameChatMsg.job.room", opentracing.ChildOf(p.Span.Context()))
				}

				hasChange = true
				p.WriteTo(buf)
			default:
				if hasChange {
					_ = r.job.PushRoom(firstSpan, r.id, buf.Buffer())
					if firstSpan != nil {
						firstSpan.Finish()
						firstSpan = nil
					}
				}
				log.L().Infof("room:%s goroutine exit", r.id)
				return
			}
		}
	}()
	defer td.Stop()

	for {
		if p = <-r.proto; p == nil {
			break // exit
		} else if p != roomReadyProto {
			if p.Span != nil && firstSpan == nil {
				firstSpan = opentracing.GlobalTracer().StartSpan("GameChatMsg.job.room", opentracing.ChildOf(p.Span.Context()))
			}

			// merge buffer ignore error, always nil
			p.WriteTo(buf)
			if n++; n == 1 {
				last = time.Now()
				td.Reset(sigTime)
				continue
			} else if n < batch {
				if sigTime > time.Since(last) {
					continue
				}
			}
		} else {
			if n == 0 {
				break
			}
		}
		_ = r.job.PushRoom(firstSpan, r.id, buf.Buffer())
		if firstSpan != nil {
			firstSpan.Finish()
			firstSpan = nil
		}
		buf.Reset()
		n = 0
		if r.c.Room.Idle != 0 {
			td.Reset(time.Second * time.Duration(r.c.Room.Idle))
		} else {
			td.Reset(time.Minute)
		}
	}

	r.job.delRoom(r.id)
}

func (job *Job) delRoom(roomID string) {
	job.mutex.Lock()
	delete(job.rooms, roomID)
	job.mutex.Unlock()
}

func (job *Job) getRoom(roomID string) *Room {
	job.mutex.RLock()
	room, ok := job.rooms[roomID]
	job.mutex.RUnlock()
	if !ok {
		job.mutex.Lock()
		if room, ok = job.rooms[roomID]; !ok {
			room = NewRoom(job, roomID, &config.Cfg)
			job.rooms[roomID] = room
		}
		roomsLen := len(job.rooms)
		job.mutex.Unlock()
		log.L().Infof("new a room:%s active:%d", roomID, roomsLen)
	}
	return room
}
