package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gomodule/redigo/redis"
	"github.com/nghichtu91/platform/share/x/auth/models"
)

// IIPLimiter IP限制接口
type IIPLimiter interface {
	Allow(ip string) bool
}

const (
	limitIPPrefix   = "limit_ip:" // 默认redis key前缀
	defaultLimitSec = 1           // 默认的限制时长
)

var (
	ipLimiter IIPLimiter
)

//  RedisLimiter 使用redis实现的IP限流
type RedisLimiter struct {
	keyPrefix    string // 前缀
	limitTimes   int    // 限制次数，大于此数字限制生效
	limitSeconds int    // 限制持续秒数
}

// InitRedisIPLimiter 使用旧limit参数初始化RedisIP限流
func InitRedisIPLimiter(lTimes int64) {
	rl := NewRedisLimiter("", int(lTimes), defaultLimitSec)
	if rl != nil {
		ipLimiter = rl
	}
}

// SetIPLimiter 设置ipLimiter
func SetIPLimiter(l IIPLimiter) {
	ipLimiter = l
}

// NewRedisLimiter 新建RedisLimiter
// prefix为Redis内Key的前缀，默认为limit:
// lTimes 限制访问次数，大于此数字限制生效
// lSec 限制支持秒数
func NewRedisLimiter(prefix string, lTimes, lSec int) *RedisLimiter {
	if lSec <= 0 || lTimes <= 0 {
		return nil
	}
	if prefix == "" {
		prefix = limitIPPrefix
	}

	return &RedisLimiter{
		keyPrefix:    prefix,
		limitTimes:   lTimes,
		limitSeconds: lSec,
	}
}

// Allow 判断当前key是否允许访问
// 如果redis出现问题，总是返回true，因为这里会返回TooManyRequests，无法反应真实状态
func (l RedisLimiter) Allow(key string) bool {
	conn := models.GetRedisLimitConn()
	defer conn.Close()

	if conn.IsNil() {
		return true
	}

	n, err := redis.Int(conn.Do("auth.limit", "INCR", l.keyPrefix+key))
	if err != nil {
		return true
	}

	if n > l.limitTimes {
		return false
	}

	// 如果之前不存在key，设置TTL
	if n == 1 {
		conn.Do("auth.limit", "EXPIRE", l.keyPrefix+key, l.limitSeconds)
	}

	return true
}

// UserIPLimiter 限制IP中间件
// 无法获取IP时暂不处理
func UserIPLimiter(c *gin.Context) {
	if ipLimiter != nil && !ipLimiter.Allow(c.ClientIP()) {
		c.AbortWithStatusJSON(
			http.StatusTooManyRequests,
			gin.H{"error": "too many requests"},
		)
		return
	}

	c.Next()
}
