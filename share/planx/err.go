package planx

import "errors"

var (
	ErrRoleOffline          = errors.New("role is offline")
	ErrMsg                  = errors.New("message err")
	ErrSendFail             = errors.New("message send failed")
	ErrGameDataNotFound     = errors.New("gamedata not found")
	ErrGameDataFault        = errors.New("gamedata fault")
	ErrSwitchNoCase         = errors.New("switch no case")
	ErrAddChanTimeOut       = errors.New("add to chan time out")
	ErrWaitChanReplyTimeOut = errors.New("wait reply from chan time out")
	ErrTypeNoDef            = errors.New("type no define")
)
