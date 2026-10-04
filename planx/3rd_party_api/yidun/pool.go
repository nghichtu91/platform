package yidun

import (
	"sync"
)

var (
	tvReqPool = sync.Pool{
		New: func() interface{} {
			return NewTextValidReq()
		},
	}

	tvRespPool = sync.Pool{
		New: func() interface{} {
			return new(textValidResp)
		},
	}
)

func getTVReq() *textValidReq {
	req := tvReqPool.Get().(*textValidReq)
	req.reset()

	return req
}

func putTVReq(req *textValidReq) {
	tvReqPool.Put(req)
}

func getTVResp() *textValidResp {
	return tvRespPool.Get().(*textValidResp)
}

func putTVResp(resp *textValidResp) {
	tvRespPool.Put(resp)
}
