package protocol

import (
	"encoding/binary"

	"github.com/golang/protobuf/proto"
)

const (
	_header_size = 8
)

func generateMsg(message proto.Message) []byte {
	buf, err := proto.Marshal(message)
	if err != nil {
		return nil
	}
	return buf
}

func generateCommon(messageId uint32, protoSize int) []byte {
	buf := make([]byte, _header_size)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(protoSize+_header_size))
	binary.LittleEndian.PutUint32(buf[4:8], messageId)
	return buf
}
