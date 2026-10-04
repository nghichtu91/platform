package http

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
)

func (s *Router) Forbidden(c *gin.Context) {
	var (
		err  error
		time int
	)
	targetId := c.PostForm("target")
	reason := c.PostForm("reason")
	time, err = strconv.Atoi(c.PostForm("endTime"))
	endTime := int64(time)

	log.L().Infof("router forbidden target:%s, reason:%s, time:%s", targetId, reason, time)
	if err = s.logic.PushForbidden(context.TODO(), targetId, reason, endTime); err != nil {
		result(c, nil, RequestErr)
		return
	}
	result(c, nil, OK)
}

func (s *Router) UnForbidden(c *gin.Context) {
	var (
		err error
	)
	targetId := c.PostForm("target")

	log.L().Infof("router unforbidden target:%s", targetId)
	if err = s.logic.PushUnForbidden(context.TODO(), targetId); err != nil {
		result(c, nil, RequestErr)
		return
	}
	result(c, nil, OK)
}
