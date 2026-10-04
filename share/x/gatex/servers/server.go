package servers

import (
	"crypto/tls"
	// "crypto/x509"
	"net"
	// "os"
	// "sync"
	"time"

	net2 "github.com/nghichtu91/platform/share/planx/net"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	pkgerrors "github.com/pkg/errors"

	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/gatex/config"
)

const (
	Number_Of_Acceptor     = 1
	Number_Of_WaitingQueue = 64
)

type ConServer interface {
	Listen() error
	Start()
	Stop()

	GetWaitingConnChan() <-chan net.Conn
	// TODO: zehong, Only useful for LimitConn Server. OPTMIZE
	ReleaseConnChan(net.Conn)
}

type ConnServer struct {
	quit chan struct{}

	waitingConn chan net.Conn
	// quitingConn chan net.Conn

	listento     string
	SslCfg       config.SSLCertCfg
	HostSslCfg   config.HostSSLCertCfg
	listener     net.Listener
	try_listener net.Listener
}

type NewConnServerCfg struct {
	ListenTo             string
	NumberOfAcceptor     uint
	NumberOfWaitingQueue uint
	SslCfg               config.SSLCertCfg
	HostSslCfg           config.HostSSLCertCfg
}

func NewConnServer(cfg NewConnServerCfg, l net.Listener) *ConnServer {
	if cfg.NumberOfAcceptor <= 0 {
		cfg.NumberOfAcceptor = Number_Of_Acceptor
	}
	if cfg.NumberOfWaitingQueue <= 0 {
		cfg.NumberOfWaitingQueue = Number_Of_WaitingQueue
	}
	return &ConnServer{
		quit: make(chan struct{}),

		waitingConn: make(chan net.Conn, cfg.NumberOfWaitingQueue),
		// quitingConn: make(chan net.Conn),
		SslCfg:       cfg.SslCfg,
		HostSslCfg:   cfg.HostSslCfg,
		listento:     cfg.ListenTo,
		try_listener: l,
	}
}

// A listener implements a network listener (net.Listener) for TLS connections.
type xlistener struct {
	net.Listener
	config *tls.Config
}

// Accept waits for and returns the next incoming TLS connection.
// The returned connection c is a *tls.Conn.
func (l *xlistener) Accept() (c net.Conn, err error) {
	c, err = l.Listener.Accept()
	if err != nil {
		return
	}

	if tcpconn, ok := c.(*net.TCPConn); ok {
		tcpconn.SetKeepAlive(true)
		tcpconn.SetKeepAlivePeriod(time.Second * 60)
		tcpconn.SetLinger(-1)
	}

	if l.config != nil {
		c = tls.Server(c, l.config)
	}
	return
}

func newXListener(listento string, sslCfg []config.SSLCertCfg) (net.Listener, error) {
	tilogs.L().Infof("Start Listen %s", listento)
	listener, err := net.Listen("tcp", listento)
	if err != nil {
		return nil, err
	}
	return newXListener_2(listener, sslCfg)
	//return newXListener_l(listener, sslCfg)
}

func newXListener_l(listener net.Listener, sslConfigs []config.SSLCertCfg) (net.Listener, error) {
	xl := &xlistener{Listener: listener}


    // 不再创建 certPool 和 tlsCfg
    for _, sslCfg := range sslConfigs {
        if !sslCfg.IsSet() {
            continue
        }
        tilogs.L().Infof("Listener build without sslcfg %+v", sslCfg)
        // 如果需要保留某些日志，可以在这里输出信息
    }

	// var certPool *x509.CertPool
	// var tlsCfg *tls.Config
	// for _, sslCfg := range sslConfigs {
	// 	if !sslCfg.IsSet() {
	// 		continue
	// 	}
	// 	if tlsCfg == nil {
	// 		tlsCfg = &tls.Config{
	// 			InsecureSkipVerify: true,
	// 			ClientCAs:          certPool,
	// 		}
	// 		xl.config = tlsCfg
	// 	}
	// 	cer, err := tls.LoadX509KeyPair(sslCfg.Cert, sslCfg.Key)
	// 	if err != nil {
	// 		tilogs.L().Errorf("getListener LoadX509KeyPair %+v err:%s", sslCfg, err.Error())
	// 		return nil, err
	// 	}
	// 	tilogs.L().Infof("Listener build with sslcfg %+v", sslCfg)
	// 	tlsCfg.Certificates = append(tlsCfg.Certificates, cer)
	// 	if !sslCfg.HasCA() {
	// 		continue
	// 	}
	// 	if certPool == nil {
	// 		certPool = x509.NewCertPool()
	// 		tlsCfg.ClientCAs = certPool
	// 	}
	// 	certPEMBlock, err := os.ReadFile(sslCfg.CA)
	// 	if err != nil {
	// 		tilogs.L().Errorf("getListener ReadFile %+v err:%s", sslCfg, err.Error())
	// 		return nil, err
	// 	}
	// 	certPool.AppendCertsFromPEM(certPEMBlock)
	// 	tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	// }

	return xl, nil
}

func newXListener_2(listener net.Listener, sslConfigs []config.SSLCertCfg) (net.Listener, error) {
	// 不使用 SSL，直接返回监听器
	xl := &xlistener{Listener: listener}

	// 去掉与 SSL 相关的配置和逻辑
	// 原代码中的 sslConfigs 也不再需要处理

	tilogs.L().Infof("Listener build without SSL configuration")
	return xl, nil
}

func (server *ConnServer) Listen() error {
	var err error
	var listener net.Listener
	allSslCfg := server.GetSslConfigs()
	if server.listento == "" {
		//listener, err = newXListener_l(server.try_listener, allSslCfg)
		listener, err = newXListener_2(server.try_listener, allSslCfg)
		
		tilogs.L().Debugf("ConnServer listen use try listener")
	} else {
		listener, err = newXListener(server.listento, allSslCfg)
		tilogs.L().Debugf("ConnServer listen  %s", server.listento)
	}
	if err != nil {
		tilogs.L().Errorf("ConnServer listen err %s", err.Error())
		return err
	}
	server.listener = listener
	return nil
}

// GetSslConfigs 获取所有的ssl配置
func (server *ConnServer) GetSslConfigs() []config.SSLCertCfg {
	return []config.SSLCertCfg{server.SslCfg, server.HostSslCfg.SSLCertCfg}
}

func (server *ConnServer) Start() {
	var waitGroup util.WaitGroupWrapper
	acceptors := Number_Of_Acceptor
	if acceptors < 1 {
		acceptors = 1
	}
	for i := 0; i < acceptors; i++ {
		waitGroup.Wrap(func() { server.acceptor() })
	}
	// waitGroup.Wrap(func() { server.closer() })
	waitGroup.Wait()
}

func (server *ConnServer) Stop() {
	server.listener.Close()
	// close(server.quitingConn)
	close(server.waitingConn)
	close(server.quit)
}

// func (server *ConnServer) closer() {
// tilogs.L().Infof("ConnServer closer started.")
// for conn := range server.quitingConn {
// conn.Close()
// }
// tilogs.L().Debugf("closer done")
// }

func (server *ConnServer) acceptor() {
	tilogs.L().Infof("ConnServer acceptor start %s", server.listento)
	timeoutDuration := time.Millisecond * 3000
	timer := time.NewTimer(timeoutDuration)
	for {
		conn, err := server.listener.Accept()

		// PS: learn from Serve func in net/http in go source code
		var tempDelay time.Duration // how long to sleep on accept failure
		if err != nil {
			if net2.NetIsClosed(err) {
				tilogs.L().Infof("Accepter quit by NetIsClosed")
				return
			}

			err = pkgerrors.Cause(err)
			ne, ok := err.(net.Error)
			if ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				tilogs.L().Warnf("Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			tempDelay = 0

			if ne.Timeout() {
				continue
			}

			tilogs.L().Errorf(err.Error())
			tilogs.L().Infof("Accepter quit")
			return
		}

		rAddr := conn.RemoteAddr().String()
		if util.NeedFilterElbAddr(rAddr, config.Cfg.GateConfig.ElbAddr) {
			tilogs.L().Debugf("lsb health check, addr:%s", rAddr)
			go serveLSB(conn)
			continue
		}

		for len(timer.C) > 0 {
			<-timer.C
		}
		timer.Reset(timeoutDuration)
		select {
		// XXX 可能会有一点小问题，因为listener是提前close的，所以这里不应该出现写
		case server.waitingConn <- conn:
		case <-timer.C:
			tilogs.L().Infof("ERR Out of service %s", conn.RemoteAddr())
			conn.Close()
		}

	}
}

func (server *ConnServer) GetWaitingConnChan() <-chan net.Conn {
	return server.waitingConn
}

// Only useful for LimitConn Server. OPTMIZE
func (server *ConnServer) ReleaseConnChan(con net.Conn) {
	// server.quitingConn <- con
}
