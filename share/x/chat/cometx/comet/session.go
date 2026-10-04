package comet

import (
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/cometx/config"
	"github.com/nghichtu91/platform/share/x/chat/pkg/bufio"
)

// Session 相当于goim的channel
type Session struct {
	Rooms    map[string]*Room
	Cds      map[string]int64
	CliProto Ring
	signal   chan *pb.ClientPackageWithSpan
	Writer   bufio.Writer
	Reader   bufio.Reader

	// 禁言结束时间 0代表不是禁言
	ForbiddenEndTime int64

	Key       string // acid
	IP        string
	SessionId string // uuid
}

func NewSession(c *config.ChatConfig) *Session {
	s := new(Session)
	s.Rooms = make(map[string]*Room, 64)
	s.Cds = make(map[string]int64, 64)
	s.CliProto.Init(c.Protocol.CliProto)
	s.signal = make(chan *pb.ClientPackageWithSpan, c.Protocol.SvrProto)
	s.ForbiddenEndTime = 0
	s.SessionId = uuid.NewV4().String()
	return s
}

// Push server push message.
func (s *Session) Push(p *pb.ClientPackageWithSpan) (err error) {
	select {
	case s.signal <- p:
	default:
		// tilogs.L().Errorf("Session Push Msg channel full, key:%s, msg:%v", s.Key, p)
		tilogs.L().Warnf("Session Push Msg channel full, key:%s", s.Key)
	}
	return
}

// Ready check the channel ready or close?
func (s *Session) Ready() *pb.ClientPackageWithSpan {
	return <-s.signal
}

// Signal send signal to the channel, protocol ready.
func (s *Session) Signal() {
	s.signal <- pb.ProtoReady
}

// Close close the channel.
func (s *Session) Close() {
	s.signal <- pb.ProtoFinish
}

// room exist
func (s *Session) HasRoom(roomId string) bool {
	_, ok := s.Rooms[roomId]
	return ok
}

// room msg cd
func (s *Session) RoomCd(roomId string) {
	s.Cds[roomId] = time.Now().Unix()
}

// room cd中
func (s *Session) InRoomCd(roomId string) bool {
	stamp, ok := s.Cds[roomId]
	if !ok {
		return false
	}
	return stamp+int64(config.Cfg.Room.Cd) >= time.Now().Unix()
}
