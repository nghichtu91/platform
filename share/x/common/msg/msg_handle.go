package msg

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/nghichtu91/platform/share/planx/secure"
)

var (
	ErrPkgSizeExceed             = errors.New("packet size exceed limit")
	ErrInvalidHandShakeLines     = errors.New("invalid handshake packet lines")
	ErrInvalidHandShakeRespLines = errors.New("invalid handshake resp lines")
)

const (
	MaxHandShakePkgSize = 1024 // 握手包最大限制
	lineBreakWin        = "\r\n"
	lineBreakUnix       = "\n"
)

// 客户端握手包每行序号
// 也需要和客户端完全对应

const (
	IdxToken = iota
	IdxGzipLimit
	IdxChannelID
	IdxBDCId
	IdxVersion
	IdxDeviceType
	IdxDeviceSystem
	IdxUpdateVer
	IdxIP
	IdxIDFA
	IdxDeviceMemory
	IdxAddressFlag
	IdxDeviceID
	IdxAppChannelID
	IdxSDKChannelUID
	IdxBDCCommonInfo
	IdxLanguage
	IdxClientBuildTimeAndHotVer
	IdxBDCDeviceID
	IdxSubChannelID
	IdxClientPlatform
	IdxDistinctID
	IdxClientType
	IdxRevers1 // 保留行1
	IdxRevers2 // 保留行2
	IdxRevers3 // 保留行3
	IdxHandShakeCount
)

const (
	IdxResult = iota
	IdxEncAcID
	IdxSrvTS
	IdxVer
	IdxEncNumID
	IdxReserve // 额外的换行
	IdxRespCount
)

func NewBuffPool(size int) *sync.Pool {
	return &sync.Pool{New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, size))
	}}
}

func SplitByLine(v string) []string {
	// 牺牲一些性能换取可读性
	v = strings.ReplaceAll(v, lineBreakWin, lineBreakUnix)
	return strings.Split(v, lineBreakUnix)
}

// Marshal 握手信息转换为可以直接发送的byte slice
func (info *ClientHandShakeInfo) Marshal(buff *bytes.Buffer) {
	w := func(v string) { buff.Write([]byte(v + lineBreakUnix)) }

	// 有顺序要求，新增需要和客户端核对新字段所在位置
	w(info.LoginToken)
	w(info.GZipLimitS)
	w(info.ChannelId)
	w(info.BDCId)
	w(info.Version)
	w(info.DeviceType)
	w(info.DeviceSystem)
	w(info.UpdateVer)
	w(info.IP)
	w(info.IDFA)
	w(info.DeviceMemory)
	w(info.AddressFlag)
	w(info.DeviceId)
	w(info.AppChannelId)
	w(info.SdkChannelUid)
	w(info.BdcCommonInfo)
	w(info.Language)
	w(info.ClientBuildTimeAndHotVer)
	w(info.BDCDeviceId)
	w(info.SubChannelID)
	w(info.ClientPlatform)
	w(info.DistinctID)
	w(info.ClientType)

	// 保留空行*3
	for i := 0; i < 3; i++ {
		w("reserved line")
	}
}

// Unmarshal buff信息转换握手信息
func (info *ClientHandShakeInfo) Unmarshal(buff *bytes.Buffer) error {
	// 限制握手包大小，如果包太大直接认为非法
	if buff.Len() >= MaxHandShakePkgSize {
		return ErrPkgSizeExceed
	}

	parts := SplitByLine(buff.String())
	if len(parts) != IdxHandShakeCount {
		return ErrInvalidHandShakeLines
	}

	info.LoginToken = parts[IdxToken]
	info.GZipLimitS = parts[IdxGzipLimit]
	info.ChannelId = parts[IdxChannelID]
	info.BDCId = parts[IdxBDCId]
	info.Version = parts[IdxVersion]
	info.DeviceType = parts[IdxDeviceType]
	info.DeviceSystem = parts[IdxDeviceSystem]
	info.UpdateVer = parts[IdxUpdateVer]
	info.IP = parts[IdxIP]
	info.IDFA = parts[IdxIDFA]
	info.DeviceMemory = parts[IdxDeviceMemory]
	info.AddressFlag = parts[IdxAddressFlag]
	info.DeviceId = parts[IdxDeviceID]
	info.AppChannelId = parts[IdxAppChannelID]
	info.SdkChannelUid = parts[IdxSDKChannelUID]
	info.BdcCommonInfo = parts[IdxBDCCommonInfo]
	info.Language = parts[IdxLanguage]
	info.ClientBuildTimeAndHotVer = parts[IdxClientBuildTimeAndHotVer]
	info.BDCDeviceId = parts[IdxBDCDeviceID]
	info.SubChannelID = parts[IdxSubChannelID]
	info.ClientPlatform = parts[IdxClientPlatform]
	info.DistinctID = parts[IdxDistinctID]
	info.ClientType = parts[IdxClientType]

	return nil
}

// Unmarshal buff信息转握手响应
func (resp *HandShakeResp) Unmarshal(buff *bytes.Buffer) error {
	parts := SplitByLine(buff.String())
	switch len(parts) {
	case 1:
		resp.Result = parts[IdxResult]
		return nil
	case IdxRespCount:
	default:
		return ErrInvalidHandShakeRespLines
	}

	resp.Result = parts[IdxResult]
	resp.EncAcID = parts[IdxEncAcID]
	resp.SrvTS = parts[IdxSrvTS]
	resp.Ver = parts[IdxVer]
	resp.EncNumID = parts[IdxEncNumID]

	v, err := secure.Decode64FromNet(resp.EncAcID)
	if err != nil {
		return err
	}
	resp.AccountID = string(v)

	v, err = secure.Decode64FromNet(resp.EncNumID)
	if err != nil {
		return err
	}
	num, err := strconv.Atoi(string(v))
	if err != nil {
		return err
	}
	resp.NumID = num

	return nil
}
