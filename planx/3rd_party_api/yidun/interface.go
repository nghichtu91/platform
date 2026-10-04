package yidun

type ITextValidateReq interface {
	GetText() string
	GetAcid() string
	GetNickname() string
	GetReceiver() string
	GetDeviceID() string
	GetIp() string
	GetNtID() string
	GetDeviceOS() string
	GetMessageType() uint64
}
