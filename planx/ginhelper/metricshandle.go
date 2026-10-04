package ginhelper

import (
	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

func MetricHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		allmetrics.AddAuthAllReqCount()
		c.Next()
	}
}
