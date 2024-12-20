package middlewares

import (
	"sync"
	"tarot/pkg/app"
	"tarot/pkg/limiter"
	"tarot/pkg/logger"
	"tarot/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"golang.org/x/time/rate"
)

const (
	// DefaultBurst 默认突发请求数量
	DefaultBurst = 100
	// DefaultTimeout 默认等待超时时间
	DefaultTimeout = 50 * time.Millisecond
)

var (
	// 用于存储限流器的并发安全缓存
	limiters sync.Map
	// 用于存储上次清理时间的并发安全Map
	lastCleanup sync.Map
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Limit   string
	Burst   int
	Timeout time.Duration
}

// LimitIP 全局限流中间件，针对 IP 进行限流
//
// 支持的限流格式:
// - 5 reqs/second:   "5-S"
// - 10 reqs/minute:  "10-M"
// - 1000 reqs/hour:  "1000-H"
// - 2000 reqs/day:   "2000-D"
//
// 特性:
// - 支持突发流量处理
// - 自动清理过期限流器
// - 高并发安全
// - 优雅降级
func LimitIP(limit string) gin.HandlerFunc {
	// 测试环境使用较大限制
	if app.IsTesting() {
		limit = "1000000-H"
	}

	config := RateLimitConfig{
		Limit:   limit,
		Burst:   DefaultBurst,
		Timeout: DefaultTimeout,
	}

	return createLimiterHandler(func(c *gin.Context) string {
		return limiter.GetKeyIP(c)
	}, config)
}

// LimitPerRoute 针对单个路由的限流中间件
//
// 特性:
// - 基于 IP + 路由路径进行限流
// - 支持动态调整限流策略
// - 自动清理过期数据
func LimitPerRoute(limit string) gin.HandlerFunc {
	if app.IsTesting() {
		limit = "1000000-H"
	}

	config := RateLimitConfig{
		Limit:   limit,
		Burst:   DefaultBurst,
		Timeout: DefaultTimeout,
	}

	return createLimiterHandler(func(c *gin.Context) string {
		return limiter.GetKeyRouteWithIP(c)
	}, config)
}

// createLimiterHandler 创建限流处理器
// keyFunc: 用于生成限流键的函数
// config: 限流配置
func createLimiterHandler(keyFunc func(*gin.Context) string, config RateLimitConfig) gin.HandlerFunc {
	// 定期清理过期的限流器
	go cleanupLimiters()

	return func(c *gin.Context) {
		key := keyFunc(c)

		// 获取或创建限流器
		limiter, err := getLimiter(key, config)
		if err != nil {
			logger.ErrorString("限流器", "创建失败", err.Error())
			// 降级处理：允许请求通过
			c.Next()
			return
		}

		// 尝试获取令牌
		if !limiter.Allow() {
			response.JSON(c, gin.H{
				"code":    429,
				"message": "请求太频繁，请稍后再试",
				"error":   "Too Many Requests",
			})
			c.Abort()
			return
		}

		// 设置 RateLimit 相关响应头
		setRateLimitHeaders(c, limiter)

		c.Next()
	}
}

// getLimiter 获取或创建限流器
func getLimiter(key string, config RateLimitConfig) (*rate.Limiter, error) {
	// 尝试从缓存获取限流器
	if lim, exists := limiters.Load(key); exists {
		return lim.(*rate.Limiter), nil
	}

	// 解析限流配置
	r, err := limiter.ParseLimit(config.Limit)
	if err != nil {
		return nil, err
	}

	// 创建新的限流器
	lim := rate.NewLimiter(rate.Limit(r.Rate), config.Burst)

	// 并发安全地存储限流器
	actual, _ := limiters.LoadOrStore(key, lim)
	return actual.(*rate.Limiter), nil
}

// setRateLimitHeaders 设置限流相关的响应头
func setRateLimitHeaders(c *gin.Context, lim *rate.Limiter) {
	c.Header("X-RateLimit-Limit", cast.ToString(lim.Limit()))
	c.Header("X-RateLimit-Remaining", cast.ToString(lim.Tokens()))
	c.Header("X-RateLimit-Reset", cast.ToString(time.Now().Add(time.Second).Unix()))
}

// cleanupLimiters 定期清理过期的限流器
func cleanupLimiters() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		now := time.Now()
		limiters.Range(func(key, value interface{}) bool {
			lastAccess, _ := lastCleanup.Load(key)
			if lastAccess == nil {
				lastCleanup.Store(key, now)
				return true
			}

			// 清理超过24小时未使用的限流器
			if now.Sub(lastAccess.(time.Time)) > 24*time.Hour {
				limiters.Delete(key)
				lastCleanup.Delete(key)
			}
			return true
		})
	}
}
