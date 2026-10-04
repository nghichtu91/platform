package ginhelper

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// PlanXController ...
type GinController struct {
}

func (p *GinController) GetString(c *gin.Context, name string) string {
	return c.DefaultQuery(name, "")
}

func (p *GinController) GetInt(c *gin.Context, name string) (int, error) {
	s := c.Query(name)
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

func (p *GinController) GetInt64(c *gin.Context, name string) (int64, error) {
	s := c.Query(name)
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

/*
	1、直接对外提供服务的 Web 应用，在进行与安全有关的操作时，只能通过 Remote Address 获取 IP，不能相信任何请求头；
	2、使用 Nginx 等 Web Server 进行反向代理的 Web 应用，在配置正确的前提下，要用 X-Forwarded-For 最后一节 或 X-Real-IP 来获取 IP（因为 Remote Address 得到的是 Nginx 所在服务器的内网 IP）；同时还应该禁止 Web 应用直接对外提供服务；
	https://imququ.com/post/x-forwarded-for-header-in-http.html
	如果有gin，可以直接调用gin的ClientIP()
*/
func (p *GinController) GetClientIp(c *gin.Context) string {
	return GetClientIp(c.Request)
}

func GetClientIp(req *http.Request) string {
	ips := Proxy(req)
	if len(ips) > 0 && ips[0] != "" {
		rip, _, err := net.SplitHostPort(ips[0])
		if err != nil {
			rip = ips[0]
		}
		return rip
	}
	if ip, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		return ip
	}
	return req.RemoteAddr
}

func Proxy(req *http.Request) []string {
	if ips := req.Header.Get("X-Forwarded-For"); ips != "" {
		return strings.Split(ips, ",")
	}
	return []string{}
}
