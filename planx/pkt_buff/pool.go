package pkt_buff

import (
	"sync"
)

var (
	gpPool = sync.Pool{New: func() interface{} {
		return NewGamePktBuff()
	}}

	zipPool = sync.Pool{New: func() interface{} {
		return NewZipBuff()
	}}
)
