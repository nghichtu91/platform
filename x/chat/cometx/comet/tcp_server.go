package comet

import (
	"context"
	"io"
	"math"
	"net"
	"sync/atomic"
	"time"

	"github.com/opentracing/opentracing-go"

	"github.com/nghichtu91/platform/share/x/common/consts"

	"github.com/golang/protobuf/proto"

	"github.com/nghichtu91/platform/share/planx/metrics"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/chat/api/const_value"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/cometx/config"
	"github.com/nghichtu91/platform/share/x/chat/pkg/bufio"
	"github.com/nghichtu91/platform/share/x/chat/pkg/bytes"
	xtime "github.com/nghichtu91/platform/share/x/chat/pkg/time"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

// 监听连接
func (s *Server) InitTcp(accept int, publicIp string) {
	var err error
	s.tcpListener, err = net.Listen("tcp", publicIp)
	if err != nil {
		log.L().Errorf("listen failed %s, %v", publicIp, err)
		return
	}

	log.L().Infof("init tcp, listen at %s", s.tcpListener.Addr().String())
	// 按照cpu核数监听
	for i := 0; i < accept; i++ {
		go s.acceptTcp(s.tcpListener)
	}

	s.isOpen = true
}

// 为了减少goroutines的个数 发送使用一个goroutine 接收还是用connection的goroutine
func (s *Server) HandleConnection(conn net.Conn, rp, wp *bytes.Pool, tr *xtime.Timer) {
	var (
		session = NewSession(&config.Cfg)
		err     error
		trd     *xtime.TimerData
		b       *Bucket
		p       *pb.ClientPackage
		rb      = rp.Get()
		wb      = wp.Get()
		rr      = &session.Reader
		wr      = &session.Writer
	)
	log.L().Debugf("handle connection")

	session.Reader.ResetBuffer(conn, rb.Bytes())
	session.Writer.ResetBuffer(conn, wb.Bytes())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 心跳
	trd = tr.Add(time.Second*time.Duration(config.Cfg.Protocol.HandshakeTimeout), func() {
		conn.Close()
		log.L().Warnf("key: %s remoteIP: %s tcp handshake timeout", session.Key, conn.RemoteAddr().String())
	})

	session.IP, _, _ = net.SplitHostPort(conn.RemoteAddr().String())
	// 将session放到桶中
	if p, err = session.CliProto.Set(); err == nil {
		if err = s.authTCP(ctx, session, rr, wr, p); err == nil {
			b = s.Bucket(session.Key)
			err = b.Put(session)
			if config.IsDevelopMode() {
				log.L().Infof("tcp connnected key:%s", session.Key)
			}
		}
	}

	// 握手失败
	if err != nil {
		conn.Close()
		rp.Put(rb)
		wp.Put(wb)
		tr.Del(trd)
		log.L().Warnf("key: %s remoteIP: %s handshake failed error(%v)", session.Key, conn.RemoteAddr().String(), err)
		return
	}

	var wg util.WaitGroupWrapper
	// 发送单开一个goroutine
	wg.Wrap(func() {
		s.SendLoop(conn, session, wp, wb)
	})

	// 接受直接使用connect的这个goroutine
	s.ReceiveLoop(ctx, session, b, trd, tr)

	// close
	b.Del(session)
	tr.Del(trd)
	rp.Put(rb)
	conn.Close()
	session.Close()
	if err = s.Disconnect(ctx, session.Key, session.SessionId); err != nil {
		log.L().Errorf("key: %s operator do disconnect error(%v)", session.Key, err)
	}

	wg.Wait()
	s.ConnNumLock.Lock()
	s.ConnNum--
	log.L().Infof("player exit. session %s, ccu:%d", session.Key, s.ConnNum)
	s.ConnNumLock.Unlock()

	if session.Key != consts.RobotCometxID {
		allmetrics.ReduceCometxCCU()
	}
}

func (s *Server) acceptTcp(listener net.Listener) {
	var (
		conn net.Conn
		err  error
		r    int
	)

	for {
		if conn, err = listener.Accept(); err != nil {
			// if listener close then return
			log.L().Errorf("listener.Accept(\"%s\") error(%v)", listener.Addr().String(), err)
			return
		}

		// ip addr
		lAddr := conn.LocalAddr().String()
		rAddr := conn.RemoteAddr().String()
		if const_value.IsIgnoreHealthIp(rAddr) {
			log.L().Debugf("lsb health check, addr:%s", rAddr)
			go serveLSB(conn)
			continue
		}

		log.L().Infof("start tcp serve \"%s\" with \"%s\"", lAddr, rAddr)
		go serveTCP(s, conn, r)
		if r++; r == math.MaxInt32 {
			r = 0
		}

		/*
			if err = conn.SetDeadline(time.Now().Add(time.Second * time.Duration(config.Cfg.TCP.DeadLine))); err != nil {
				log.L().Errorf("conn.SetKeepAlive() error(%v)", err)
				return
			}
		*/
	}
}

// 处理负载均衡的健康检查
func serveLSB(conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(
		time.Second * time.Duration(config.Cfg.TCP.LSBDeadLine)))

	buf := make([]byte, 4)
	n, err := io.ReadFull(conn, buf)
	log.L().Debugf("serve lsb connection, n:%d, err:%v", n, err)
}

func serveTCP(s *Server, conn net.Conn, r int) {
	var (
		// round分配buffer
		tr = s.round.Timer(r)
		rp = s.round.Reader(r)
		wp = s.round.Writer(r)
	)

	s.HandleConnection(conn, rp, wp, tr)
}

func (s *Server) ReceiveLoop(ctx context.Context, session *Session, b *Bucket, trd *xtime.TimerData, tr *xtime.Timer) {
	var (
		err      error
		p        *pb.ClientPackage
		needGrow bool
		hb       time.Duration
	)

	trd.Key = session.Key
	hb = time.Second * time.Duration(config.Cfg.Protocol.HandshakeTimeout)
	tr.Set(trd, hb)

	log.L().Debugf("enter receive loop")
	for {
		needGrow = true
		if p, err = session.CliProto.Set(); err != nil {
			log.L().Warnf("ReceiveLoop get a proto failed error(%v), key:%s", err, session.Key)
			break
		}
		if err = p.ReadTCP(&session.Reader); err != nil {
			log.L().Warnf("ReceiveLoop read tcp failed error(%v), key:%s", err, session.Key)
			break
		}

		metrics.ReqCountAndSizeStatistics(allmetrics.CometPrefix, pb.ChatOperation_name[int32(p.GetMessageId())], len(p.GetRawData()))
		if p.GetMessageId() == uint32(pb.ChatOperation_C2SHeartbeat) {
			tr.Set(trd, hb)
			*p.MessageId = uint32(pb.ChatOperation_S2CHeartbeat)
			p.RawData = p.RawData[:0]
			log.L().Debugf("tcp heartbeat receive key:%s", session.Key)
		} else {
			// goim 很多情况就是直接断开 而不是返回错误
			// break
			allmetrics.CometCountAddNewRequest(1)
			needGrow = s.Operate(ctx, p, session, b)
		}

		// 有些消息是需要直接返回给客户端的
		// 有些消息是经由logic job返回回来的
		// 需要返回的消息 直接使用之前的读出来的buf返回
		if needGrow {
			session.CliProto.SetAdv()
			// 所有消息都是由 send loop发送
			session.Signal()
		}
	}
	log.L().Infof("comet receive loop end, key:%s", session.Key)
}

// SendLoop 发送循环状态
func (s *Server) SendLoop(conn net.Conn, session *Session, wp *bytes.Pool, wb *bytes.Buffer) {
	var (
		err    error
		finish bool
	)
	log.L().Debugf("enter send loop")

	for {
		var p = session.Ready()
		log.L().Debugf("key %s dispatch msg:%d %v", session.Key, p.GetMessageId(), p)
		switch p {
		case pb.ProtoFinish:
			log.L().Infof("key %s wakeup exit dispatch goroutine", session.Key)
			finish = true
			goto failed
		case pb.ProtoReady:
			// 这里和goim 有差别的是 goim聊天的消息都是 通过http直接发送到logic的
			// 我们是所有都从comet来的
			// 这里会把所有的消息都消费掉
			var clientPackage *pb.ClientPackage
			for {
				if clientPackage, err = session.CliProto.Get(); err != nil {
					break
				}
				if err = clientPackage.WriteTCP(&session.Writer); err != nil {
					goto failed
				}

				// 改为通过判断长度复用slice，避免频繁gc
				if len(clientPackage.RawData) > int(pb.MaxBodySize) {
					clientPackage.RawData = make([]byte, 0, int(pb.MaxBodySize))
				} else {
					clientPackage.RawData = clientPackage.RawData[:0]
				}

				session.CliProto.GetAdv()
			}
		default:
			var span opentracing.Span
			if p.Span != nil {
				span = opentracing.GlobalTracer().StartSpan("GameChatMsg.comet.send", opentracing.ChildOf(p.Span.Context()))
			}

			// 这里的发送是job发送来的消费消息
			if err = p.WriteTCP(&session.Writer); err != nil {
				goto failed
			}
			if span != nil {
				span.Finish()
			}
			log.L().Debugf("tcp sent a message key:%s message id:%d", session.Key, p.GetMessageId())
		}
		// only hungry flush response
		if err = session.Writer.Flush(); err != nil {
			break
		}
		allmetrics.CometCountAddNewResponse(1)
	}
failed:
	if err != nil {
		log.L().Warnf("key: %s dispatch tcp error(%v)", session.Key, err)
	}
	conn.Close()
	wp.Put(wb)
	// must ensure all channel message discard, for reader won't blocking Signal
	for !finish {
		finish = session.Ready() == pb.ProtoFinish
	}
	log.L().Infof("comet send loop end, key:%s", session.Key)
}

// etcd's interface implement
// 获得连接数
func (s *Server) GetConnNum() int32 {
	s.ConnNumLock.RLock()
	defer s.ConnNumLock.RUnlock()

	return s.ConnNum
}

// etcd's interface implement
// 服务是否开启
func (s *Server) IsOpen() bool {
	return s.isOpen
}

// auth for goim handshake with client, use rsa & aes.
func (s *Server) authTCP(ctx context.Context, session *Session, rr *bufio.Reader, wr *bufio.Writer, p *pb.ClientPackage) (err error) {
	var (
		reason  string
		endTime int64
		key     string
	)
	for {
		if err = p.ReadTCP(rr); err != nil {
			return
		}

		// 第一个包必须是登陆包
		if p.GetMessageId() == uint32(pb.ChatOperation_C2SLogin) {
			break
		} else {
			log.L().Warnf("tcp request operation(%d) not auth", p.GetMessageId())
		}
	}

	loginMsg := &pb.ChatLogin{}
	err = proto.Unmarshal(p.RawData, loginMsg)
	// 构建返回包
	*p.MessageId = uint32(pb.ChatOperation_S2CLogin)

	if err != nil {
		log.L().Warnf("login package unmarshal failed player id is %s", loginMsg.GetPlayerId())
		return
	}

	if key, reason, endTime, err = s.Connect(ctx, loginMsg.GetPlayerId(), loginMsg.GetToken(), session.SessionId); err != nil {
		log.L().Warnf("authTCP.Connect(key:%s).err(%v)", loginMsg.GetPlayerId(), err)
		return
	}

	session.Key = key
	if endTime != 0 {
		atomic.StoreInt64(&session.ForbiddenEndTime, endTime)
	}

	// 将resp写到原client package中
	loginResp := &pb.ChatLoginResp{
		Code:    pb.ErrorCode_Success.Enum(),
		Reason:  &reason,
		EndTime: &endTime,
	}
	buf, err := proto.Marshal(loginResp)

	if cap(p.RawData) >= len(buf) {
		p.RawData = p.RawData[0:len(buf)]
	} else {
		p.RawData = make([]byte, len(buf))
	}
	copy(p.RawData, buf)

	// p.RawData = p.RawData[:0]
	// p.RawData = append(p.RawData, buf...)

	if err = p.WriteTCP(wr); err != nil {
		log.L().Errorf("authTCP.WriteTCP(key:%v).err(%v)", key, err)
		return
	}
	err = wr.Flush()
	s.ConnNumLock.Lock()
	s.ConnNum++
	log.L().Infof("auth competed!, key:%s, ccu:%d", key, s.ConnNum)
	s.ConnNumLock.Unlock()

	if loginMsg.GetPlayerId() != consts.RobotCometxID {
		allmetrics.AddCometxCCU()
	}

	return
}
