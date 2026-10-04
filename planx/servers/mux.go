package servers

import (
	"sync"
	"sync/atomic"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type Request struct {
	Code     uint32
	RawBytes []byte
}

const SYSTEM_TICK_30_CODE = 30

var SYSTEM_TICK_30_REQUEST = Request{Code: SYSTEM_TICK_30_CODE, RawBytes: nil}

type Response struct {
	Code     uint32
	RawBytes []byte

	// 用来统计各种错误码
	RespCode int
}

type HandlerFunc func(Request) *Response

func (f HandlerFunc) Serve(r Request) *Response {
	return f(r)
}

type Handler interface {
	Serve(Request) *Response
}

type RequestHook interface {
	PreRequest(Request)
	PostRequest(*Response) // *Response可能为nil
}

//Mux 请给每个玩家配置一个该数据实例
//nRequests代表了每个玩家的单调增长的统计，可以采样这个数据然后利用这个数据计算出玩家QPS
type Mux struct {
	mu        sync.RWMutex
	m         map[uint32]Handler
	hooks     []RequestHook
	nRequests uint64 //请求的数量
}

func NewMux() *Mux {
	m := &Mux{
		m:     make(map[uint32]Handler),
		hooks: make([]RequestHook, 0, 4),
	}
	m.RegisterRequestHook(m)
	return m
}

func (mux *Mux) PreRequest(Request) {
	atomic.AddUint64(&mux.nRequests, 1)
}
func (mux *Mux) PostRequest(*Response) {
}

// 注册Hook，不能注销
func (mux *Mux) RegisterRequestHook(h RequestHook) {
	mux.hooks = append(mux.hooks, h)
}

func (mux *Mux) runPreHooks(r Request) {
	for _, h := range mux.hooks {
		h.PreRequest(r)
	}
}

func (mux *Mux) runPostHooks(resp *Response) {
	for _, h := range mux.hooks {
		h.PostRequest(resp)
	}
}

func (mux *Mux) GetNumberRequests() uint64 {
	return mux.nRequests
}

// Serve HTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (mux *Mux) Serve(r Request) *Response {

	//Try to get full path directly
	h, ok := mux.m[r.Code]
	if ok {
		mux.runPreHooks(r)
		var resp *Response
		resp = h.Serve(r)
		//if resp != nil {
		mux.runPostHooks(resp) // resp为nil的时候也调用
		//}
		return resp
	} else {
		tilogs.L().Warnf("Serve %d no handler!", r.Code)
	}

	return nil
}

func (mux *Mux) Handle(pattern uint32, handler Handler) {

	if pattern == 0 {
		panic("mux: invalid pattern 0")
	}
	if handler == nil {
		panic("mux: nil handler")
	}

	if _, ok := mux.m[pattern]; ok {
		tilogs.L().Warnf("mux.packetconn router %d register got duplicated, last one works!", pattern)
	}
	mux.m[pattern] = handler

}

// HandleFunc registers the handler function for the given pattern.
func (mux *Mux) HandleFunc(pattern uint32, handler func(Request) *Response) {
	mux.Handle(pattern, HandlerFunc(handler))
}

func (mux *Mux) RemoveFunc(pattern uint32) {

	delete(mux.m, pattern)
}

// DefaultMux is the default Mux used by Serve.
var DefaultMux = NewMux()

func Handle(pattern uint32, handler Handler) {
	DefaultMux.Handle(pattern, handler)
}

// HandleFunc registers the handler function for the given pattern
// in the DefaultMux.
// The documentation for Mux explains how patterns are matched.
func HandleFunc(pattern uint32, handler func(Request) *Response) {
	DefaultMux.HandleFunc(pattern, handler)
}
