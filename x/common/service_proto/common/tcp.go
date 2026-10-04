package common

import (
	"errors"
)

var (
	ErrRawMsgSizeExceed   = errors.New("raw msg size exceed")
	ErrRawMsgSizeInvalid  = errors.New("raw msg size invalid")
	ErrMarshalTargetIsNil = errors.New("marshal target is nil")
)
