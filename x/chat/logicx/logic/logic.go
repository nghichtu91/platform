package logic

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nats-io/nats.go"

	"github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/planx/nats_cli"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/logicx/config"
	"github.com/nghichtu91/platform/share/x/chat/logicx/logic/dao"
	"github.com/nghichtu91/platform/share/x/chat/logicx/logic/handles"
)

const (
	_onlineTick     = time.Second * 10
	_onlineDeadline = time.Minute * 5
	_forLoopMax     = 10
)

// Logic struct
type Logic struct {
	dao *dao.Dao

	ConnNum     int32
	ConnNumLock sync.RWMutex

	isOpen bool

	subscriptions []*nats.Subscription
}

// New init
func New() (l *Logic) {
	l = &Logic{
		dao:           dao.New(),
		subscriptions: make([]*nats.Subscription, 0, 4),
	}
	l.InitGamexNatsPublish()
	return l
}

// Ping ping resources is ok.
func (l *Logic) Ping(c context.Context) (err error) {
	// return l.dao.Ping(c)
	return
}

// Close close resources.
func (l *Logic) Close() {
	// l.dao.Close()
}

// etcd's interface implement
// 获得连接数
func (l *Logic) GetConnNum() int32 {
	l.ConnNumLock.RLock()
	defer l.ConnNumLock.RUnlock()

	return l.ConnNum
}

// etcd's interface implement
// 服务是否开启
func (l *Logic) IsOpen() bool {
	return l.isOpen
}

func (l *Logic) InitGamexNatsPublish() {
	gid := strconv.Itoa(int(config.Cfg.Gid))
	sub, _ := nats_cli.GetHandlersMgr().RegQueueNoSubWithLock(nats_cli.ChatGamexSubj(gid),
		&handles.PushMsgHandle{
			Cb: l,
		})
	sub2, _ := nats_cli.GetHandlersMgr().RegQueueNoSubWithLock(nats_cli.LogicValidSubj(gid),
		&handles.TextValidHandler{})
	l.subscriptions = append(l.subscriptions, sub, sub2)
}

func (l *Logic) Downline() {
	for _, sub := range l.subscriptions {
		sub.Unsubscribe()
	}

	tilogs.L().Infof("logicx Downline subscription.Unsubscribe")
}

type splitIndex struct {
	cont string
	// 是否是标记字符串
	isMark bool
}

/*func sensitiveReplace(content string) string {
	r, _ := regexp.Compile("<a href=.*</a>")
	if r == nil {
		// 一旦有错误 就整个字符串做敏感词检查
		return illegalwords.Replace(content)
	}

	splitRet := make([]splitIndex, 0)
	for i := 0; i < _forLoopMax; i++ {
		mark := r.FindStringIndex(content)
		if content == "" {
			break
		}
		if mark == nil {
			// 正常内容
			splitRet = append(splitRet, splitIndex{
				cont:   content,
				isMark: false,
			})
			break
		}
		b, e := mark[0], mark[1]

		if mark[0] == 0 {
			// 标记字符串
			splitRet = append(splitRet, splitIndex{
				cont:   content[b:e],
				isMark: true,
			})
		} else {
			// 正常内容
			splitRet = append(splitRet, splitIndex{
				cont:   content[0:b],
				isMark: false,
			})
			// 标记字符串
			splitRet = append(splitRet, splitIndex{
				cont:   content[b:e],
				isMark: true,
			})
		}
		content = content[e:]
	}

	var finalRet string
	// 开始分段检测
	for _, s := range splitRet {
		if !s.isMark {
			s.cont = illegalwords.Replace(s.cont)
		}
		// 再将分段检测结果 拼回原字符串
		finalRet += s.cont
	}

	return finalRet
}
*/
func (l *Logic) Push(fatherSpan opentracing.Span, p *pb.GamexChatMsg) *pb.GamexChatMsgRet {
	var (
		err error
	)
	resp := &pb.GamexChatMsgRet{
		ErrInfo: proto.String(""),
	}

	// 是否需要进行敏感词检测
	//if p.GetIsNeedSentiveCheck() && len(p.GetContent()) > 0 && !chat_util.IsPureEmoji(p.GetContent()) {
	//	// 替换敏感词后 改变原content
	//	p.Content = proto.String(illegalwords.Replace(p.GetContent()))
	//}

	now := timeutil.Now().Unix()
	if p.GetType() == pb.ChatType_Personal {
		err = l.PushKey(context.Background(), fatherSpan, p, now)
	} else if p.GetType() == pb.ChatType_Room {
		err = l.PushRoom(context.Background(), fatherSpan, p, now)
	}

	if err != nil {
		resp.ErrInfo = proto.String(err.Error())
	}
	return resp
}
