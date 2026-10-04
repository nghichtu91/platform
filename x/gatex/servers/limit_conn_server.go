package servers

import (
	"net"
	"runtime/debug"
	"time"

	net2 "github.com/nghichtu91/platform/share/planx/net"
	"github.com/nghichtu91/platform/share/x/gatex/config"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	pkgerrors "github.com/pkg/errors"

	"github.com/nghichtu91/platform/share/planx/util"
)

type LimitConnServer struct {
	ConnServer

	spaces  chan struct{}
	maxconn uint
}

type NewLimitConnServerCfg struct {
	NewConnServerCfg
	MaxConn uint
}

func NewLimitConnServer(cfg NewLimitConnServerCfg, l net.Listener) *LimitConnServer {
	if cfg.MaxConn <= 0 {
		cfg.MaxConn = 3
	}

	s := &LimitConnServer{
		ConnServer: *NewConnServer(cfg.NewConnServerCfg, l),

		maxconn: cfg.MaxConn,
		spaces:  make(chan struct{}, cfg.MaxConn),
	}
	for i := uint(0); i < cfg.MaxConn; i++ {
		s.spaces <- struct{}{}
	}
	return s
}

func (server *LimitConnServer) Listen() error {
	listener, err := newXListener(server.listento, server.GetSslConfigs())
	if err != nil {
		tilogs.L().Errorf("LimitConnServer listen err %s", err.Error())
		return err
	} else {
		server.listener = listener
	}
	return nil
}

func (server *LimitConnServer) Start() {
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

// func (server *LimitConnServer) closer() {
// tilogs.L().Infof("LimitConnServer closer started.")
// for conn := range server.quitingConn {
// conn.Close()
// server.spaces <- true
// }
// tilogs.L().Debugf("closer done")

// }

func (server *LimitConnServer) acceptor() {
	defer func() {
		if err := recover(); err != nil {
			tilogs.L().Errorf("[LimitConnServer] acceptor recover error %v stack:%s", err, string(debug.Stack()))
		}
	}()
	tilogs.L().Infof("LimitConnServer acceptor started.")
	timeoutDuration := time.Millisecond * 5000
	timer := time.NewTimer(timeoutDuration)
	for {
		// server.listener.SetDeadline()
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
		case _, ok := <-server.spaces:
			if ok {
				tilogs.L().Infof("limit conn server accept %s", conn.RemoteAddr())
				server.waitingConn <- conn
			}
		case <-timer.C:
			tilogs.L().Infof("ERR Out of service %s", conn.RemoteAddr())
			conn.Close()
		}
	}
}

func (server *LimitConnServer) GetMaxConn() uint {
	return server.maxconn
}

// Only useful for LimitConn Server. OPTMIZE
func (server *LimitConnServer) ReleaseConnChan(con net.Conn) {
	// server.quitingConn <- con
	server.spaces <- struct{}{}
}
