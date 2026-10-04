package client

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	// _ "github.com/nghichtu91/platform/share/planx/util/logs"
)

type PacketID int32

const (
	PacketIDContent PacketID = iota
	PacketIDReqResp
	PacketIDPingPong
	PacketIDGateSession
	PacketIDGatePkt
	PacketChatToken = 5

	PacketIDBatchCheck = 999 // 批量检查包, used by gatex
)

var BytePing []byte

// var PacketPing *Packet

func init() {
	BytePing = []byte("PING")
}

func NewPingPacket(sid uint) *pb.ClientPacket {
	var buf bytes.Buffer
	buf.Write(BytePing)
	buf.Write([]byte(strconv.Itoa(int(timeutil.NowByShardId(sid).Unix()))))
	// fmt.Fprintf(&buf, "%d", )
	// tilogs.L().Debugf("Ping To Client %s", string(buf.Bytes()))
	_PacketId := int32(PacketIDPingPong)

	return &pb.ClientPacket{
		PacketId: &(_PacketId),
		RawData:  buf.Bytes(),
	}
}

// UpdatePingPacket 更新PingPacket内时间
func UpdatePingPacket(sid uint, cp *pb.ClientPacket) {
	var buf bytes.Buffer
	buf.Write(BytePing)
	buf.Write([]byte(strconv.Itoa(int(timeutil.NowByShardId(sid).Unix()))))

	cp.RawData = buf.Bytes()
}

func NewChatTokenPacket(token string) *pb.ClientPacket {
	var buf bytes.Buffer
	buf.Write([]byte(token))
	tilogs.L().Debugf("chat token To Client %s", token)
	_PacketId := int32(PacketChatToken)

	return &pb.ClientPacket{
		PacketId: &(_PacketId),
		RawData:  buf.Bytes(),
	}
}

func NewPacket(data []byte, id PacketID) *pb.Packet {
	pkt := &pb.Packet{
		RawData:  data,
		PacketId: int32(id),
		// crc:     uint32(0),
	}
	return pkt
}

const max_Msg_Size = 1024 * 1024

var msg_size_err = errors.New("msg_size_err")
var msg_read_err = errors.New("msg_read_err")

// Instead of binary.LittleEndian.Uint32, better performance without reflect
func ReadLittleEndianUint32(r io.Reader) (uint32, error) {
	var buf32 [4]byte
	n, err := io.ReadFull(r, buf32[:])
	if err != nil {
		if n != 0 {
			tilogs.L().Warnf("ReadLittleEndianUint32 got error with ReadFull: size(%d), %s", n, err.Error())
		}
		return 0, err
	}
	if n != 4 {
		return 0, fmt.Errorf("ReadPacket header failed, len is not 4")
	}

	ubufint32 := binary.LittleEndian.Uint32(buf32[:])
	return ubufint32, nil
}

func ReadPacket(r io.Reader, setReadDeadline func(int)) (*pb.ClientPacket, error) {
	var msgSize int32

	// TODO: by YZH 这个第一个Deadline是用来处理消息消息主循环的等待，这个方法不是很理想
	// 如果恰好读完这个头，然后引起了一个数据的Read出错timeout则是有问题的。
	// 因此在此处和下面分别设置不同的数据Deadline
	// 理应在这个消息头到达后，应该立即有后续数据跟进，因此如果后续数据未能在指定时间出现
	// 则认为timeout也应该断线。
	setReadDeadline(15)
	// message size
	if uMsgSize, err := ReadLittleEndianUint32(r); err != nil {
		return nil, err
	} else {
		msgSize = int32(uMsgSize)
	}
	// protect bigger than 1MB message, Have to protect msgSize to avoid memory exhaust problem
	if msgSize < 0 || msgSize >= max_Msg_Size {
		return nil, fmt.Errorf("%s, size:%d", msg_size_err.Error(), msgSize)
	}

	// message binary data
	setReadDeadline(5)
	readSize := msgSize + 9 // read packetId, msgId
	buf := make([]byte, readSize)
	if n, err := io.ReadFull(r, buf); err != nil {
		if n != 0 {
			tilogs.L().Warnf("ReadPacket got error with ReadFull: size(%d), %s", n, err.Error())
		}

		e, ok := err.(net.Error)
		if ok && e.Temporary() {
			if e.Timeout() {
				return nil, fmt.Errorf("ReadPacket error: the  msgbody part come too late!! you are killed.")
			}
		}
		return nil, err
	} else {
		if int32(n) != readSize {
			return nil, msg_read_err
		}
	}

	packetID := int32(binary.LittleEndian.Uint32(buf[:4]))
	msgId := binary.LittleEndian.Uint32(buf[4:8])
	bcompress := buf[8] == 1
	// tilogs.L().Infof("ReadPacket id %x", buf)
	// tilogs.L().Infof("got packet size %d, content:%s", msgSize, string(buf))
	return &pb.ClientPacket{
		PacketId:  &packetID,
		MessageId: &msgId,
		Compress:  &bcompress,
		RawData:   buf[9:],
	}, nil
}

func Read2Packet(r io.Reader, pkt *pb.ClientPacket) error {
	var (
		msgSize uint32
		n       int
		err     error
	)

	// 处理数据头
	head := headerPool.Get().([]byte)
	defer headerPool.Put(head)
	if n, err = io.ReadFull(r, head); err != nil {
		if n != 0 {
			tilogs.L().Warnf("ReadPacket got error with head ReadFull: size(%d), %s", n, err.Error())
		}

		e, ok := err.(net.Error)
		if ok && e.Temporary() {
			if e.Timeout() {
				return fmt.Errorf("ReadPacket error: the msg head part come too late!! you are killed")
			}
		}
		return err
	} else {
		if int32(n) != 13 {
			return msg_read_err
		}
	}

	// message size
	msgSize = binary.LittleEndian.Uint32(head[:4])
	// protect bigger than 1MB message, Have to protect msgSize to avoid memory exhaust problem
	if msgSize < 0 || msgSize >= max_Msg_Size {
		return fmt.Errorf("%s, size:%d", msg_size_err.Error(), msgSize)
	}

	*pkt.PacketId = int32(binary.LittleEndian.Uint32(head[4:8]))
	*pkt.MessageId = binary.LittleEndian.Uint32(head[8:12])
	*pkt.Compress = head[12] == 1

	// 处理数据包
	if cap(pkt.RawData) < int(msgSize) {
		pkt.RawData = make([]byte, msgSize)
	} else {
		pkt.RawData = pkt.RawData[:msgSize]
	}

	if n, err = io.ReadFull(r, pkt.RawData); err != nil {
		if n != 0 {
			tilogs.L().Warnf("ReadPacket got error with body ReadFull: size(%d), %s", n, err.Error())
		}

		e, ok := err.(net.Error)
		if ok && e.Temporary() {
			if e.Timeout() {
				return fmt.Errorf("ReadPacket error: the msg body part come too late!! you are killed")
			}
		}
		return err
	} else {
		if uint32(n) != msgSize {
			return msg_read_err
		}
	}

	return nil
}

func SendBytes(w io.Writer, data []byte, id PacketID) (int, error) {
	// return SendPacket(w, NewPacket(data, id)) TODO
	return 0, nil
}

func SendPacket(w io.Writer, pkt *pb.ClientPacket) (int, error) {
	frameLen := 4 + 4 + 4 + 1 // size + packetId + messageId + compress

	// 包头为定长，每次收发都会重写，
	// 不需要做额外操作，拿出来用再放回去就可以
	beBuf := headerPool.Get().([]byte)
	defer headerPool.Put(beBuf)

	size := uint32(len(pkt.RawData))

	binary.LittleEndian.PutUint32(beBuf, size)
	binary.LittleEndian.PutUint32(beBuf[4:], uint32(pkt.GetPacketId()))
	binary.LittleEndian.PutUint32(beBuf[8:], pkt.GetMessageId())
	if pkt.GetCompress() {
		beBuf[12] = 1
	} else {
		beBuf[12] = 0
	}

	n, err := w.Write(beBuf)

	if err != nil {
		return n, err
	}

	// rawData might be longer than its content.
	n, err = w.Write(pkt.RawData[:size])
	return n + frameLen, err
}
