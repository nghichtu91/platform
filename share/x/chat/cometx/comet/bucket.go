package comet

import (
	"sync"
	"sync/atomic"

	log "github.com/nghichtu91/platform/share/planx/tilogs"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/cometx/config"
)

// Bucket is a channel holder.
type Bucket struct {
	c *config.ChatConfig
	// protect chs and rooms
	cLock sync.RWMutex
	// 这个bucket上的所有session
	chs map[string]*Session
	// 这个bucket上的所有room
	rooms map[string]*Room // bucket room channels
	// 为了优化房间消息 为房间消息分配多个channel
	routines []chan *pb.BroadcastRoomReqWithSpan
	// room routine's count
	routinesNum uint64
}

// NewBucket new a bucket struct. store the key with im channel.
func NewBucket(c *config.ChatConfig) (b *Bucket) {
	b = new(Bucket)
	b.chs = make(map[string]*Session, c.Bucket.Channel)
	b.c = c
	b.rooms = make(map[string]*Room, c.Bucket.Room)
	b.routines = make([]chan *pb.BroadcastRoomReqWithSpan, c.Bucket.RoutineAmount)
	for i := 0; i < c.Bucket.RoutineAmount; i++ {
		c := make(chan *pb.BroadcastRoomReqWithSpan, c.Bucket.RoutineSize)
		b.routines[i] = c
		go b.roomproc(c)
	}
	return
}

// ChannelCount channel count in the bucket
func (b *Bucket) ChannelCount() int {
	return len(b.chs)
}

// RoomCount room count in the bucket
func (b *Bucket) RoomCount() int {
	return len(b.rooms)
}

// RoomsCount get all room id where online number > 0.
func (b *Bucket) RoomsCount() (res map[string]int32) {
	var (
		roomID string
		room   *Room
	)
	b.cLock.RLock()
	res = make(map[string]int32, len(b.rooms))
	for roomID, room = range b.rooms {
		if room.Online > 0 {
			res[roomID] = room.Online
		}
	}
	b.cLock.RUnlock()
	return
}

// Join room
func (b *Bucket) JoinRoom(nrid string, ch *Session) (err error) {
	var (
		nroom *Room
		ok    bool
	)

	b.cLock.Lock()
	if nroom, ok = b.rooms[nrid]; !ok {
		nroom = NewRoom(nrid)
		b.rooms[nrid] = nroom
	}
	b.cLock.Unlock()

	if err = nroom.Put(ch); err != nil {
		return
	}
	ch.Rooms[nrid] = nroom
	return
}

// Leave room
func (b *Bucket) LeaveRoom(drid string, ch *Session) (err error, delroom bool) {
	var (
		oroom *Room
		ok    bool
	)

	b.cLock.RLock()
	oroom, ok = b.rooms[drid]
	b.cLock.RUnlock()
	// Warning 这里没有加锁 只是在Del和DelRoom中加锁
	// 所以在Del之后 DelRoom之前有可能有其他函数拿到锁 对oroom进行操作
	// 如再离开过程中 有其他人正好在加入此房间 就会拿到一个即将删除的房间导致进入房间失败
	if ok && oroom != nil && oroom.Del(ch) {
		b.DelRoom(oroom)
		delroom = true
	}

	delete(ch.Rooms, drid)
	delete(ch.Cds, drid)
	return
}

// Put put a channel according with sub key.
func (b *Bucket) Put(ch *Session) (err error) {
	var (
		room *Room
	)
	b.cLock.Lock()
	// close old channel
	if dch := b.chs[ch.Key]; dch != nil {
		dch.Close()
	}
	b.chs[ch.Key] = ch
	b.cLock.Unlock()
	if room != nil {
		err = room.Put(ch)
	}
	return
}

// Del delete the channel by sub key.
func (b *Bucket) Del(dch *Session) {
	var (
		ok bool
		ch *Session
	)
	log.L().Infof("bucket del session:%s", dch.Key)
	b.cLock.Lock()
	if ch, ok = b.chs[dch.Key]; ok {
		if ch == dch {
			delete(b.chs, ch.Key)
		}
	}
	b.cLock.Unlock()
	for _, room := range dch.Rooms {
		if room != nil && ch != nil && room.Del(ch) {
			// if empty room, must delete from bucket
			b.DelRoom(room)
		}
	}
	dch.Cds = make(map[string]int64, 64)
}

// Channel get a channel by sub key.
func (b *Bucket) Channel(key string) (ch *Session) {
	b.cLock.RLock()
	ch = b.chs[key]
	b.cLock.RUnlock()
	return
}

// Broadcast push msgs to all channels in the bucket.
func (b *Bucket) Broadcast(p *pb.ClientPackageWithSpan, op int32) {
	var ch *Session
	b.cLock.RLock()
	for _, ch = range b.chs {
		/*
			if !ch.NeedPush(op) {
				continue
			}
		*/
		_ = ch.Push(p)
	}
	b.cLock.RUnlock()
}

// Room get a room by roomid.
func (b *Bucket) Room(rid string) (room *Room) {
	b.cLock.RLock()
	room = b.rooms[rid]
	b.cLock.RUnlock()
	return
}

// DelRoom delete a room by roomid.
func (b *Bucket) DelRoom(room *Room) {
	b.cLock.Lock()
	delete(b.rooms, room.ID)
	b.cLock.Unlock()
	room.Close()
}

// BroadcastRoom broadcast a message to specified room
func (b *Bucket) BroadcastRoom(arg *pb.BroadcastRoomReqWithSpan) {
	num := atomic.AddUint64(&b.routinesNum, 1) % uint64(b.c.Bucket.RoutineAmount)
	select {
	case b.routines[num] <- arg:
	default:
		log.L().Errorf("Bucket BroadcastRoom Msg Miss msg:%v", arg)
	}
}

// Rooms get all room id where online number > 0.
func (b *Bucket) Rooms() (res map[string]struct{}) {
	var (
		roomID string
		room   *Room
	)
	res = make(map[string]struct{}, len(b.rooms))
	b.cLock.RLock()
	for roomID, room = range b.rooms {
		if room.Online > 0 {
			res[roomID] = struct{}{}
		}
	}
	b.cLock.RUnlock()
	return
}

// UpRoomsCount update all room count
func (b *Bucket) UpRoomsCount(roomCountMap map[string]int32) {
	var (
		roomID string
		room   *Room
	)
	b.cLock.RLock()
	for roomID, room = range b.rooms {
		room.AllOnline = roomCountMap[roomID]
	}
	b.cLock.RUnlock()
}

// roomproc
func (b *Bucket) roomproc(c chan *pb.BroadcastRoomReqWithSpan) {
	for {
		arg := <-c
		if room := b.Room(arg.GetRoomID()); room != nil {
			room.Push(&pb.ClientPackageWithSpan{
				Span:          arg.Span,
				ClientPackage: arg.Proto,
			})
		}
	}
}
