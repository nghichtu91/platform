package feishu

import (
	"sync"
	"time"

	"github.com/astaxie/beego/httplib"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

// Sender 用于异步发送飞书请求
type Sender struct {
	// 接收发送内容
	recChan chan Req

	quit chan struct{}

	sync.Once
}

type Req struct {
	Url     string
	Data    string
	IsAtAll bool
}

// ReqJson 向飞书机器人发请求的json struct
type ReqJson struct {
	MsgType   string  `json:"msg_type"`
	Timestamp int64   `json:"timestamp"`
	Sign      string  `json:"sign"`
	Content   Content `json:"content"`
	Card      card    `json:"card"`
}

type card struct {
	Elements []element `json:"elements"`
}
type element struct {
	Tag  string `json:"tag"`
	Text text   `json:"text"`
}
type text struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}
type Content struct {
	Text string `json:"text"`
}

func (s *Sender) start() {
	for {
		select {
		case req := <-s.recChan:
			func(req Req) {
				defer tilogs.PanicCatcher("feishu sender send data %s panic", data)
				if req.IsAtAll {
					es := make([]element, 1)
					es[0] = element{
						Tag:  "div",
						Text: text{Content: req.Data + "<at id=all></at>", Tag: "lark_md"},
					}
					r, err := httplib.Post(req.Url).SetTimeout(timeout, timeout).JSONBody(&ReqJson{
						MsgType:   "interactive",
						Timestamp: time.Now().Unix(),
						Sign:      getSign(),
						Card:      card{es},
					})
					if err != nil {
						tilogs.L().Errorf("feishu post err %v", err)
					} else {
						// 只发，不处理异常
						resp, err := r.Response()
						if err != nil {
							tilogs.L().Warnf("feishu post err %v", err)
						} else {
							resp.Body.Close()
						}
					}
				} else {
					r, err := httplib.Post(req.Url).SetTimeout(timeout, timeout).JSONBody(&ReqJson{
						MsgType:   "text",
						Timestamp: time.Now().Unix(),
						Sign:      getSign(),
						Content: Content{
							Text: req.Data,
						},
					})
					if err != nil {
						tilogs.L().Errorf("feishu post err %v", err)
					} else {
						// 只发，不处理异常
						resp, err := r.Response()
						if err != nil {
							tilogs.L().Warnf("feishu post err %v", err)
						} else {
							resp.Body.Close()
						}
					}
				}
			}(req)
		case <-s.quit:
			tilogs.L().Infof("feishu sender quit")
			return
		}
	}
}

func (s *Sender) stop() {
	s.Once.Do(func() {
		close(s.quit)
	})
}
