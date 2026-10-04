package limit

import (
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/ginhelper"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/mailgun/timetools"
	"github.com/vulcand/oxy/utils"
	"github.com/nghichtu91/platform/share/planx/iputil"
	"github.com/nghichtu91/platform/share/planx/limit/ratelimit"

	"github.com/gin-gonic/gin"
)

var (
	internalUrlRegex *regexp.Regexp
)

func InitMatchLimitedUrlRegex() (err error) {
	internalUrlRegex, err = regexp.Compile(LimitCfg.LimitUrlRegex)
	return
}

func CheckIdentity(specHeader, specContent string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if internalUrlRegex == nil {
			c.JSON(http.StatusUnauthorized, "IT1")
			c.Abort()
			return
		}
		ok := internalUrlRegex.MatchString(c.Request.URL.Path)
		tilogs.L().Debugf("CheckIdentity Regex: %s flag: %t", internalUrlRegex,ok)
		// if ok {
		// 	spec := c.Request.Header.Get(specHeader)
		// 	if spec == "" || spec != specContent {
		// 		c.JSON(http.StatusUnauthorized, "-_-!")
		// 		c.Abort()
		// 		return
		// 	}
		// }
		c.Next()
	}
}

func isLimitedIP(ip string, ip_cfg []internalIP) bool {
	IpOnly := ip
	if strings.Contains(ip, ":") {
		ipOnly, _, err := net.SplitHostPort(ip)
		if err != nil {
			tilogs.L().Errorf("isLimitedIP err %v", err)
			return false
		}
		IpOnly = ipOnly
	}

	var limited bool
	ipOnly := net.ParseIP(IpOnly)
	for _, v := range ip_cfg {
		limited = limited || iputil.IpBetween(v.From, v.To, ipOnly)
		if limited {
			return true
		}
	}
	return false
}

func isInternalIp(ip string, ip_cfg []internalIP) bool {
	IpOnly := ip
	if strings.Contains(ip, ":") {
		ipOnly, _, err := net.SplitHostPort(ip)
		if err != nil {
			return false
		}
		IpOnly = ipOnly
	}
	//检查是否是内部IP
	var isInternal bool
	isInternal = (IpOnly == "127.0.0.1")
	if isInternal {
		return true
	}
	return isLimitedIP(ip, ip_cfg)
}

func RateLimit(c *gin.Context) {
	if LimitCfg.RateLimitValid { // 功能开启
		// url是否需要检查
		if internalUrlRegex == nil {
			c.JSON(http.StatusUnauthorized, "IT2")
			c.Abort()
			return
		}
		ok := internalUrlRegex.MatchString(c.Request.URL.Path)
		if ok {
			//limited url
			if err := rateLimit.Consume(c.Request); err != nil {
				tilogs.L().Errorf("[limit] ratelimit ip: %s", c.ClientIP())
				c.JSON(429, "Too Many Requests")
				c.Abort()
				return
			}
		} else {
			//url is not in limit, etc, /api
			//所有内部API都不做限制
		}
	}
	c.Next()
}

var (
	rateLimit *ratelimit.TokenLimiter
)

func Init() bool {
	err := InitMatchLimitedUrlRegex()
	if err != nil {
		tilogs.L().Errorf("init ratelimite url regex err: " + err.Error())
		return false
	}

	if LimitCfg.RateLimitValid {
		rates := ratelimit.NewRateSet()
		rates.Add(time.Second, LimitCfg.RateLimitAverage, LimitCfg.RateLimitBurst)

		extr := utils.ExtractorFunc(extractClientIP)
		l, err := ratelimit.NewForHttp(
			nil,
			extr,
			rates,
			ratelimit.Clock(&timetools.RealTime{}))
		if err != nil {
			tilogs.L().Errorf("init ratelimite new limiter err: " + err.Error())
			return false
		}
		rateLimit = l
	}
	return true
}

func extractClientIP(req *http.Request) (string, int64, error) {
	ip := ginhelper.GetClientIp(req)
	return ip, 1, nil
}
