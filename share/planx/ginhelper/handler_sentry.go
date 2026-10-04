package ginhelper

import (
	"fmt"
	"net/http"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/gin-gonic/gin"
)

func SentryGinLog() gin.HandlerFunc {
	return sentryRecovery(false)
}

func sentryRecovery(onlyCrashes bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rval := recover(); rval != nil {
				//debug.PrintStack()
				rvalStr := fmt.Sprint(rval)
				tilogs.L().WithUser(c.Request.RequestURI).Errorf("gin panic recover err %s", rvalStr)
				c.Writer.WriteHeader(http.StatusInternalServerError)
			}
			if !onlyCrashes {
				for _, item := range c.Errors {
					tilogs.L().Errorf("gin errors error:%v param:%v", item.Error(), item.Meta)
				}
			}
		}()
		c.Next()
	}
}
