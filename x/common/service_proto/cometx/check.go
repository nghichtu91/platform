package cometx

import (
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

var (
	pktPool = sync.Pool{
		New: func() interface{} {
			return new(protogen.ClientPackage)
		},
	}
)

var (
	loginReq = &protogen.ChatLogin{
		PlayerId: proto.String(consts.RobotCometxID),
		Token:    proto.String(consts.RobotCometxToken),
	}
	loginResp     = &protogen.ChatLoginResp{}
	joinRoomReq   = &protogen.JoinRoom{RoomId: []string{consts.RobotCometxRoom}}
	joinRoomResp  = &protogen.JoinRoomResp{}
	leaveRoomReq  = &protogen.LeaveRoom{RoomId: []string{consts.RobotCometxRoom}}
	leaveRoomResp = &protogen.LeaveRoomResp{}
)

// Check 进行cometx相关检查，目前流程如下：
// handshake -> join room -> leave room
func Check(ip string) error {
	var (
		buff []byte
	)

	conn, err := net.Dial("tcp", ip)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 发包
	sendPkt := func(id uint32, msg proto.Message) error {
		if err := conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
			return err
		}

		raw, err := proto.Marshal(msg)
		if err != nil {
			return err
		}

		data, err := TCPEncode(id, raw)
		if err != nil {
			return err
		}

		_, err = conn.Write(data)

		return err
	}

	// 收包
	receivePkt := func(msg proto.Message) error {
		buff = make([]byte, MaxCometxPktSize)

		if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return err
		}

		n, err := conn.Read(buff)
		if err != nil {
			return err
		}

		pkt := pktPool.Get().(*protogen.ClientPackage)
		defer pktPool.Put(pkt)

		err = Marshal(buff[:n], pkt)
		if err != nil {
			return err
		}

		if err := proto.Unmarshal(pkt.RawData, msg); err != nil {
			return err
		}

		return err
	}

	// 收发整合
	handle := func(id uint32, req, resp proto.Message) error {
		if err := sendPkt(id, req); err != nil {
			return err
		}

		if err := receivePkt(resp); err != nil {
			return err
		}

		// tilogs.L().Debugf("resp %+v",resp)

		return nil
	}

	// 登录
	if err := handle(uint32(protogen.ChatOperation_C2SLogin), loginReq, loginResp); err != nil {
		return err
	}

	// 进特殊房间
	if err := handle(uint32(protogen.ChatOperation_C2SJoinRoom), joinRoomReq, joinRoomResp); err != nil {
		return err
	}

	// 退出特殊房间
	if err := handle(uint32(protogen.ChatOperation_C2SLeaveRoom), leaveRoomReq, leaveRoomResp); err != nil {
		return err
	}

	return err
}
