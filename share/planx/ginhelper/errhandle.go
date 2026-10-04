package ginhelper

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func ErrRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rval := recover(); rval != nil {
				tilogs.L().Errorf("Gin Panic recovery -> %v", rval)
				c.Writer.WriteHeader(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
