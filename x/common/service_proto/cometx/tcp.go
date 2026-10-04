package cometx

import (
	"encoding/binary"

	"github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/common/service_proto/common"
)

const (
	// MaxCometxPktSize
	// Cometx TCP最大长度，超过会被cometx拦截
	MaxCometxPktSize = 10240 + cometxOffset

	// 通用长度
	sizeLen  = 4
	msgIDLen = 4

	// cometx包
	cometxSizeOffset  = sizeLen            // 4字节，包长度
	cometxMsgIDOffset = sizeLen + msgIDLen // 4字节，包ID
	cometxOffset      = sizeLen + msgIDLen
)

// TCPEncode Cometx使用的Tcp包编码
func TCPEncode(msgID uint32, raw []byte) ([]byte, error) {
	size := len(raw) + cometxOffset
	if size >= MaxCometxPktSize {
		return nil, common.ErrRawMsgSizeExceed
	}

	msg := make([]byte, size)
	// size
	// 与客户端tcp不一样，size包括了tcp头的长度
	binary.LittleEndian.PutUint32(msg[:cometxSizeOffset], uint32(size))
	// msgID
	binary.LittleEndian.PutUint32(msg[cometxSizeOffset:cometxMsgIDOffset], msgID)
	// data
	copy(msg[cometxOffset:size], raw)

	return msg, nil
}

// Marshal Cometx的Tcp解码过程
// 未优化，如果需要高性能，需要配合各种buffer重新实现
func Marshal(data []byte, pkg *protogen.ClientPackage) error {
	if pkg == nil {
		return common.ErrMarshalTargetIsNil
	}

	size := len(data)
	if size > MaxCometxPktSize {
		return common.ErrRawMsgSizeExceed
	}
	if size < cometxOffset {
		return common.ErrRawMsgSizeInvalid
	}
	dataLen := binary.LittleEndian.Uint32(data[:cometxSizeOffset])
	if size != int(dataLen) {
		return common.ErrRawMsgSizeInvalid
	}

	pkg.MessageId = proto.Uint32(binary.LittleEndian.Uint32(data[cometxSizeOffset:cometxMsgIDOffset]))

	if msgLen := dataLen - cometxOffset; msgLen > 0 {
		pkg.RawData = make([]byte, msgLen)
		copy(pkg.RawData, data[cometxOffset:])
	}

	return nil
}
