package comet

import (
	"errors"
	"sync"

	log "github.com/nghichtu91/platform/share/planx/tilogs"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
)

// Room is a room and store channel room info.
type Room struct {
	ID        string
	rLock     sync.RWMutex
	members   map[string]*Session
	drop      bool
	Online    int32 // dirty read is ok
	AllOnline int32
}

// NewRoom new a room struct, store channel room info.
func NewRoom(id string) (r *Room) {
	r = new(Room)
	r.ID = id
	r.drop = false
	r.Online = 0
	r.members = make(map[string]*Session, 64)

	log.L().Infof("NewRoom id:%s", id)
	return
}

// Put put channel into the room.
func (r *Room) Put(ch *Session) (err error) {
	r.rLock.Lock()
	if !r.drop {
		r.members[ch.Key] = ch

		log.L().Infof("Room:%s put session:%s", r.ID, ch.Key)
		r.Online++
	} else {
		err = errors.New(pb.ErrorCode_RoomDrop.String())
	}
	r.rLock.Unlock()
	return
}

// Del delete channel from the room.
func (r *Room) Del(ch *Session) bool {
	r.rLock.Lock()
	log.L().Infof("Room:%s del session:%s", r.ID, ch.Key)
	delete(r.members, ch.Key)
	r.Online--
	r.drop = (r.Online == 0)
	r.rLock.Unlock()
	return r.drop
}

// Push push msg to the room, if chan full discard it.
func (r *Room) Push(p *pb.ClientPackageWithSpan) {
	r.rLock.RLock()
	for _, ch := range r.members {
		ch.Push(p)
	}
	r.rLock.RUnlock()
}

// Close close the room.
func (r *Room) Close() {
	log.L().Infof("comet room:%s close", r.ID)
	r.rLock.RLock()
	for _, ch := range r.members {
		ch.Close()
	}
	r.rLock.RUnlock()
}

// OnlineNum the room all online.
func (r *Room) OnlineNum() int32 {
	if r.AllOnline > 0 {
		return r.AllOnline
	}
	return r.Online
}
