package nt

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var (
	sendChan chan *Req

	isOpen int32

	wg sync.WaitGroup
)

func Start() {
	sendChan = make(chan *Req, sendChanSize*runtime.NumCPU())

	for i := 0; i < runtime.NumCPU(); i++ {
		go newSender(sendChan, &wg).start()
	}

	atomic.StoreInt32(&isOpen, 1)
}

// Send 异步发送http请求，避免Handler阻塞
// 不需要返回结果时，使用这个接口
func Send(url string, data interface{}) error {
	if atomic.LoadInt32(&isOpen) == 0 {
		return ErrNtNotInit
	}

	req := &Req{
		Url:  url,
		Data: data,
	}

	select {
	case sendChan <- req:
	default:
		tilogs.L().Errorf("nt send chan full, req %+v", req)
		return ErrSendChannelFull
	}

	return nil
}

// Request 同步发送http请求
// 需要返回值的请求，如果放sender里发，并发请求时会有很高延迟
// 所以这里直接同步发送
func Request(url string, data interface{}) []byte {
	return send(&Req{
		Url:  url,
		Data: data,
	})
}

func Stop() {
	atomic.CompareAndSwapInt32(&isOpen, 1, 0)

	close(sendChan)

	wg.Wait()
}
