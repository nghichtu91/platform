package pkt_buff

import (
	"encoding/binary"
	"net"

	"github.com/golang/protobuf/proto"
)

const (
	// 默认的buff长度
	defaultBuffSize = 8192

	// 下行数据消息大于此大小就开始压缩，此值和客户端一致，若要改需要和客户端一起改
	compressSize = 1024
)

var (
	pingRaw = []byte{4, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 80, 73, 78, 71} // 基于mp0041的PING包，后续有变化需要手动改
)

// GamePktBuff 游戏TCP包的缓存结构
type GamePktBuff struct {
	// 包长度，仅发送时使用
	// 4字节
	pktLen []byte

	// packetId + messageId + isCompressed
	// 10字节
	head []byte

	// 处理消息用
	pb *proto.Buffer
}

func NewGamePktBuff() *GamePktBuff {
	return &GamePktBuff{
		pktLen: make([]byte, 4),
		head:   make([]byte, 9),
		pb:     proto.NewBuffer(make([]byte, 0, defaultBuffSize)),
	}
}

// SendGamexMsg 直接发送message
func SendGamexMsg(conn net.Conn, pktID, msgID uint32, msg proto.Message) error {
	var (
		err error
	)

	gb := gpPool.Get().(*GamePktBuff)
	defer func() {
		gb.pb.Reset()
		gpPool.Put(gb)
	}()

	err = gb.pb.Marshal(msg)
	if err != nil {
		return err
	}

	binary.LittleEndian.PutUint32(gb.head, pktID)
	binary.LittleEndian.PutUint32(gb.head[4:], msgID)

	msgSize := len(gb.pb.Bytes())
	if msgSize > compressSize {
		zb := GetZipBuff()
		defer PutZipWBuff(zb)

		err = zb.Zip(gb.pb.Bytes())
		if err != nil {
			return err
		}

		// 长度
		binary.LittleEndian.PutUint32(gb.pktLen, uint32(len(zb.b.Bytes())))
		// zip标记位
		gb.head[8] = 1

		return sendConn(conn, gb.pktLen, gb.head, zb.b.Bytes())
	} else {
		// 长度
		binary.LittleEndian.PutUint32(gb.pktLen, uint32(len(gb.pb.Bytes())))
		// zip标记位
		gb.head[8] = 0

		return sendConn(conn, gb.pktLen, gb.head, gb.pb.Bytes())
	}
}

// SendGamexPing 模拟客户端发送PING
func SendGamexPing(conn net.Conn) error {
	return sendConn(conn, pingRaw)
}

func sendConn(conn net.Conn, raw ...[]byte) error {
	var (
		err error
	)

	for i := 0; i < len(raw); i++ {
		_, err = conn.Write(raw[i])
		if err != nil {
			return err
		}
	}

	return nil
}
