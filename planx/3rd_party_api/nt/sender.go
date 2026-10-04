package nt

import (
	"sync"

	"github.com/astaxie/beego/httplib"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type Sender struct {
	recChan chan *Req

	wg *sync.WaitGroup
}

type Req struct {
	Url     string
	Data    interface{}
	RetChan chan []byte // 如果需要处理返回值，需要传入一个chan，将返回值以[]byte形式传回
}

func newSender(reqChan chan *Req, wg *sync.WaitGroup) *Sender {
	return &Sender{
		recChan: reqChan,
		wg:      wg,
	}
}

// 发送http请求
func send(req *Req) []byte {
	defer tilogs.PanicCatcher("nt sender send %+v panic", req)

	httpReq, err := httplib.Post(req.Url).JSONBody(req.Data)
	if err != nil {
		tilogs.L().Errorf("nt sender req %+v failed, err %s", req, err.Error())
		return nil
	}

	httpReq.SetTimeout(timeout, timeout)

	resp, err := httpReq.Bytes()
	if err != nil {
		tilogs.L().Errorf("nt sender req %+v failed, err %s", req, err.Error())
		return nil
	}

	tilogs.L().Debugf("nt sender req %+v success, resp %+v", req, string(resp))
	return resp
}

func (s *Sender) start() {
	s.wg.Add(1)
	defer s.wg.Done()

	for {
		select {
		case req, ok := <-s.recChan:
			if !ok {
				return
			}
			send(req)
		}
	}
}
