package gatex

import (
	"encoding/binary"

	"github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/x/common/msg/gategamex/pb"
	"github.com/nghichtu91/platform/share/x/common/service_proto/common"
)

const (
	// MaxClientPktSize
	// Client TCP最大长度1024*1024，超过会被gate拦截
	MaxClientPktSize = 1048576

	// 通用长度
	sizeLen   = 4
	pkgIDLen  = 4
	msgIDLen  = 4
	isCompLen = 1

	// gamex包
	clientSizeOffset  = sizeLen                                   // 4字节，包长度
	clientPkgIDOffset = sizeLen + pkgIDLen                        // 4字节，PackageID
	clientMsgIDOffset = sizeLen + pkgIDLen + msgIDLen             // 4字节，MessageID
	clientCompOffset  = sizeLen + pkgIDLen + msgIDLen + isCompLen // 1字节，是否使用压缩
	clientOffset      = sizeLen + pkgIDLen + msgIDLen + isCompLen
)

// ClientTCPEncode 客户端常规TCP协议包编码
func ClientTCPEncode(pkgID, msgID uint32, isCompress bool, raw []byte) ([]byte, error) {
	size := len(raw) + clientOffset
	if size > MaxClientPktSize {
		return nil, common.ErrRawMsgSizeExceed
	}

	msg := make([]byte, size)
	// size
	// 只包括raw长度
	binary.LittleEndian.PutUint32(msg[:clientSizeOffset], uint32(len(raw)))
	// PackageID
	binary.LittleEndian.PutUint32(msg[clientSizeOffset:clientPkgIDOffset], pkgID)
	// MessageID
	binary.LittleEndian.PutUint32(msg[clientPkgIDOffset:clientMsgIDOffset], msgID)
	// IsCompress
	if isCompress {
		msg[clientCompOffset] = 1
	}
	// data
	copy(msg[clientOffset:size], raw)

	return msg, nil
}

// ClientTCPMarshal 客户端常规TCP协议包解码
// 未优化，如果需要高性能，需要配合各种buffer重新实现
func ClientTCPMarshal(data []byte, pkt *pb.ClientPacket) error {
	if pkt == nil {
		return common.ErrMarshalTargetIsNil
	}

	// 校验
	size := len(data)
	if size < clientSizeOffset {
		return common.ErrRawMsgSizeInvalid
	}
	if size > MaxClientPktSize {
		return common.ErrRawMsgSizeExceed
	}
	rawLen := binary.LittleEndian.Uint32(data[:clientSizeOffset])
	if size-clientOffset != int(rawLen) {
		return common.ErrRawMsgSizeInvalid
	}

	// decode
	pkt.PacketId = proto.Int32(int32(binary.LittleEndian.Uint32(data[clientSizeOffset:clientPkgIDOffset])))
	pkt.MessageId = proto.Uint32(binary.LittleEndian.Uint32(data[clientPkgIDOffset:clientCompOffset]))
	if data[clientCompOffset] == 1 {
		pkt.Compress = proto.Bool(true)
	}
	pkt.RawData = make([]byte, rawLen)
	copy(pkt.RawData, data[clientOffset:])

	return nil
}
