package rpc

import (
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"net"

	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/x/gatex/servers"
)

type GateRPC struct {
	quit chan struct{}
	servers.ConServer
}

func NewGateRPCServer(l net.Listener) *GateRPC {
	scfg := servers.NewConnServerCfg{
		NumberOfAcceptor:     1,
		NumberOfWaitingQueue: 10000,
	}
	return &GateRPC{
		ConServer: servers.NewConnServer(scfg, l),
		quit:      make(chan struct{}),
	}
}

func (g *GateRPC) Start(receiver interface{}) {
	rpc.Register(receiver)

	go g.ConServer.Start()
	var waitGroup util.WaitGroupWrapper
	waitGroup.Wrap(func() {
		for {
			select {
			case <-g.quit:
				tilogs.L().Infof("Gate.rpc.rpcloop, server quit")
				return
			case conn, ok := <-g.GetWaitingConnChan():
				if ok {
					go func() {
						jsonrpc.ServeConn(conn)
						g.ReleaseConnChan(conn)
					}()
				}
			}
		}
	})
	waitGroup.Wait()
	g.ConServer.Stop()
}

func (g *GateRPC) Stop() {
	close(g.quit)
}
