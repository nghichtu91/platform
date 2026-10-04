package http

import (
	"github.com/gin-gonic/gin"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"

	//conf "github.com/nghichtu91/platform/share/x/chat/logicx/config"
	"net"
	"net/http"

	"github.com/nghichtu91/platform/share/x/chat/logicx/logic"
)

type Router struct {
	engine *gin.Engine
	logic  *logic.Logic
}

// New new a http server.
func New(l *logic.Logic, listener net.Listener, engine *gin.Engine) *Router {
	//engine := gin.New()
	engine.Use(loggerHandler, recoverHandler)

	chanInternalErr := make(chan error, 1)
	go func(c chan error) {
		if err := http.Serve(listener, engine); err != nil {
			log.L().Errorf("http.Serve %s err %s", listener.Addr().String(), err.Error())
			c <- err
		}
	}(chanInternalErr)
	if !util.CheckGoroutineStartErr(chanInternalErr, "http.Serve %s", listener.Addr().String()) {
		return nil
	}

	//r, exitfun := ginhelper.MakeGinEngine("accesslog.xml")

	/*
		go func() {
			if err := engine.Run(conf.Cfg.HttpPort); err != nil {
				panic(err)
			}
		}()
	*/

	s := &Router{
		engine: engine,
		logic:  l,
	}
	s.initRouter()
	return s
}

func (s *Router) initRouter() {
	group := s.engine.Group("/chat")
	group.POST("/forbidden", s.Forbidden)
	group.POST("/unforbidden", s.UnForbidden)
}

// Close close the server.
func (s *Router) Close() {

}
