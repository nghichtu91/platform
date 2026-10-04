package nats_cli

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/util"

	"github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
	"github.com/siddontang/go/sync2"
	"google.golang.org/protobuf/proto"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/nats_cli/pb"
	"github.com/nghichtu91/platform/share/planx/pbbuff"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tracing"
)

type INatsConn interface {
	Drain() error
	Flush() error
	Close()
	Request(subj, msgName string, data []byte, timeout time.Duration) (*nats.Msg, error)
	Publish(subj, msgName string, data []byte) error
	Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error)
	QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error)
}

var (
	natsClient INatsConn = nil

	// 是否为主动关闭
	isCloseActive sync2.AtomicBool

	pmo = new(proto.MarshalOptions)
)

const (
	msgAlarmLen = 1024 * 1024 * 6 // size of msgAlarm is 6MB
)

func init() {
	pbbuff.Init()
}

func GetNatsClient() INatsConn {
	if isCloseActive.Get() {
		return nil
	}
	return natsClient
}

// 若多个，则用","隔开，如：nats://127.0.0.1:4222,nats://127.0.0.1:4222
func InitNats(gid, natsAddr string, isLive bool) error {
	isCloseActive.Set(false)

	cli, err := nats.Connect(natsAddr,
		nats.Timeout(5*time.Second),
		nats.PingInterval(1*time.Second),
		nats.MaxPingsOutstanding(10),
		nats.MaxReconnects(-1),            // 无限次数重连
		nats.ReconnectWait(1*time.Second), // 重连间隔
		nats.DisconnectErrHandler(onDisconnect),
		// nats.ClosedHandler(onClosed),
		nats.ReconnectHandler(onReconnect),
	)

	if err != nil {
		return err
	}

	natsClient = &natsWrapper{
		gid:    gid,
		Conn:   cli,
		isLive: isLive,
	}

	return nil
}

func CloseNats() {
	if natsClient != nil && !isCloseActive.Get() {
		natsClient.Drain()
		isCloseActive.Set(true)
		natsClient.Flush()
		natsClient.Close()
		tilogs.L().Infof("CloseNats")
	}
}

func onDisconnect(conn *nats.Conn, err error) {
	if !isCloseActive.Get() {
		tilogs.L().Alarm("disconnect from nats server,err=%v LastError=%v", err, conn.LastError())
	}
	// natsClient = nil
}

func onClosed(*nats.Conn) {
	tilogs.L().Infof("close nats connection success")
	// natsClient = nil
}

func onReconnect(conn *nats.Conn) {
	tilogs.L().Alarm("reconnect to nats server")
	// natsClient = conn
	isCloseActive.Set(false)
}

// SendMsg 发送消息到指定的msgKey,异步
func SendMsg(scope []string, msg proto.Message) error {
	if GetNatsClient() == nil {
		return fmt.Errorf("can not connect to nats")
	}

	if len(scope) <= 0 {
		return fmt.Errorf("nats_cli.Request len(scope) <= 0")
	}

	return sendMsg(scope, msg, nil)
}

func sendMsg(scope []string, msg proto.Message, cb []byte) error {
	return sendMsgWithTopic(msg, cb, strings.Join(scope, "."))
}

// SendMsgWithTopic 发送消息到指定的msgKey,异步
func SendMsgWithTopic(fatherSpan opentracing.Span, msg proto.Message, topics ...string) error {
	if GetNatsClient() == nil {
		return fmt.Errorf("can not connect to nats")
	}

	if len(topics) <= 0 {
		return fmt.Errorf("nats_cli.SendMsgWithTopic len(topics) <= 0")
	}

	if fatherSpan != nil {
		bs, err := tracing.InjectSpanBytes(fatherSpan.Context())
		if err != nil {
			return fmt.Errorf("nats_cli.Request span from context fail, err %s", err.Error())
		}
		return sendMsgWithTopic(msg, bs, topics...)
	}

	return sendMsgWithTopic(msg, nil, topics...)
}

func sendMsgWithTopic(msg proto.Message, cb []byte, topics ...string) error {
	var (
		w           *pb.NatsWrapper
		msgName     string
		buff, buff2 *bytes.Buffer
		wBytes      []byte
		err         error
	)

	w = GetNatsWrapper(w)
	buff = pbbuff.GetBuffer()
	buff2 = pbbuff.GetBuffer()
	defer func() {
		pbbuff.PutBuffer(buff2, &wBytes)
		pbbuff.PutBuffer(buff, &wBytes)
		PutNatsWrapper(w)
	}()

	// 处理msg
	if cb != nil {
		w.Ctx = cb
	}
	w.Content, err = pmo.MarshalAppend(buff.Bytes(), msg)
	if err != nil {
		return err
	}

	msgName = reflect.TypeOf(msg).Elem().String()
	w.Name = msgName

	// 处理natsWrapper
	wBytes, err = pmo.MarshalAppend(buff2.Bytes(), w)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		if topic == "" {
			continue
		}
		err = natsClient.Publish(topic, msgName, wBytes)
		if err != nil {
			tilogs.L().Errorf("nats_cli.sendMsgWithTopic publish failed, err %s", err.Error())
			continue
		}
	}

	// metrics.SimpleSend("natsmsg."+msgName, strconv.Itoa(len(wBytes)))

	return nil
}

// SendMsgWithCtx 带链路追踪的SendMsg方法
func SendMsgWithCtx(fatherSpan opentracing.Span, scope []string, msg proto.Message) error {
	if GetNatsClient() == nil {
		return fmt.Errorf("can not connect to nats")
	}

	if len(scope) <= 0 {
		return fmt.Errorf("nats_cli.Request len(scope) <= 0")
	}

	var bs []byte
	if fatherSpan != nil {
		var err error
		bs, err = tracing.InjectSpanBytes(fatherSpan.Context())
		if err != nil {
			return fmt.Errorf("nats_cli.Request span from context fail, err %s", err.Error())
		}
	}

	return sendMsg(scope, msg, bs)
}

// Request 同步获取数据
// scope为前缀范围，如：要给指定shard发，则scope就是shardId；要给指定跨服分组发，就是分组id；要广播，就填空字符串
func Request(scope []string, req proto.Message) ([]byte, error) {
	if GetNatsClient() == nil {
		return nil, fmt.Errorf("nats_cli.Request can not connect to nats")
	}
	if len(scope) <= 0 {
		return nil, fmt.Errorf("nats_cli.Request len(scope) <= 0")
	}

	return request(scope, req)
}

func wrapResp(ret []byte, reqErr error, resp proto.Message) error {
	if reqErr != nil {
		return fmt.Errorf("request error %w", reqErr)
	}
	unmarshalErr := proto.Unmarshal(ret, resp)
	if unmarshalErr != nil {
		return fmt.Errorf("unmarshal %v error %w", util.GetTypeName(resp), unmarshalErr)
	}
	return nil
}

func RequestResponse(scope []string, req proto.Message, resp proto.Message) error {
	return RequestResponseWithTimeout(planx.UpTimeOut, scope, req, resp)
}

func RequestResponseWithTimeout(timeout time.Duration, scope []string, req proto.Message, resp proto.Message) error {
	ret, reqErr := RequestWithTimeoutCtx(scope, req, timeout, nil)
	return wrapResp(ret, reqErr, resp)

}

// RequestWithTimeoutCtx
// 自定义超时同步获取数据，并携带对应context消息
// scope为前缀范围，如：要给指定shard发，则scope就是shardId；要给指定跨服分组发，就是分组id；要广播，就填空字符串
func RequestWithTimeoutCtx(scope []string, req proto.Message, timeout time.Duration, bs []byte) ([]byte, error) {
	if GetNatsClient() == nil {
		return nil, fmt.Errorf("nats_cli.Request can not connect to nats")
	}
	if len(scope) <= 0 {
		return nil, fmt.Errorf("nats_cli.Request len(scope) <= 0")
	}

	return requestWithTimeOut(scope, req, timeout, bs)
}

// RequestWithTimeout
// 自定义超时同步获取数据
// scope为前缀范围，如：要给指定shard发，则scope就是shardId；要给指定跨服分组发，就是分组id；要广播，就填空字符串
func RequestWithTimeout(scope []string, req proto.Message, timeout time.Duration) ([]byte, error) {
	return RequestWithTimeoutCtx(scope, req, timeout, nil)
}

func requestWithTimeOut(scope []string, req proto.Message, timeout time.Duration, bs []byte) ([]byte, error) {
	var (
		w           *pb.NatsWrapper
		buff, buff2 *bytes.Buffer
		wBytes      []byte
		err         error
	)

	w = GetNatsWrapper(w)
	buff = pbbuff.GetBuffer()
	buff2 = pbbuff.GetBuffer()
	defer func() {
		pbbuff.PutBuffer(buff2, &wBytes)
		pbbuff.PutBuffer(buff, &wBytes)
		PutNatsWrapper(w)
	}()

	msgName := reflect.TypeOf(req).Elem().String()
	// t := time.Now().UnixNano()
	// defer metrics.ReqTimeStatistics("natsreq", strings.TrimPrefix(msgName, "protogen."), t)

	// 处理req
	if bs != nil {
		w.Ctx = bs
	}
	w.Content, err = pmo.MarshalAppend(buff.Bytes(), req)
	if err != nil {
		return nil, err
	}
	w.Name = msgName

	// 处理natsWrapper
	wBytes, err = pmo.MarshalAppend(buff2.Bytes(), w)
	if err != nil {
		return nil, err
	}

	replyMsg, err := natsClient.Request(strings.Join(scope, "."), msgName, wBytes, timeout)
	if err != nil {
		return nil, err
	}

	// metrics.SimpleSend("natsreq."+msgName, strconv.Itoa(len(wBytes)))

	return replyMsg.Data, err
}

func request(scope []string, req proto.Message) ([]byte, error) {
	return requestWithTimeOut(scope, req, planx.UpTimeOut, nil)
}

// RequestWithCtx 带链路追踪的Request方法
func RequestWithCtx(ctx context.Context, scope []string, req proto.Message) ([]byte, error) {
	if GetNatsClient() == nil {
		return nil, fmt.Errorf("nats_cli.Request can not connect to nats")
	}
	if len(scope) <= 0 {
		return nil, fmt.Errorf("nats_cli.Request len(scope) <= 0")
	}

	bs, err := tracing.InjectCtxBytes(ctx)
	if err != nil {
		return nil, fmt.Errorf("nats_cli.Request span from context fail, err %s", err.Error())
	}
	timeOut := planx.UpTimeOut
	dl, ok := ctx.Deadline()
	if ok {
		timeOut = dl.Sub(time.Now())
	}
	return requestWithTimeOut(scope, req, timeOut, bs)
}

func RequestWithSpan(fatherSpan opentracing.Span, scope []string, req proto.Message, timeout int64) ([]byte, error) {
	if GetNatsClient() == nil {
		return nil, fmt.Errorf("nats_cli.RequestWithSpan can not connect to nats")
	}
	if len(scope) <= 0 {
		return nil, fmt.Errorf("nats_cli.RequestWithSpan len(scope) <= 0")
	}
	_timeOut := planx.UpTimeOut
	if timeout > 0 {
		_timeOut = time.Duration(timeout)
	}
	if fatherSpan != nil {
		bs, err := tracing.InjectSpanBytes(fatherSpan.Context())
		if err != nil {
			return nil, fmt.Errorf("nats_cli.RequestWithSpan span from fatherSpanfail, err %s", err.Error())
		}
		return requestWithTimeOut(scope, req, _timeOut, bs)
	} else {
		return requestWithTimeOut(scope, req, planx.UpTimeOut, nil)
	}

}

type natsWrapper struct {
	gid string
	*nats.Conn
	isLive bool
}

func (n *natsWrapper) Request(subj, msgName string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	m, k4stat := topic2Metric(n.gid, subj, "req")
	if m != "" {
		i := strings.LastIndex(msgName, ".")
		if i >= 0 {
			msgName = msgName[i+1:]
		}
		m = m + msgName
		metrics.CountStatistics(m)
		metrics.SizeStatistics(m, len(data))
		n.sizeAlarm(subj, msgName, len(data))

		lastRequestStart := time.Now().UnixNano()
		defer metrics.TimeStatistics(m, k4stat+msgName, lastRequestStart)
	} else {
		n.sizeAlarm(subj, msgName, len(data))
	}
	return n.Conn.Request(subj, data, timeout)
}
func (n *natsWrapper) Publish(subj, msgName string, data []byte) error {
	m, k4stat := topic2Metric(n.gid, subj, "pub")
	if m != "" {
		i := strings.LastIndex(msgName, ".")
		if i >= 0 {
			msgName = msgName[i+1:]
		}
		m = m + msgName
		metrics.CountStatistics(m)
		metrics.SizeStatistics(m, len(data))
		n.sizeAlarm(subj, msgName, len(data))

		lastRequestStart := time.Now().UnixNano()
		defer metrics.TimeStatistics(m, k4stat+msgName, lastRequestStart)
	} else {
		n.sizeAlarm(subj, msgName, len(data))
	}
	return n.Conn.Publish(subj, data)
}
func (n *natsWrapper) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return n.Conn.Subscribe(subj, cb)
}
func (n *natsWrapper) QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return n.Conn.QueueSubscribe(subj, queue, cb)
}

func (n *natsWrapper) Drain() error {
	return n.Conn.Drain()
}

func (n *natsWrapper) Flush() error {
	return n.Conn.Flush()
}

func (n *natsWrapper) Close() {
	n.Conn.Close()
}

func (n *natsWrapper) sizeAlarm(subj, msgName string, dataLen int) {
	if dataLen > msgAlarmLen {
		if n.isLive {
			tilogs.L().Alarm("natsWrapper.Request data too large, subj %s, msgName %s, len(data) %d", subj, msgName, dataLen)
		} else {
			panic(fmt.Sprintf("natsWrapper.Request data too large, subj %s, msgName %s, len(data) %d", subj, msgName, dataLen))
		}
	}
}
